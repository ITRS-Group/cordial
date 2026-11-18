// Package runas lets the documentation generators run a copy of
// themselves under another executable name.
//
// The tools derive their name, and so the names in their default file
// names and help text, from the name of the running executable. To get
// documentation that matches a tool, the generator has to run with the
// name of that tool.
package runas

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/itrs-group/cordial"
)

// Run runs a copy of the current executable, named name, in the
// current directory with the given arguments and with the same standard
// input, output and error. The copy is removed afterwards.
func Run(name string, args ...string) (err error) {
	dir, err := os.MkdirTemp("", "cordial-docs-")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)

	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	exe := filepath.Join(dir, name)
	if err = linkOrCopy(cordial.ExecutablePath(), exe); err != nil {
		return
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func linkOrCopy(src, dst string) (err error) {
	if err = os.Link(src, dst); err == nil {
		return
	}

	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return
	}
	return out.Close()
}
