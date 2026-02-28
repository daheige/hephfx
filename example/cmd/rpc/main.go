package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/daheige/hephfx/example/internal/interfaces/rpc/interceptor"
	"github.com/daheige/hephfx/example/pb"
	"github.com/daheige/hephfx/logger"
	"github.com/daheige/hephfx/micro"
	"github.com/daheige/hephfx/monitor"
)

func main() {
	// 初始化日志配置
	logger.Default(
		logger.WithStdout(true),
		logger.WithJsonFormat(true),
		logger.WriteToFile(false),
	)

	grpcPort := 50051

	// 健康检查路由地址，可以根据实际情况添加
	routerOpts := micro.WithRoutes(micro.Route{
		Method: "GET",
		Path:   "/healthz",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("request path:", r.URL.Path)
			b, _ := json.Marshal(map[string]interface{}{
				"code":    0,
				"message": "Ok",
				"active":  true,
				"time":    time.Now().Format("2006-01-02 15:04:05"),
			})

			w.Write(b)
		},
	})

	// 创建grpc微服务实例
	s := micro.NewService(
		fmt.Sprintf("0.0.0.0:%d", grpcPort),

		// start grpc and http gateway use one address
		micro.WithEnableGRPCShareAddress(),

		// micro.WithGRPCHTTPAddress(fmt.Sprintf("0.0.0.0:%d", 8080)),
		micro.WithHandlerFromEndpoints(pb.RegisterGreeterHandlerFromEndpoint), // register http endpoint

		micro.WithLogger(micro.LoggerFunc(log.Printf)),
		micro.WithShutdownTimeout(5*time.Second),
		micro.WithEnablePrometheus(), // prometheus interceptor

		micro.WithEnableRequestValidator(), // request validator interceptor
		// 使用自定义请求拦截器
		micro.WithUnaryInterceptor(interceptor.AccessLog),
		micro.WithShutdownFunc(func() {
			time.Sleep(3 * time.Second) // mock long operations
			log.Println("grpc server shutdown")
		}),
		routerOpts,
	)

	// 初始化prometheus和pprof，可以根据实际情况更改
	// metrics访问地址：http://localhost:8090/metrics
	// pprof访问地址：http://localhost:8090/debug/pprof
	// 在metrics接口可以搜索grpc_开头的指标，表示当前微服务接口运行情况
	monitor.InitMonitor(8090)

	// 初始化greeter service
	service := &GreeterServer{}

	// 注册grpc微服务
	pb.RegisterGreeterServer(s.GRPCServer, service)

	// 运行服务
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

// GreeterServer 实现greeter微服务
type GreeterServer struct {
	// 这里必须包含这个解构体才可以，否则就是没有实现
	pb.UnimplementedGreeterServer
}

// SayHello 实现say hello方法
func (s *GreeterServer) SayHello(ctx context.Context, req *pb.HelloReq) (*pb.HelloReply, error) {
	reply := &pb.HelloReply{
		Message: fmt.Sprintf("hello,%s", req.Name),
	}

	return reply, nil
}
