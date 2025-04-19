package slogger

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Init golang slog package init
func Init(opts ...Option) {
	opt := Options{
		level:        slog.LevelInfo, // default:info level
		replaceAttr:  nil,
		outputToFile: false,
	}

	for _, o := range opts {
		o(&opt)
	}

	var w io.Writer
	if opt.outputToFile {
		if opt.fileMaxSize == 0 {
			opt.fileMaxSize = 100
		}

		if opt.fileName == "" {
			opt.fileName = "app.log"
		}

		if opt.fileMaxBackups == 0 {
			opt.fileMaxBackups = 10
		}

		if opt.fileMaxAge == 0 {
			opt.fileMaxAge = 30
		}

		// set lumberjack
		w = &lumberjack.Logger{
			Filename:   opt.fileName,
			MaxSize:    opt.fileMaxSize,
			MaxBackups: opt.fileMaxBackups,
			MaxAge:     opt.fileMaxAge,
			Compress:   opt.compress,
		}
	} else {
		w = os.Stdout
	}

	var handler slog.Handler
	handlerOption := &slog.HandlerOptions{
		Level:       opt.level,
		AddSource:   opt.addSource,
		ReplaceAttr: opt.replaceAttr,
	}
	if opt.json {
		handler = slog.NewJSONHandler(w, handlerOption)
	} else {
		handler = slog.NewTextHandler(w, handlerOption)
	}

	// set global Logger
	slog.SetDefault(slog.New(handler))
}

// With returns a Logger that includes the given attributes
// in each output operation. Arguments are converted to
// attributes as if by slog [Logger.Log].
func With(args ...any) *slog.Logger {
	return slog.Default().With(args...)
}
