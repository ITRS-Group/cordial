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
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
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

// GetPID returns the PID of the process started with binary name and
// all args (in any order) on host h. If not found then an err of
// os.ErrProcessDone is returned.
//
// walk the /proc directory (local or remote) and find the matching pid.
// This is subject to races, but not much we can do
//
// TODO: add support for windows hosts - the lookups are based on the
// host h and not the local system
func GetPID(h host.Host, binary string, args ...string) (pid int, err error) {
	var pids []int

	// safe to ignore error as it can only be bad pattern,
	// which means no matches to range over
	dirs, _ := h.Glob("/proc/[0-9]*")

	for _, dir := range dirs {
		p, _ := strconv.Atoi(filepath.Base(dir))
		pids = append(pids, p)
	}

	sort.Ints(pids)

	var data []byte
PIDS:
	for _, pid = range pids {
		if data, err = h.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err != nil {
			// process may disappear by this point, ignore error
			continue
		}
		procargs := bytes.Split(data, []byte("\000"))
		execfile := filepath.Base(string(procargs[0]))
		if strings.HasPrefix(execfile, binary) {
			argmap := make(map[string]bool)
			for _, arg := range procargs[1:] {
				argmap[string(arg)] = true
			}
			for _, arg := range args {
				if !argmap[arg] {
					continue PIDS
				}
			}
			return
		}
	}
	return 0, os.ErrProcessDone
}

// Start runs a process in the background on host h. username should be
// am empty string (for now). home is the working directory for the
// process, defaults to the home dir of the user if empty. out is the
// log to write stdout and stderr to. args are process args, env is a
// slice of key=value env vars.
func Start(h host.Host, binary string, username string, home string, out string, args, env []string) (err error) {
	// GetPID ...

	// changing users is not supported
	hUsername := h.Username()

	if username != "" && username != hUsername {
		return fmt.Errorf("cannot run as different user, yet")
	}
	if username == "" {
		u, _ := user.Current()
		username = u.Username
	}
	if home == "" {
		u, _ := user.Lookup(username)
		home = u.HomeDir
	}

	if _, err = h.Stat(binary); err != nil {
		return fmt.Errorf("%q %w", binary, err)
	}

	h.Start(exec.Command(binary, args...), env, home, out)

	pid, err := GetPID(h, binary)
	if err != nil {
		return err
	}
	log.Debug().Msgf("started with PID %d\n", pid)
	return nil
}

// keep this for later, not yet required but useful
func writerFromAny(dest any, flags int, perms fs.FileMode) (writer io.Writer, err error) {
	switch e := dest.(type) {
	case string:
		// open and set
		w, err := os.OpenFile(e, flags, perms)
		if err != nil {
			return nil, err

		}
		writer = w
	default:
		if w, ok := e.(io.Writer); ok {
			writer = w
		}
		err = fmt.Errorf("unknown writer destination type %T", e)
	}
	return
}
