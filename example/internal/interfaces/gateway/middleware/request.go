package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"

	"github.com/daheige/hephfx/ctxkeys"
	"github.com/daheige/hephfx/gutils"
	"github.com/daheige/hephfx/logger"
)

// LogWare 日志中间件
type LogWare struct{}

// Access 记录访问日志
func (ware *LogWare) Access() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// uri := c.Request.RequestURI
		// 性能分析后发现 log.Println 输出需要分配大量的内存空间,而且每次写入都需要加锁处理
		// log.Println("request before")
		// log.Println("request uri: ", uri)

		// 如果采用了nginx x-request-id功能，可以获得x-request-id
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			// logId = gutils.RndUuid() // 日志id
			requestID = gutils.Uuid()
		}

		// 设置跟请求相关的ctx信息
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, ctxkeys.XRequestID, requestID)
		ctx = context.WithValue(ctx, ctxkeys.ClientIP, c.ClientIP())
		ctx = context.WithValue(ctx, ctxkeys.RequestURI, c.Request.RequestURI)
		ctx = context.WithValue(ctx, ctxkeys.UserAgent, c.GetHeader("User-Agent"))
		ctx = context.WithValue(ctx, ctxkeys.RequestMethod, c.Request.Method)
		c.Request = c.Request.WithContext(ctx)

		// 记录请求日志
		logger.Info(ctx, "exec begin", nil)

		c.Next()

		// 请求结束记录日志
		fields := map[string]interface{}{
			"exec_time": fmt.Sprintf("%.4f", time.Now().Sub(start).Seconds()),
		}
		if code := c.Writer.Status(); code != 200 {
			fields["response_code"] = code
		}

		logger.Info(c.Request.Context(), "exec end", c)
	}
}

// Recover gin请求处理中遇到异常或panic捕获
func (ware *LogWare) Recover() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// log.Printf("error:%v", err)
				c := ctx.Request.Context()
				logger.Warn(c, "exec panic", map[string]interface{}{
					"trace_error": fmt.Sprintf("%v", err),
					"full_stack":  string(gutils.CatchStack()),
				})

				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						errMsg := strings.ToLower(se.Error())
						// 记录操作日志
						logger.Warn(c, "os syscall error", map[string]interface{}{
							"trace_error": errMsg,
						})

						if strings.Contains(errMsg, "broken pipe") ||
							strings.Contains(errMsg, "reset by peer") ||
							strings.Contains(errMsg, "request headers: small read buffer") ||
							strings.Contains(errMsg, "unexpected EOF") ||
							strings.Contains(errMsg, "i/o timeout") {
							brokenPipe = true
						}
					}
				}

				// 是否是 brokenPipe类型的错误，如果是，这里就不能往写入流中再写入内容
				// 如果是该类型的错误，就不需要返回任何数据给客户端
				// 代码参考gin recovery.go RecoveryWithWriter方法实现
				// If the connection is dead, we can't write a status to it.
				if brokenPipe {
					// ctx.Error(err.(error)) // nolint: errcheck
					ctx.AbortWithStatus(http.StatusInternalServerError)
					return
				}

				// 响应状态
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
					"code":    http.StatusInternalServerError,
					"message": "server inner error",
				})
			}
		}()

		ctx.Next()
	}
}

// NotFoundHandler router not found.
func NotFoundHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(404, gin.H{
			"code":    codes.NotFound,
			"message": "this page not found",
			"details": []struct{}{},
		})
	}
}
