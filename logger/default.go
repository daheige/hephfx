package logger

import (
	"context"
	"runtime/debug"

	"go.uber.org/zap"

	"github.com/daheige/hephfx/ctxkeys"
)

var (
	// logEntry default logger entry.
	logEntry Logger

	// defaultLogDir default log dir.
	defaultLogDir = "./logs"
)

// Default 初始化默认zap logger接口
// 默认日志写到终端中
func Default(opts ...Option) {
	options := []Option{
		WithJsonFormat(true),        // 默认json格式化输出
		WithCallerSkip(2),           // 如果基于这个Logger包，需要设置适当的skip
		WithLogLevel(zap.InfoLevel), // 设置日志打印最低级别,如果不设置,默认为info级别
		WithStdout(true),            // 日志默认输出到终端

		// WithLogDir(defaultLogDir),       // 日志目录
		// WithLogFilename(defaultLogFile), // 日志文件名，默认zap.log
		// WithMaxAge(3),                   // 最大保存3天
		// WithMaxSize(200),                // 每个日志文件最大200MB
		// WithWriteToFile(true),           // 默认开启日志写入文件中
	}

	if len(opts) > 0 {
		options = append(options, opts...)
	}

	logEntry = New(options...)
}

// Debug debug级别日志
func Debug(ctx context.Context, msg string, fields ...interface{}) {
	logEntry.Debug(ctx, msg, fields...)
}

// Info info级别日志
func Info(ctx context.Context, msg string, fields ...interface{}) {
	logEntry.Info(ctx, msg, fields...)
}

// Error 错误类型的日志
func Error(ctx context.Context, msg string, fields ...interface{}) {
	logEntry.Error(ctx, msg, fields...)
}

// Warn 警告类型的日志
func Warn(ctx context.Context, msg string, fields ...interface{}) {
	logEntry.Warn(ctx, msg, fields...)
}

// DPanic 调试模式下的panic，程序不退出，继续运行
func DPanic(ctx context.Context, msg string, fields ...interface{}) {
	logEntry.DPanic(ctx, msg, fields...)
}

// Recover 用来捕获程序运行出现的panic信息，并记录到日志中
// 这个panic信息，将采用 DPanic 方法进行记录
func Recover(ctx context.Context, msg string, fields ...interface{}) {
	if err := recover(); err != nil {
		if len(fields) == 0 {
			fields = make([]interface{}, 0, 2)
		}

		fields = append(fields, ctxkeys.Fullstack.String(), string(debug.Stack()))
		logEntry.DPanic(ctx, msg, fields...)
	}
}

// Panic 抛出panic的时候，先记录日志，然后执行panic,退出当前goroutine
// 如果没有捕获，就会退出当前程序,建议程序做defer捕获处理
func Panic(ctx context.Context, msg string, fields ...interface{}) {
	logEntry.Panic(ctx, msg, fields...)
}

// Fatal 抛出致命错误，然后退出程序
func Fatal(ctx context.Context, msg string, fields ...interface{}) {
	logEntry.Fatal(ctx, msg, fields...)
}
