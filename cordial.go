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
	"sync"

	"github.com/spf13/cobra"
	"src.elv.sh/pkg/md"
)

// VERSION is the Cordial version embedded in the binary at build time.
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

var (
	// cached absolute path to the executable, with symlinks resolved
	executablePath = sync.OnceValue(resolveExecutablePath)

	execMutex      sync.Mutex
	executableName string // cached processed executable name, with symlinks resolved and version suffix removed if present
)

// resolveExecutablePath returns the absolute path to the running
// executable with symlinks resolved, falling back to the unresolved path
// from os.Executable and then to os.Args[0] if either step fails.
func resolveExecutablePath() string {
	execpath, _ := os.Executable()
	if resolved, err := filepath.EvalSymlinks(execpath); err == nil && resolved != "" {
		return resolved
	}
	if execpath == "" && len(os.Args) > 0 {
		return os.Args[0]
	}
	return execpath
}

// executableBaseName processes the executable path to a name, as
// described for [ExecutableName].
func executableBaseName(execpath, version string) string {
	execname := path.Base(filepath.ToSlash(execpath))

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
	if version != "" {
		execname = strings.TrimSuffix(execname, "-"+version)
	}
	return execname
}

// ExecutableName returns a processed executable name. Symlinks are
// resolved and the basename of the resulting file has, at most, one
// extension removed and then if there is a '-' followed by the matching
// cordial version string. Note this final version string must match
// first version argument or the cordial one compiled into the binary.
// The result is cached by the first call and is safe for concurrent
// use.
//
// For example:
//
//	`geneos.exe` -> `geneos`
//	`dv2email-v1.10.0` -> `dv2email`
func ExecutableName(version ...string) string {
	execMutex.Lock()
	defer execMutex.Unlock()

	if executableName != "" {
		return executableName
	}

	v := VERSION
	if len(version) > 0 {
		v = version[0]
	}
	execname := executableBaseName(executablePath(), v)
	if execname == "" || execname == "." || execname == "/" {
		// last resort, do not cache a meaningless name
		if len(os.Args) > 0 {
			return executableBaseName(os.Args[0], v)
		}
		return execname
	}

	executableName = execname
	return execname
}

// ExecutablePath returns the absolute path to the executable, with all
// symlinks resolved. The result is cached after the first call to avoid
// repeated filesystem lookups and is safe for concurrent use.
func ExecutablePath() string {
	return executablePath()
}

func renderMD(in string) (out string) {
	out = md.RenderString(in, &md.FmtCodec{Width: 76})
	return
}
