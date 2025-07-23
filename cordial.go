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
	_ "embed" // embed the VERSION in the top-level package
	"html"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// VERSION is a semi-global string variable
//
//go:embed VERSION
var VERSION string

// RenderHelpAsMD updated the given command to use glamour to render the
// command's Long description as markdown formatted text to an ANSI
// terminal.
func RenderHelpAsMD(command *cobra.Command) {
	// render help with glamour
	cobra.AddTemplateFunc("md", renderMD)
	// start on first line, one line gap, usage without leading spaces
	command.SetHelpTemplate(`{{with (or .Long .Short)}}{{. | md}}{{end}}
{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}
`)
}

var executableName string

// ExecutableName returns a processed executable name. Symlinks are
// resolved and the basename of the resulting file has, at most, one
// extension removed and then if there is a '-' followed by the matching
// cordial version string. Note this final version string must match
// first version argument or the cordial one compiled into the binary.
//
// For example:
//
//	`geneos.exe` -> `geneos`
//	`dv2email-v1.10.0` -> `dv2email`
func ExecutableName(version ...string) (execname string) {
	if executableName != "" {
		return executableName
	}

	execname, _ = os.Executable()
	execname, _ = filepath.EvalSymlinks(execname)
	execname = path.Base(filepath.ToSlash(execname))

	// strip any extension from the binary, to allow windows .EXE
	// binary to work. Note we get the extension first, it may be
	// capitalised. This will also remove any other extensions, users
	// should use '-' or '_' instead.
	execname = strings.TrimSuffix(execname, path.Ext(execname))

	// finally strip the VERSION, if found, prefixed by a dash, on the
	// end of the basename
	//
	// this way you can run a versioned binary and still see the right
	// config files
	if len(version) > 0 {
		execname = strings.TrimSuffix(execname, "-"+version[0])
	} else {
		execname = strings.TrimSuffix(execname, "-"+VERSION)
	}

	// cache
	executableName = execname
	return
}

func renderMD(in string) (out string) {
	var width int = 72
	var err error

	style := glamour.WithAutoStyle()
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		style = glamour.WithStandardStyle("ascii")
	} else {
		width, _, err = term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			width = 72
		}
		if width > 96 {
			width = 96
		}
	}

	// Bug https://github.com/charmbracelet/glamour/issues/407 with word-wrapping, awaiting fix
	tr, err := glamour.NewTermRenderer(
		style,
		glamour.WithStylesFromJSONBytes([]byte(`{ "document": { "margin": 2 } }`)),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return in
	}
	out, err = tr.Render(in)
	if err != nil {
		return in
	}
	out = html.UnescapeString(out)
	return
}
