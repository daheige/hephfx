package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/daheige/hephfx/example/internal/interfaces/gateway/middleware"
	"github.com/daheige/hephfx/example/pb"
	"github.com/daheige/hephfx/logger"
	"github.com/daheige/hephfx/monitor"
)

func main() {
	// 初始化日志配置
	logger.Default(
		logger.WithStdout(true),
		logger.WithJsonFormat(true),
		logger.WriteToFile(false),
	)

	// prometheus metrics和pprof接入
	var grpcGatewayPort uint16 = 8091
	// 初始化prometheus和pprof，可以根据实际情况更改
	// metrics访问地址：http://localhost:9091/metrics
	// pprof访问地址：http://localhost:9091/debug/pprof
	// 具体指标参考：hephfx/monitor/monitor.go
	monitor.InitMonitor(grpcGatewayPort+1000, true)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// 这个grpc微服务地址，一般来说是一个远程的ip:port，可以根据实际情况更改
	gRPCAddress := fmt.Sprintf("0.0.0.0:%d", 50051)
	err := pb.RegisterGreeterHandlerFromEndpoint(ctx, mux, gRPCAddress, opts)
	if err != nil {
		log.Fatal(ctx, "failed to register grpc endpoint", map[string]interface{}{
			"trace_error": err.Error(),
		})
	}

	// 初始化http router
	router := gin.New()
	initRouter(router, mux)

	// 服务server设置
	address := fmt.Sprintf("0.0.0.0:%d", grpcGatewayPort)
	server := &http.Server{
		Handler:           router,
		Addr:              address,
		IdleTimeout:       20 * time.Second, // tcp idle time
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
	}

	// 在独立携程中运行
	log.Println("server run on:", address)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Warn(context.Background(), "server exec panic", "trace_error",
					fmt.Sprintf("%v", r), slog.String("full_stack", string(debug.Stack())),
				)
			}
		}()

		if err2 := server.ListenAndServe(); err2 != nil {
			if !errors.Is(err2, http.ErrServerClosed) {
				logger.Error(context.Background(), "server close error", map[string]interface{}{
					"trace_error": err2.Error(),
				})

				log.Println("server close error:", err2)
				return
			}

			log.Println("server will exit...")
		}
	}()

	// server平滑重启
	ch := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// receive signal to exit main goroutine.
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	sig := <-ch

	log.Println("exit signal: ", sig.String())
	// Create a deadline to wait for.
	ctx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// if your application should wait for other services
	// to finalize based on context cancellation.
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)

		_ = server.Shutdown(ctx)
	}()

	<-done
	<-ctx.Done()

	log.Println("server shutting down")
}

func initRouter(router *gin.Engine, mux *runtime.ServeMux) {
	// 访问日志中间件处理
	logWare := &middleware.LogWare{}

	// 对所有的请求进行性能监控，一般来说生产环境，可以对指定的接口做性能监控
	router.Use(logWare.Access(), logWare.Recover(), middleware.TimeoutHandler(10*time.Second))

	// gin 框架prometheus接入
	router.Use(middleware.WrapMonitor())

	// 路由找不到的情况
	router.NoRoute(middleware.NotFoundHandler())

	// gateway http proxy
	// 这里将proto文件中的路由地址进行路由注册
	// 访问方式：http://localhost:8091/v1/say/1
	router.Any("/v1/*any", gin.WrapH(mux))

	// 添加自定义路由地址
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
}
