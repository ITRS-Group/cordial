package logger

import (
	"io"
	"log/slog"
	"os"
)

type handlerOpts struct {
	delimiter  string
	prefix     string
	level      *slog.LevelVar
	w          io.Writer
	timeFormat string
	json       bool
}

type Option func(*handlerOpts)

// JSON configures the handler to output JSON instead of a
// human-readable format.
func JSON() Option {
	return func(opts *handlerOpts) {
		opts.json = true
	}
}

// Delimiter sets the delimiter between attributes in the output. The default is a single dot.
func Delimiter(delimiter string) Option {
	return func(opts *handlerOpts) {
		opts.delimiter = delimiter
	}
}

// SourceTrimTo sets the directory in the source path to trim up to,
// including a tailing '/'. e.g. if the source path is
// "/home/user/project/pkg/file.go" and the anchor is "project", the
// resulting source path in the log will be "pkg/file.go".
func SourceTrimTo(anchor string) Option {
	return func(opts *handlerOpts) {
		opts.prefix = anchor
	}
}

// Leveler sets the log level for the handler using a slog.Leveler.
// This allows the log level to be changed at runtime. The default is
// slog.LevelInfo.
func Leveler(level *slog.LevelVar) Option {
	return func(opts *handlerOpts) {
		opts.level = level
	}
}

// Writer sets the output writer for the handler. The default is
// os.Stderr.
func Writer(w io.Writer) Option {
	return func(opts *handlerOpts) {
		opts.w = w
	}
}

// TimeFormat sets the time format for time attributes. The default is
// "2006-01-02T15:04:05.000Z07:00". The format should be a valid Go time
// format string.
func TimeFormat(format string) Option {
	return func(opts *handlerOpts) {
		opts.timeFormat = format
	}
}

func evalOpts(options ...Option) *handlerOpts {
	opts := &handlerOpts{
		w:          os.Stderr,
		level:      &slog.LevelVar{},
		timeFormat: "2006-01-02T15:04:05.000Z07:00",
	}
	for _, o := range options {
		o(opts)
	}
	return opts
}
