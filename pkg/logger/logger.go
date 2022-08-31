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
package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime" // placeholder
	"strings"
	"time"
)

type GeneosLogger struct {
	Writer     io.Writer
	Level      Level
	ShowPrefix bool
}

// debuglog must be defined so it can be set in EnableDebugLog()
// so for consistency do the same for all three loggers
var (
	Logger      = GeneosLogger{os.Stdout, INFO, false}
	DebugLogger = GeneosLogger{os.Stderr, DEBUG, true}
	ErrorLogger = GeneosLogger{os.Stderr, ERROR, true}

	Log   = log.New(Logger, "", 0)
	Debug = log.New(DebugLogger, "", 0)
	Error = log.New(ErrorLogger, "", 0)
)

type Level int

const (
	INFO Level = iota
	DEBUG
	WARNING
	ERROR
	FATAL
)

func (level Level) String() string {
	switch level {
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	case ERROR:
		return "ERROR"
	case WARNING:
		return "WARNING"
	default:
		return "UNKNOWN"
	}
}

func init() {
	DisableDebugLog()
}

func refresh() {
	Log = log.New(Logger, "", 0)
	Debug = log.New(DebugLogger, "", 0)
	Error = log.New(ErrorLogger, "", 0)
}

func EnableDebugLog() {
	Debug.SetOutput(DebugLogger)
}

func DisableDebugLog() {
	Debug.SetOutput(ioutil.Discard)
}

func (g *GeneosLogger) EnablePrefix() {
	g.ShowPrefix = true
	refresh()
}

func (g *GeneosLogger) DisablePrefix() {
	g.ShowPrefix = false
	refresh()
}

func (g GeneosLogger) Write(p []byte) (n int, err error) {
	var prefix string
	if g.ShowPrefix {
		prefix = fmt.Sprintf("%s %s: ", time.Now().Format(time.RFC3339), g.Level)
	}

	var line string
	switch g.Level {
	case FATAL:
		line = fmt.Sprintf("%s%s", prefix, p)
		io.WriteString(g.Writer, line)
		os.Exit(1)
	case ERROR:
		var fnName string = "UNKNOWN"
		pc, _, ln, ok := runtime.Caller(3)
		if ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				fnName = fn.Name()
			}
		}
		fnName = filepath.Base(fnName)
		fnName = strings.TrimPrefix(fnName, "main.")

		line = fmt.Sprintf("%s%s():%d %s", prefix, fnName, ln, p)
	case DEBUG:
		var fnName string = "UNKNOWN"
		pc, f, ln, ok := runtime.Caller(3)
		if ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				fnName = fn.Name()
			}
		}
		fnName = strings.TrimPrefix(fnName, "main.")

		// filename is either relative (-trimpath) or the basename with a ./ prefix
		// this lets VSCode make the location clickable
		if filepath.IsAbs(f) {
			f = "./" + filepath.Base(f)
		}
		line = fmt.Sprintf("%s%s() %s:%d %s", prefix, fnName, f, ln, p)
	default:
		line = fmt.Sprintf("%s%s", prefix, p)
	}
	return io.WriteString(g.Writer, line)
}
