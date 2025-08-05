/*
Copyright © 2023 ITRS Group

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

package process

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-reap"
	"github.com/rs/zerolog/log"

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
// the parent. writepid is closed in the parent.
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

	// OS specific (compile time/build constraint) changes to cmd
	prepareCmd(cmd)

	if err = cmd.Start(); err != nil {
		return
	}

	// write the resulting PID to writepid if non-nil. close writepid
	if writepid != nil {
		fmt.Fprintln(writepid, cmd.Process.Pid)
		writepid.Close()
	}
	if cmd.Process != nil {
		cmd.Process.Release()
	}
	os.Exit(0)
	return // not reached
}

// RemoveArgs is a helper function for Daemon(). Daemon calls the
// function with os.Args[1;] and removes any arguments matching members
// of the slice `remove` and returns the result. Only bare arguments are
// removed and no pattern matching or adjacent values are removed. If
// this is required then pass your own function with the same signature.
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

type procCache struct {
	// LastUpdate is the time when the cache was last updated
	LastUpdate time.Time

	// Entries is the map of process entries, indexed by PID
	Entries map[int]procCacheEntry
}

type procCacheEntry struct {
	PID     int
	Exe     string
	Cmdline []string
}

var procCacheTTL = 5 * time.Second

// procCache is a map of host to procCache, which is used to cache
// process entries for each host. It is used to avoid repeated calls to
// the host to get the process entries, which can be expensive.
//
// The cache is updated every 5 seconds, or when the cache is empty.
//
// The cache map is protected by a mutex to ensure that only one
// goroutine can update the cache at a time.
var procCacheMutex sync.Mutex
var procCacheMap = make(map[host.Host]procCache)

func getProcCache(h host.Host) (c procCache, ok bool) {
	if h.IsLocal() {
		return getLocalProcCache()
	}

	// remote support for Linux only for now
	if !strings.Contains(h.ServerVersion(), "linux") {
		return
	}

	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()

	if c, ok = procCacheMap[nil]; ok {
		if time.Since(c.LastUpdate) < procCacheTTL {
			return
		}
	}

	// cache is empty or expired, update it
	dirs, err := h.Glob("/proc/[0-9]*")
	if err != nil {
		return c, false
	}
	c.Entries = make(map[int]procCacheEntry, len(dirs))

	for _, dir := range dirs {
		pid, _ := strconv.Atoi(path.Base(dir))
		exe, _ := h.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		b, err := h.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			log.Debug().Err(err).Msgf("failed to read cmdline for pid %d", pid)
			continue
		}
		cmdline := strings.Split(strings.TrimSuffix(string(b), "\000"), "\000")
		c.Entries[pid] = procCacheEntry{
			PID:     pid,
			Exe:     exe,
			Cmdline: cmdline,
		}
	}
	c.LastUpdate = time.Now()
	procCacheMap[h] = c

	return c, true
}

// GetPID returns the PID of the process started with binary name and
// all args (in any order) on host h. If not found then an err of
// os.ErrProcessDone is returned.
//
// customCheckFunc() is a custom function to validate the args against
// the each process found and checkargs is passed to the function as a
// parameter. If the function returns true then the process is a match.
//
// walk the /proc directory (local or remote) and find the matching pid.
// This is subject to races, but not much we can do
//
// We use a process cache to avoid repeated calls to the host to get the
// process entries, which can be expensive. The cache is updated every 5
// seconds, or when the cache is empty.
func GetPID(h host.Host, binary string, customCheckFunc func(checkarg any, cmdline []string) bool, checkarg any, args ...string) (int, error) {
	c, ok := getProcCache(h)
	if !ok {
		return 0, fmt.Errorf("host %s does not support process lookups", h.ServerVersion())
	}

	if condition := h == nil || binary == ""; condition {
		return 0, fmt.Errorf("host cannot be nil and binaryPrefix must not be empty")
	}

	for _, pc := range c.Entries {
		// remove Linux specific suffixes
		exe := strings.TrimSuffix(pc.Exe, " (deleted)")

		if filepath.Base(exe) != binary {
			continue
		}

		if len(args) > len(pc.Cmdline)-1 {
			// not enough arguments to test, so it can't match all args
			continue
		}

		if customCheckFunc != nil {
			if customCheckFunc(checkarg, pc.Cmdline) {
				return pc.PID, nil
			}
			continue
		}

		if len(args) == 0 {
			// no args to check, so return the first match
			return pc.PID, nil
		}

		// check that all args passed to the function are somewhere on
		// the command line. we do not check the order of the args, just
		// that they are all present in the command line
		if slices.ContainsFunc(pc.Cmdline[1:], func(a string) bool {
			return slices.Contains(args, a)
		}) {
			return pc.PID, nil
		}
	}

	return 0, os.ErrProcessDone
}

type ProcessInfo struct {
	PID          int
	Exe          string
	Cmdline      []string
	CreationTime time.Time
	UID          int
	GID          int
	TCPPorts     []int
	UDPPorts     []int
}

// GetProcessInfo returns information about the process pid on host h.
func GetProcessInfo(h host.Host, pid int) (err error) {
	c, ok := getProcCache(h)
	if !ok {
		return fmt.Errorf("host %s does not support process lookups", h.ServerVersion())
	}

	if pc, ok := c.Entries[pid]; ok {
		log.Debug().Msgf("matched pid %d exe %s cmdline %v", pc.PID, pc.Exe, pc.Cmdline)
		return nil
	}
	return
}

// Program is a highly simplified representation of a program to manage
// with Start or Batch.
//
// Args and Env can be either a comma delimited string, which is split
// by viper/mapstructure, or a slice of strings which is used as-is
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
// unless Foreground is true.
//
// TODO: return error windows
// TODO: look at remote processes
func Start(h host.Host, program Program, options ...Options) (pid int, err error) {
	opts := evalOptions(options...)

	if h == nil {
		return 0, fmt.Errorf("host cannot be nil")
	}

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
		program.Args = config.ExpandStringSlice(program.Args, config.LookupTable(opts.lookup...))
	}

	if opts.expandEnv {
		program.Env = config.ExpandStringSlice(program.Env, config.LookupTable(opts.lookup...))
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
		cmd.Env = program.Env
		cmd.Dir = program.WorkingDir
		if _, err := h.Run(cmd, program.ErrLog); err != nil {
			return 0, retErrIfFalse(program.IgnoreErr, err)
		}
	case program.Restart:
		go reap.ReapChildren(nil, nil, nil, nil)
		go func() {
			for {
				cmd := exec.Command(program.Executable, program.Args...)
				setCredentialsFromUsername(cmd, program.Username)
				cmd.Env = program.Env
				cmd.Dir = program.WorkingDir
				h.Run(cmd, program.ErrLog)
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
		cmd.Env = program.Env
		cmd.Dir = program.WorkingDir
		if err = h.Start(cmd, program.ErrLog); err != nil {
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
