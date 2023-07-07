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
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/hashicorp/go-reap"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
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
// written to writepid, if not nil. Remember to only open the file
// inside the test for daemon mode in the caller, otherwise on
// re-execution the file will be re-opened and overwrite the one from
// the parent.
//
// On failure the function does return with an error.
//
//	process.Daemon(os.Stdout, process.RemoveArgs, "-D", "--daemon")
func Daemon(writepid io.WriteCloser, processArgs func([]string, ...string) []string, args ...string) (err error) {
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
		if _, err = fmt.Fprintln(writepid, cmd.Process.Pid); err != nil {
			// too late to return now, just try to log
			log.Error().Err(err).Msgf("pid %d", cmd.Process.Pid)
		}
	}
	writepid.Close()
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

// key is host name (not hostname)
var pidcache = make(map[string][]int)
var dirs = make(map[string][]string)

// GetPID returns the PID of the process started with binary name and
// all args (in any order) on host h. If not found then an err of
// os.ErrProcessDone is returned.
//
// walk the /proc directory (local or remote) and find the matching pid.
// This is subject to races, but not much we can do
//
// TODO: add support for windows hosts - the lookups are based on the
// host h and not the local system
//
// TODO: cache /proc entries for a period, this is very likely to be
// used over and over in the same proc
func GetPID(h host.Host, binary string, checkfn func(string, interface{}, string, [][]byte) bool, checkarg interface{}, args ...string) (pid int, err error) {
	if strings.Contains(h.ServerVersion(), "windows") {
		return 0, os.ErrProcessDone
	}

	hostname := h.String()

	if cmdlineCacheTime[hostname].IsZero() || time.Since(cmdlineCacheTime[hostname]) > 5*time.Second {
		// (re)initialise caches

		cmdlineCacheTime[hostname] = time.Now()
		cmdlineCache[hostname] = map[int][]byte{}

		// safe to ignore error as it can only be bad pattern,
		// which means no matches to range over
		dirs[hostname], err = h.Glob("/proc/[0-9]*")
		if err != nil {
			err = os.ErrProcessDone
			return
		}

		pids := []int{}
		for _, dir := range dirs[hostname] {
			p, _ := strconv.Atoi(path.Base(dir))
			pids = append(pids, p)
		}

		sort.Ints(pids)
		pidcache[hostname] = pids
	}

	var data []byte
PIDS:
	for _, pid = range pidcache[hostname] {
		var ok bool
		if data, ok = cmdlineCache[hostname][pid]; !ok {
			if data, err = h.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err != nil {
				// process may disappear by this point, perms denied etc., ignore error
				continue
			}
			cmdlineCache[hostname][pid] = data
		}
		procargs := bytes.Split(data, []byte("\000"))
		execfile := path.Base(string(procargs[0]))
		if checkfn != nil {
			if checkfn(binary, checkarg, execfile, procargs) {
				return
			}
		} else {
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
	}
	return 0, os.ErrProcessDone
}

var cmdlineCache = make(map[string]map[int][]byte)
var cmdlineCacheTime = make(map[string]time.Time)

// Program is a highly simplified representation of a program to
// manage with Start or Batch.
//
// Args and Env can be either a string, which is split on whitespace or
// a slice of strings which is used as-is
type Program struct {
	Executable string        `json:"executable,omitempty"` // Path to program, passed through exec.LookPath
	Username   string        `json:"username,omitempty"`   // The name of the user, if empty use current
	WorkingDir string        `json:"workingdir,omitempty"` // The working directory, defaults to home dir of user
	ErrLog     string        `json:"errlog,omitempty"`     // The name of the logfile, defaults to basename of program+".log" in Dir
	Args       []string      `json:"args,omitempty"`       // Args not including program
	Env        []string      `json:"env,omitempty"`        // Env as key=value pairs
	Foreground bool          `json:"foreground,omitempty"` // Run in foreground, to completion, return if result != 0
	Restart    bool          `json:"restart,omitempty"`    // restart on exit
	IgnoreErr  bool          `json:"ignoreerr,omitempty"`  // If true do not return err on failure
	Wait       time.Duration `json:"wait,omitempty"`       // period to wait after starting, default none
}

func retErrIfFalse(ret bool, err error) error {
	if !ret {
		return err
	}
	return nil
}

// Start runs a process on host h. It is run detached in the background
// unless Foreground is true. username should be am empty string (for
// now). home is the working directory for the process, defaults to the
// home dir of the user if empty. out is the log to write stdout and
// stderr to. args are process args, env is a slice of key=value env
// vars.
//
// TODO: return error windows
// TODO: look at remote processes
func Start(h host.Host, program Program, options ...Options) (pid int, err error) {
	opts := evalOptions(options...)

	if program.Username != "" && program.Username != h.Username() {
		// if username is set and is not current user for host h

		// if not root
		if os.Getuid() != 0 && os.Geteuid() != 0 {
			return 0, os.ErrPermission
		}

		// if root, and not localhost
		if h != host.Localhost {
			return 0, host.ErrInvalidArgs
		}
		u, err := user.Lookup(program.Username)
		if err != nil {
			return 0, err
		}
		if program.WorkingDir == "" {
			program.WorkingDir = u.HomeDir
		}
	} else {
		if program.Username == "" {
			if u, err := user.Current(); err == nil {
				program.Username = u.Username
			} else {
				program.Username = os.Getenv("USER")
			}
		}

		if program.WorkingDir == "" {
			u, _ := user.Lookup(program.Username)
			program.WorkingDir = u.HomeDir
		}
	}

	// build basic env after any caller values set, keep PATH
	program.Env = append(program.Env,
		"HOME="+program.WorkingDir,
		"SHELL=/bin/bash",
		"USER="+program.Username,
		"LOGNAME="+program.Username,
		"PATH="+os.Getenv("PATH"),
	)

	// check for expand options
	if opts.expandArgs {
		program.Args = config.ExpandStringSlice(program.Args, config.LookupTables(opts.lookup))
	}

	if opts.expandEnv {
		program.Env = config.ExpandStringSlice(program.Env, config.LookupTables(opts.lookup))
	}

	if program.ErrLog == "" {
		program.ErrLog = path.Join(program.WorkingDir, path.Base(program.Executable+".log"))
	} else if !path.IsAbs(program.ErrLog) {
		program.ErrLog = path.Join(program.WorkingDir, program.ErrLog)
	}

	p, err := exec.LookPath(program.Executable)
	if err != nil {
		return 0, retErrIfFalse(program.IgnoreErr, fmt.Errorf("%q %w", p, err))
	}

	switch {
	case program.Foreground:
		cmd := exec.Command(program.Executable, program.Args...)
		setCredentialsFromUsername(cmd, program.Username)
		if _, err := h.Run(cmd, program.Env, program.WorkingDir, program.ErrLog); err != nil {
			log.Fatal().Err(err).Msg("")
			return 0, retErrIfFalse(program.IgnoreErr, err)
		}
	case program.Restart:
		go reap.ReapChildren(nil, nil, nil, nil)
		go func() {
			for {
				cmd := exec.Command(program.Executable, program.Args...)
				setCredentialsFromUsername(cmd, program.Username)

				h.Run(cmd, program.Env, program.WorkingDir, program.ErrLog)
				if program.Wait != 0 {
					time.Sleep(program.Wait)
				} else {
					time.Sleep(500 * time.Millisecond)
				}
			}
		}()
	default:
		cmd := exec.Command(program.Executable, program.Args...)
		setCredentialsFromUsername(cmd, program.Username)
		if err = h.Start(cmd, program.Env, program.WorkingDir, program.ErrLog); err != nil {
			return 0, retErrIfFalse(program.IgnoreErr, err)
		}
	}

	if program.Wait != 0 {
		time.Sleep(program.Wait)
	}

	// only valid if long running
	pid, err = GetPID(h, p, nil, nil)
	err = retErrIfFalse(program.IgnoreErr, err)
	return
}

// Batch executes the slice of Program entries using Start. If any stage
// returns err then Batch returns immediately. Set IgnoreErr in Program
// to not return errors for each stage. If any stage has Restart set and
// it is supported then a reaper is run and the done channel returned.
func Batch(h host.Host, batch []Program, options ...Options) (done chan struct{}, err error) {
	r := false
	for _, program := range batch {
		if program.Restart {
			r = true
		}
		_, err = Start(h, program, options...)
		if err != nil && err != os.ErrProcessDone {
			return
		}
	}
	if r && reap.IsSupported() {
		go reap.ReapChildren(nil, nil, done, nil)
	}

	if err == os.ErrProcessDone {
		err = nil
	}
	return
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

// sliceFromAny returns a slice of strings from value. If value is a
// string then it is split using strings.Fields(), if it is a slice of
// strings then it is returned otherwise and empty slice and error are
// returned.
func sliceFromAny(value any) (out []string, err error) {
	if value == nil {
		return
	}
	switch e := value.(type) {
	case string:
		out = strings.Fields(e)
	case []string:
		out = e
	default:
		err = fmt.Errorf("unsupported type %T", e)
	}
	return
}
