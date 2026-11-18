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
	// set if RotateOnStart was used, as the default is to rotate only
	// an explicit rotator
	rotateOnStartSet bool
	slogLevel        slog.Level
	format           string
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

// SetLogfile sets the log file, which is overridden by the environment
// variable CORDIAL_LOG_OUTPUT. If no writer is given using [SetWriter]
// then the log is written using a [timberjack.Logger] with its default
// settings, which does not remove old backups of rotated files. If a
// writer was given with [SetWriter] then its Filename is replaced by
// logfile.
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

// RotateOnStart sets whether an existing log file is rotated when
// logging is started by [Init], whether or not the file name comes
// from [SetLogfile] or from the writer given to [SetWriter]. If it is
// not used then a writer given to [SetWriter] is rotated and a log
// file from [SetLogfile] alone is not.
func RotateOnStart(rotate bool) LoggerOption {
	return func(lo *loggerOpts) {
		lo.rotateOnStart = rotate
		lo.rotateOnStartSet = true
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
