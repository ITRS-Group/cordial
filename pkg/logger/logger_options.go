package logger

import (
	"log/slog"

	"github.com/DeRuina/timberjack"
	"github.com/itrs-group/cordial/pkg/config"
)

type loggerOpts struct {
	logfile       string
	lj            *timberjack.Logger
	rotateOnStart bool
	slogLevel     slog.Level
	format        string
}

type LoggerOption func(*loggerOpts)

func evalLoggerOptions(options ...LoggerOption) *loggerOpts {
	opts := &loggerOpts{
		rotateOnStart: true,
	}
	for _, opt := range options {
		opt(opts)
	}

	return opts
}

func SetLogfile(logfile string) LoggerOption {
	return func(lo *loggerOpts) {
		lo.logfile = config.ResolveHome(logfile)
	}
}

// LogRotateOptions set the log writer to the configured
// lumberjack/timeberjack settings but only if the lj.Filename field is
// not empty, otherwise it is ignored.
func LogRotateOptions(lj *timberjack.Logger) LoggerOption {
	return func(lo *loggerOpts) {
		if lj.Filename != "" {
			lj.Filename = config.ResolveHome(lj.Filename)
			lo.lj = lj
		}
	}
}

func RotateOnStart(rotate bool) LoggerOption {
	return func(lo *loggerOpts) {
		lo.rotateOnStart = rotate
	}
}

// SetLogLevel takes a slog debug level to use as a default
func SetLogLevel(level slog.Level) LoggerOption {
	return func(lo *loggerOpts) {
		lo.slogLevel = level
	}
}

func Format(format string) LoggerOption {
	return func(lo *loggerOpts) {
		lo.format = format
	}
}
