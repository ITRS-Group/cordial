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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"src.elv.sh/pkg/md"
)

// VERSION is a semi-global string variable
//
//go:embed VERSION
var VERSION string

// RenderHelpAsMD sets the help template of the given cobra.Command to
// render the long description and short description as Markdown, using
// the src.elv.sh/pkg/md package.
func RenderHelpAsMD(command *cobra.Command) {
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

	// strip any case-insensitive "exe" extension from the binary, to
	// allow windows .EXE binary to work.
	if strings.EqualFold(path.Ext(execname), ".exe") {
		execname = strings.TrimSuffix(execname, path.Ext(execname))
	}

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
	out = md.RenderString(in, &md.TTYCodec{Width: 72})
	return
}
