package interceptor

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/daheige/hephfx/ctxkeys"
	"github.com/daheige/hephfx/gutils"
	"github.com/daheige/hephfx/logger"
	"github.com/daheige/hephfx/micro"
)

// AccessLog 记录访问日志的拦截器
func AccessLog(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (res interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			// the error format defined by grpc must be used here to return code and desc
			err = status.Errorf(codes.Internal, "%s", "server inner error")
			logger.Info(ctx, "exec panic", map[string]interface{}{
				"reply":       res,
				"trace_error": fmt.Sprintf("%v", r),
				"full_stack":  string(debug.Stack()),
			})
		}
	}()

	t := time.Now()
	clientIP, _ := micro.GetGRPCClientIP(ctx)

	// log.Printf("client_ip: %s\n", clientIP)
	// log.Printf("request: %v\n", req)

	// x-request-id
	var requestId string
	if logID := ctx.Value(ctxkeys.XRequestID.String()); logID == nil {
		requestId = gutils.Uuid()
	} else {
		requestId, _ = logID.(string)
	}

	ctx = context.WithValue(ctx, ctxkeys.XRequestID, requestId)
	ctx = context.WithValue(ctx, ctxkeys.ClientIP, clientIP)
	ctx = context.WithValue(ctx, ctxkeys.RequestMethod, info.FullMethod)
	// ctx = context.WithValue(ctx, logger.RequestURI, info.FullMethod)
	ctx = context.WithValue(ctx, ctxkeys.UserAgent, "grpc-client")
	logger.Info(ctx, "exec begin", map[string]interface{}{
		"client_ip": clientIP,
	})

	res, err = handler(ctx, req)
	execTime := fmt.Sprintf("%.4f", time.Since(t).Seconds())
	if err != nil {
		logger.Error(ctx, "exec error", map[string]interface{}{
			"trace_error": err.Error(),
			"exec_time":   execTime,
			"reply":       res,
		})

		return
	}

	logger.Info(ctx, "exec end", map[string]interface{}{
		"exec_time": execTime,
	})

	return
}
