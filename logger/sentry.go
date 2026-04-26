package logger

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

type customSentryCore struct {
	level  zapcore.Level
	hub    *sentry.Hub
	fields []zapcore.Field
	// 异步上报日志间隔，建议设置3s
	flushTimeout time.Duration
}

func newSentryCore(level zapcore.Level, hub *sentry.Hub, flushTimeout time.Duration) zapcore.Core {
	c := &customSentryCore{
		fields:       make([]zapcore.Field, 0, 20),
		flushTimeout: flushTimeout,
		level:        level,
		hub:          hub,
	}
	if c.flushTimeout == 0 {
		c.flushTimeout = 3 * time.Second
	}

	return c
}

func (c *customSentryCore) Enabled(level zapcore.Level) bool {
	// log.Println("log level:", level)
	// log.Println("cur report level:", c.level)
	return level >= c.level
}

func (c *customSentryCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	clone.fields = append(clone.fields[:len(c.fields):len(c.fields)], fields...)
	// clone.fields = make([]zapcore.Field, len(c.fields))
	// copy(clone.fields, c.fields)
	// clone.fields = append(clone.fields, fields...)
	return &clone
}

func (c *customSentryCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}

	return ce
}

// 将zap level转换为 sentry level
func (c *customSentryCore) sentryLevel(level zapcore.Level) sentry.Level {
	switch level {
	case zapcore.DebugLevel:
		return sentry.LevelDebug
	case zapcore.InfoLevel:
		return sentry.LevelInfo
	case zapcore.WarnLevel:
		return sentry.LevelWarning
	case zapcore.ErrorLevel:
		return sentry.LevelError
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return sentry.LevelFatal
	default:
	}

	return sentry.LevelError
}

// Write 实现 write 方法
func (c *customSentryCore) Write(e zapcore.Entry, fields []zapcore.Field) error {
	// 合并字段
	allFields := append(c.fields, fields...)

	// 添加所有字段
	m := make(sentry.Context, len(allFields))
	for _, f := range allFields {
		m[f.Key] = FieldToValue(f)
	}

	// 使用 WithScope 创建临时 Scope 隔离，避免污染全局 Scope
	sentry.WithScope(func(scope *sentry.Scope) {
		level := c.sentryLevel(e.Level)
		scope.SetContext("logs_fields", m)
		scope.SetLevel(level)

		// 创建event
		event := sentry.NewEvent()
		event.Level = level
		event.Message = e.Message
		event.Logger = e.LoggerName
		event.Timestamp = e.Time

		// 日志上报
		c.hub.CaptureEvent(event)
	})

	return nil
}

// Sync 实现 sync
// Flush 会一直等待底层的传输将所有缓冲的事件发送到 Sentry 服务器，最多等待给定的超时时间。
// 如果达到超时时间，它将返回 false。在这种情况下，可能会有部分事件未被发送。
// 在终止程序之前应调用 Flush，以避免意外丢失事件。
// 不要在每次调用 CaptureEvent、CaptureException 或 CaptureMessage 之后随意调用 Flush
func (c *customSentryCore) Sync() error {
	c.hub.Flush(c.flushTimeout)
	return nil
}
