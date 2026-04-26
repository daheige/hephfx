package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"

	"github.com/daheige/hephfx/logger"
)

func main() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: os.Getenv("SENTRY_DSN"),
	})
	if err != nil {
		log.Fatalf("sentry.Init: %v", err)
	}

	defer sentry.Flush(2 * time.Second)
	// sentry.CaptureMessage("It works!")

	logger.Default(
		logger.WithJsonFormat(true),        // 默认json格式化输出
		logger.WithCallerSkip(2),           // 如果基于这个Logger包，需要设置适当的skip
		logger.WithLogLevel(zap.InfoLevel), // 设置日志打印最低级别,如果不设置,默认为info级别
		logger.WithStdout(true),            // 日志默认输出到终端

		logger.WithEnableSentry(true),          // 开启sentry上报
		logger.WithSentryLevel(zap.ErrorLevel), // 只允许错误级别以上的日志上报
	)

	logger.Info(context.Background(), "hello world", "plat", "mac")
	logger.Error(context.Background(), "exec begin", "foo", "abc")
	logger.DPanic(context.Background(), "exec dpanic", "foo", "abc")
	logger.Error(context.Background(), "auth error", "uid", 1)
}
