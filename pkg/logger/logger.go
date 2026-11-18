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
	"sync"

	"github.com/DeRuina/timberjack"
	"github.com/fatih/color"
	"golang.org/x/term"
)

// package globals available to callers
var (
	LogLevel   slog.LevelVar
	Logger     *slog.Logger = slog.Default()
	LogHandler slog.Handler
)

type discardCloser struct {
	io.Writer
}

// Close TODO: Document its behavior, inputs, and effects.
func (discardCloser) Close() error { return nil }

// Init is called to set-up logging with our chosen defaults. The
// default is to log to STDERR.
//
// If logfile is passed and the first element is not empty, then use
// that as the log file unless it is either "-" (which means use STDOUT
// (not STDERR) or equal to the [os.DevNull] value, in which case is
// [io.Discard].
//
// The environment variable `CORDIAL_LOG_OUTPUT` can be used to override
// the logfile option. If `CORDIAL_LOG_OUTPUT` is set, then the
// [logger.SetLogfile] option is ignored.
//
// The environment variable `CORDIAL_LOG_FORMAT` can be used to set the
// log format. If `CORDIAL_LOG_FORMAT` is set to "json", then the log
// format is JSON, otherwise the format is a human readable format, with
// colouring enabled if the output is a termial according to
// `term.IsTerminal()`. The global setting in [color.NoColor] is not
// changed.
//
// Init replaces the contents of the [Logger] pointer so that packages
// that have copied it, or that use [slog.Default], log through the new
// handler, which is not safe while other goroutines are logging. Call
// Init once, early in the program and before starting any, and do not
// call it again while they are running. Concurrent calls to Init are
// serialised.
func Init(prefix string, options ...LoggerOption) *slog.Logger {
	opts := evalLoggerOptions(options...)

	if os.Getenv("CORDIAL_LOG_OUTPUT") != "" {
		opts.logfile = os.Getenv("CORDIAL_LOG_OUTPUT")
	}

	// rotate is true if the rotator, if there is one, is to be rotated
	// before use
	rotate := opts.rotateOnStart

	switch opts.logfile {
	case "":
		if opts.rotator != nil {
			if w, ok := opts.rotator.(io.Writer); ok {
				opts.w = w
			}
		} else {
			opts.w = os.Stderr
		}
	case "-":
		opts.w = os.Stdout
	case os.DevNull:
		opts.w = discardCloser{io.Discard}
	default:
		// if given a filename, use the provided or default
		// lumberjack/timeberjack but override the filename
		switch r := opts.rotator.(type) {
		case nil:
			opts.rotator = &timberjack.Logger{
				Filename: opts.logfile,
			}
			rotate = opts.rotateOnStartSet && opts.rotateOnStart
		case *timberjack.Logger:
			r.Filename = opts.logfile
		}
		if w, ok := opts.rotator.(io.Writer); ok {
			opts.w = w
		}
	}

	if rotate && opts.rotator != nil && opts.logfile != "-" && opts.logfile != os.DevNull {
		if r, ok := opts.rotator.(interface{ Rotate() error }); ok {
			_ = r.Rotate()
		}
	}

	handlerOptions := []HandlerOption{
		Leveler(&LogLevel),
		TrimSourcePathTo(prefix),
		Delimiter("."),
		Writer(opts.w),
		// decide on colour here, for this writer only, instead of
		// changing the global setting for all other output
		Color(writerIsColour(opts.w)),
	}

	if opts.format == "json" || os.Getenv("CORDIAL_LOG_FORMAT") == "json" {
		handlerOptions = append(handlerOptions, JSON())
	}

	initMutex.Lock()
	defer initMutex.Unlock()

	LogHandler = NewHandler(handlerOptions...)

	// set up slog
	LogLevel.Set(opts.slogLevel)

	// update the pointer
	*Logger = *slog.New(LogHandler)

	return Logger
}

// initMutex serialises calls to [Init]
var initMutex sync.Mutex

// writerIsColour returns true if w is a terminal and colour has not
// been disabled globally, e.g. using NO_COLOR
func writerIsColour(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return !color.NoColor && term.IsTerminal(int(f.Fd()))
	}
	return false
}
