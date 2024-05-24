/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package cordial is a collection of tools, packages and integrations for
// Geneos written primarily in Go
package cordial

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type discardCloser struct {
	io.Writer
}

func (discardCloser) Close() error { return nil }

// LogInit is called to set-up zerolog with our chosen defaults. The
// default is to log to STDERR.
//
// If logfile is passed and the first element is not empty, then use
// that as the log file unless it is either "-" (which means use STDOUT
// (not STDERR) or equal to the [os.DevNull] value, in which case is
// [io.Discard].
func LogInit(prefix string, options ...LogOptions) {
	opts := evalLoggerOptions(options...)
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		if zerolog.GlobalLevel() > zerolog.DebugLevel {
			return ""
		}
		fnName := "UNKNOWN"
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			fnName = fn.Name()
		}
		fnName = path.Base(fnName)
		// fnName = strings.TrimPrefix(fnName, "main.")

		s := strings.SplitAfterN(file, prefix+"/", 2)
		if len(s) == 2 {
			file = s[1]
		}
		return fmt.Sprintf("%s:%d %s()", file, line, fnName)
	}

	var nocolor bool
	var out io.WriteCloser
	out = os.Stderr

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
		if opts.lj == nil {
			opts.lj = &lumberjack.Logger{}
		}
		opts.lj.Filename = opts.logfile

		out = &lumberjack.Logger{Filename: opts.logfile}
		nocolor = true
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        out,
		TimeFormat: time.RFC3339,
		NoColor:    nocolor,
		FormatLevel: func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("%s:", i))
		},
		FormatMessage: func(i interface{}) string {
			return fmt.Sprintf("%s: %s", prefix, i)
		},
	}).With().Caller().Logger()
}

type logOpts struct {
	logfile       string
	lj            *lumberjack.Logger
	rotateOnStart bool
}

type LogOptions func(*logOpts)

func evalLoggerOptions(options ...LogOptions) *logOpts {
	opts := &logOpts{
		rotateOnStart: true,
	}
	for _, opt := range options {
		opt(opts)
	}
	return opts
}

func SetLogfile(logfile string) LogOptions {
	return func(lo *logOpts) {
		lo.logfile = logfile
	}
}

func LumberjackOptions(lj *lumberjack.Logger) LogOptions {
	return func(lo *logOpts) {
		lo.lj = lj
	}
}

func RotateOnStart(rotate bool) LogOptions {
	return func(lo *logOpts) {
		lo.rotateOnStart = rotate
	}
}
