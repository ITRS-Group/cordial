package logger

import (
	"io"
	"log/slog"
	"os"

	"github.com/DeRuina/timberjack"
	"github.com/itrs-group/cordial/pkg/config"
)

type loggerOpts struct {
	w             io.Writer
	logfile       string
	rotator       any
	rotateOnStart bool
	slogLevel     slog.Level
	format        string
}

// LoggerOption TODO: Describe its purpose and intended use.
type LoggerOption func(*loggerOpts)

func evalLoggerOptions(options ...LoggerOption) *loggerOpts {
	opts := &loggerOpts{
		w:             os.Stderr,
		rotateOnStart: true,
	}
	for _, opt := range options {
		opt(opts)
	}

	return opts
}

// SetLogfile TODO: Document its behavior, inputs, and effects.
func SetLogfile(logfile string) LoggerOption {
	return func(lo *loggerOpts) {
		lo.logfile = config.ResolveHome(logfile)
	}
}

// SetWriter set the log writer to the configured lumberjack/timeberjack
// settings but only if the lj.Filename field is not empty, otherwise it
// is ignored.
func SetWriter(w io.Writer) LoggerOption {
	return func(lo *loggerOpts) {
		switch v := w.(type) {
		case *timberjack.Logger:
			if v.Filename != "" {
				v.Filename = config.ResolveHome(v.Filename)
				lo.rotator = v
			}
		}
	}
}

// RotateOnStart TODO: Document its behavior, inputs, and effects.
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

// Format TODO: Document its behavior, inputs, and effects.
func Format(format string) LoggerOption {
	return func(lo *loggerOpts) {
		lo.format = format
	}
}
