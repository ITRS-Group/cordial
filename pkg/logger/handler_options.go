package logger

import (
	"io"
	"log/slog"
	"os"
)

type handlerOpts struct {
	delimiter string
	prefix    string
	level     *slog.LevelVar
	w         io.Writer
}

type Option func(*handlerOpts)

func WithDelimiter(delimiter string) Option {
	return func(opts *handlerOpts) {
		opts.delimiter = delimiter
	}
}

func SourceRoot(prefix string) Option {
	return func(opts *handlerOpts) {
		opts.prefix = prefix
	}
}

func WithLevel(level slog.Level) Option {
	return func(opts *handlerOpts) {
		opts.level.Set(level)
	}
}

func WithLevelVar(level *slog.LevelVar) Option {
	return func(opts *handlerOpts) {
		opts.level = level
	}
}

func Writer(w io.Writer) Option {
	return func(opts *handlerOpts) {
		opts.w = w
	}
}

func evalOpts(options ...Option) *handlerOpts {
	opts := &handlerOpts{
		w:     os.Stderr,
		level: &slog.LevelVar{},
	}
	for _, o := range options {
		o(opts)
	}
	return opts
}
