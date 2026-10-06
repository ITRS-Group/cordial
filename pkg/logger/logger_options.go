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
