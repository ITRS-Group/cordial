/*
Copyright © 2023 ITRS Group

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

package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Daemon backgrounds the current process by re-executing the existing
// binary (as found by [os.Executable], so may there is a small window
// while the referenced binary can change). The function passed as
// processArgs is called with any further arguments passed to it as
// parameters and can be used to remove flags that triggered the
// daemonisation in the first place. A helper function - [RemoveArgs] -
// is available to do this.
//
// If successful the function never returns and the child process PID is
// written to writepid, which can be io.Discard if not required. On
// failure the function does return with an error.
//
//	process.Daemon(os.Stdout, process.RemoveArgs, "-D", "--daemon")
func Daemon(writepid io.Writer, processArgs func([]string, ...string) []string, args ...string) (err error) {
	bin, err := os.Executable()
	if err != nil {
		return
	}
	var newargs []string
	if processArgs == nil {
		newargs = RemoveArgs(os.Args[1:], args...)
	} else {
		newargs = processArgs(os.Args[1:], args...)
	}
	cmd := exec.Command(bin, newargs...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// OS specific (compile time/build constraint) change to cmd
	prepareCmd(cmd)

	if err = cmd.Start(); err != nil {
		return
	}
	if writepid != nil {
		fmt.Fprintln(writepid, cmd.Process.Pid)
	}
	if cmd.Process != nil {
		cmd.Process.Release()
	}
	os.Exit(0)
	return // not reached
}

// RemoveArgs is a helper function for Daemon(). Daemon calls the
// function with os.Args[1;] as in and removes any arguments
// matching members of the slice remove and returns out. Only bare
// arguments are removed and no pattern matching or adjacent values are
// removed. If this is required then pass your own function with the
// same signature.
func RemoveArgs(in []string, remove ...string) (out []string) {
OUTER:
	for _, a := range in {
		for _, r := range remove {
			if a == r {
				continue OUTER
			}
		}
		out = append(out, a)
	}
	return
}
