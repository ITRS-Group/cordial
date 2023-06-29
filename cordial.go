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
	_ "embed" // embed the VERSION in the top-level package
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// VERSION is a semi-global string variable
//
//go:embed VERSION
var VERSION string

// LogInit is called to set-up zerolog the way we like it
func LogInit(prefix string) {
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		fnName := "UNKNOWN"
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			fnName = fn.Name()
		}
		fnName = filepath.Base(fnName)
		// fnName = strings.TrimPrefix(fnName, "main.")

		s := strings.SplitAfterN(file, prefix+"/", 2)
		if len(s) == 2 {
			file = s[1]
		}
		return fmt.Sprintf("%s:%d %s()", file, line, fnName)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339, NoColor: true,
		FormatLevel: func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("%s:", i))
		},
	}).With().Caller().Logger()
}

func renderMD(in string) (md string) {
	tr, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
		glamour.WithEnvironmentConfig(),
		glamour.WithStylesFromJSONBytes([]byte(`{ "document": { "margin": 0 } }`)),
	)
	if err != nil {
		return in
	}
	md, err = tr.Render(in)
	if err != nil {
		return in
	}
	return
}

// RenderHelpAsMD updated the given command to use glamour to render the
// command's Long description as markdown formatted text to an ANSI
// terminal.
func RenderHelpAsMD(command *cobra.Command) {
	// render help with glamour
	cobra.AddTemplateFunc("md", renderMD)
	command.SetHelpTemplate(`{{with (or .Long .Short)}}{{. | md | trimTrailingWhitespaces}}

	{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}
`)
}
