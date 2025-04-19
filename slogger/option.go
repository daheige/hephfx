package slogger

import (
	"log/slog"
)

// Options slog config
type Options struct {
	json bool // true => output JSON; false => output text default:false

	outputToFile bool   // default:false
	fileName     string // default: app.log
	// fileMaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	fileMaxSize int

	// fileMaxBackups is the maximum number of old log files to retain.The default
	// is to retain 10 old log files (though MaxAge may still cause them to get
	// deleted.)
	fileMaxBackups int

	// fileMaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is 30 days.
	fileMaxAge int
	compress   bool // file compress default:false

	level slog.Level // log level,default:slog.LevelInfo

	// addSource causes the handler to compute the source code position
	// of the log statement and add a source key attribute to the output.
	// default:false
	addSource   bool
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
}

// Option slog config option
type Option func(*Options)

// WithEnableJSON log json output
func WithEnableJSON() Option {
	return func(o *Options) {
		o.json = true
	}
}

// WithOutputToFile set outputToFile
func WithOutputToFile(outputFile bool) Option {
	return func(o *Options) {
		o.outputToFile = outputFile
	}
}

// WithFileName set log filename
func WithFileName(fileName string) Option {
	return func(o *Options) {
		o.fileName = fileName
	}
}

// WithFileMaxSize set log filename max size,eg:100
func WithFileMaxSize(fileMaxSize int) Option {
	return func(o *Options) {
		o.fileMaxSize = fileMaxSize
	}
}

// WithFileMaxBackups set log file max backups
func WithFileMaxBackups(fileMaxBackups int) Option {
	return func(o *Options) {
		o.fileMaxBackups = fileMaxBackups
	}
}

// WithFileMaxAge set log file max age，eg:30 days
func WithFileMaxAge(fileMaxAge int) Option {
	return func(o *Options) {
		o.fileMaxAge = fileMaxAge
	}
}

// WithCompress set log file compress
func WithCompress(compress bool) Option {
	return func(o *Options) {
		o.compress = compress
	}
}

// WithLevel set log level
func WithLevel(level slog.Level) Option {
	return func(o *Options) {
		o.level = level
	}
}

// WithAddSource set log record location
func WithAddSource(addSource bool) Option {
	return func(o *Options) {
		o.addSource = addSource
	}
}

// WithReplaceAttr set slog replaceAttr
func WithReplaceAttr(r func(groups []string, a slog.Attr) slog.Attr) Option {
	return func(o *Options) {
		if o.replaceAttr == nil {
			return
		}

		o.replaceAttr = r
	}
}
