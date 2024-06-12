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

	"github.com/itrs-group/cordial/pkg/config"
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
			if i == nil {
				return fmt.Sprintf("%s:", prefix)
			}
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
		lo.logfile = config.ExpandHome(logfile)
	}
}

func LumberjackOptions(lj *lumberjack.Logger) LogOptions {
	return func(lo *logOpts) {
		if lj.Filename != "" {
			lj.Filename = config.ExpandHome(lj.Filename)
		}
		lo.lj = lj
	}
}

func RotateOnStart(rotate bool) LogOptions {
	return func(lo *logOpts) {
		lo.rotateOnStart = rotate
	}
}
