/*
Copyright © 2022 ITRS Group

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

// Package cordial is a collection of tools, packages and integrations for
// Geneos written primarily in Go
package cordial

import (
	"io"
	"log/slog"
	"os"

	"github.com/fatih/color"
	"golang.org/x/term"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

var (
	LogLevel   slog.LevelVar
	Logger     *slog.Logger = slog.Default()
	LogHandler slog.Handler
)

type discardCloser struct {
	io.Writer
}

func (discardCloser) Close() error { return nil }

// LogInit is called to set-up logging with our chosen defaults. The
// default is to log to STDERR.
//
// If logfile is passed and the first element is not empty, then use
// that as the log file unless it is either "-" (which means use STDOUT
// (not STDERR) or equal to the [os.DevNull] value, in which case is
// [io.Discard].
func LogInit(prefix string, options ...LogOption) *slog.Logger {
	var out io.WriteCloser
	out = os.Stderr

	opts := evalLoggerOptions(options...)

	switch opts.logfile {
	case "":
		if opts.lj != nil {
			if opts.rotateOnStart {
				opts.lj.Rotate()
			}
			out = opts.lj
		} else {
			out = os.Stderr
		}
	case "-":
		out = os.Stdout
	case os.DevNull:
		out = discardCloser{io.Discard}
	default:
		// if given a filename, use the default lumberjack but override
		// the filename
		if opts.lj == nil {
			opts.lj = &lumberjack.Logger{}
		}
		opts.lj.Filename = opts.logfile

		out = opts.lj
	}

	LogHandler = logger.NewHandler(
		logger.Leveler(&LogLevel),
		logger.SourceAnchor("cordial"),
		logger.Delimiter("."),
		logger.Writer(out),
		// logger.JSON(),
	)

	// set up slog
	LogLevel.Set(opts.slogLevel)
	Logger = slog.New(LogHandler)

	if _, ok := out.(*lumberjack.Logger); ok {
		color.NoColor = true
	} else if o, ok := out.(*os.File); ok {
		if !term.IsTerminal(int(o.Fd())) {
			color.NoColor = true
		}
	}

	return Logger
}

type logOpts struct {
	logfile       string
	lj            *lumberjack.Logger
	rotateOnStart bool
	slogLevel     slog.Level
}

type LogOption func(*logOpts)

func evalLoggerOptions(options ...LogOption) *logOpts {
	opts := &logOpts{
		rotateOnStart: true,
	}
	for _, opt := range options {
		opt(opts)
	}

	return opts
}

func SetLogfile(logfile string) LogOption {
	return func(lo *logOpts) {
		lo.logfile = config.ResolveHome(logfile)
	}
}

// LumberJackOptions set the log writer to the configured lumberjack
// settings but only if the lj.Filename field is not empty, otherwise it
// is ignored.
func LumberjackOptions(lj *lumberjack.Logger) LogOption {
	return func(lo *logOpts) {
		if lj.Filename != "" {
			lj.Filename = config.ResolveHome(lj.Filename)
			lo.lj = lj
		}
	}
}

func RotateOnStart(rotate bool) LogOption {
	return func(lo *logOpts) {
		lo.rotateOnStart = rotate
	}
}

// SetLogLevel takes a slog debug level to use as a default
func SetLogLevel(level slog.Level) LogOption {
	return func(lo *logOpts) {
		lo.slogLevel = level
	}
}
