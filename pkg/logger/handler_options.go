/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

type HandlerOption func(*handlerOpts)

func evalHandlerOpts(options ...HandlerOption) *handlerOpts {
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

// JSON configures the handler to output JSON instead of a
// human-readable format.
func JSON() HandlerOption {
	return func(opts *handlerOpts) {
		opts.json = true
	}
}

// Delimiter sets the delimiter between attributes in the output. The default is a single dot.
func Delimiter(delimiter string) HandlerOption {
	return func(opts *handlerOpts) {
		opts.delimiter = delimiter
	}
}

// TrimSourcePathTo sets the anchor in the source path to trim up to,
// automatically including a trailing '/'. e.g. if the source path is
// "/home/user/project/pkg/file.go" and the anchor is "project", the
// resulting source path in the log will be "pkg/file.go".
func TrimSourcePathTo(anchor string) HandlerOption {
	return func(opts *handlerOpts) {
		opts.prefix = anchor
	}
}

// Leveler sets the log level for the handler using a slog.Leveler.
// This allows the log level to be changed at runtime. The default is
// a pointer to [slog.LevelInfo].
func Leveler(level *slog.LevelVar) HandlerOption {
	return func(opts *handlerOpts) {
		opts.level = level
	}
}

// Writer sets the output writer for the handler. The default is
// os.Stderr.
func Writer(w io.Writer) HandlerOption {
	return func(opts *handlerOpts) {
		opts.w = w
	}
}

// TimeFormat sets the time format for time attributes. The default is
// "2006-01-02T15:04:05.000Z07:00". The format should be a valid Go time
// format string.
func TimeFormat(format string) HandlerOption {
	return func(opts *handlerOpts) {
		opts.timeFormat = format
	}
}
