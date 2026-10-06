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

	"github.com/DeRuina/timberjack"
	"github.com/fatih/color"
	"golang.org/x/term"
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
// `term.IsTerminal()`.
func Init(prefix string, options ...LoggerOption) *slog.Logger {
	var out io.WriteCloser
	out = os.Stderr

	opts := evalLoggerOptions(options...)

	if os.Getenv("CORDIAL_LOG_OUTPUT") != "" {
		opts.logfile = os.Getenv("CORDIAL_LOG_OUTPUT")
	}

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
		// if given a filename, use the provided or default
		// lumberjack/timeberjack but override the filename
		if opts.lj == nil {
			opts.lj = &timberjack.Logger{}
		}
		opts.lj.Filename = opts.logfile

		out = opts.lj
	}

	loggerOptions := []HandlerOption{
		Leveler(&LogLevel),
		SourceTrimTo(prefix),
		Delimiter("."),
		Writer(out),
	}

	if opts.format == "json" || os.Getenv("CORDIAL_LOG_FORMAT") == "json" {
		loggerOptions = append(loggerOptions, JSON())
	}

	LogHandler = NewHandler(loggerOptions...)

	// set up slog
	LogLevel.Set(opts.slogLevel)
	// update the point
	*Logger = *slog.New(LogHandler)

	switch o := out.(type) {
	case *timberjack.Logger:
		color.NoColor = true
	case *os.File:
		if !term.IsTerminal(int(o.Fd())) {
			color.NoColor = true
		}
	}

	return Logger
}
