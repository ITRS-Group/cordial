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
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
)

// ProcessInfo is an example of a structure to pass to
// instance.ProcessStatus, using a field number for `stat` and a line
// prefix for `status` tags. OpenFiles and OpenSockets fields are counts
// of their respective names. Some fields may be expensive to fill for
// all processes, so they are marked with `cache:"lazy"` to indicate
// that they should be filled on demand when requested, rather than when
// the process information is first retrieved.
type ProcessInfo struct {
	PID      int           `stat:"0" json:"-"`
	PPID     int           `stat:"3"`
	Utime    time.Duration `stat:"13"`
	Stime    time.Duration `stat:"14"`
	CUtime   time.Duration `stat:"15"`
	CStime   time.Duration `stat:"16"`
	UIDs     []string      `status:"Uid" json:"-"`
	GIDs     []string      `status:"Gid" json:"-"`
	State    string        `status:"State"`
	Threads  int64         `status:"Threads"`
	VmRSS    int64         `status:"VmRSS"`
	VmHWM    int64         `status:"VmHWM"`
	RssAnon  int64         `status:"RssAnon"`
	RssFile  int64         `status:"RssFile"`
	RssShmem int64         `status:"RssShmem"`

	// special fields that are not from /proc/PID/stat or
	// /proc/PID/status but are calculated from other information, such
	// as the number of open files and sockets
	OpenFiles      []ProcessFDs `cache:"lazy" json:"-"` // calculated from /proc/PID/fd
	OpenSockets    int64        `cache:"lazy" json:"-"` // calculated from /proc/PID/fd and /proc/PID/net/tcp and /proc/PID/net/udp
	ListeningPorts string       `cache:"lazy" json:"-"` // calculated from /proc/PID/net/tcp and /proc/PID/net/udp

	// these fields are not filled by ProcessStatus but are included in ProcessInfo for convenience
	// TCPPorts  []int     `json:"-"` // calculated from /proc/PID/net/tcp
	// UDPPorts  []int     `json:"-"` // calculated from /proc/PID/net/udp
	Cwd       string    `json:"-"` // calculated from /proc/PID/cwd
	Exe       string    `json:"-"`
	Cmdline   []string  `json:"-"`
	StartTime time.Time `json:"-"`
	Children  []int     `json:"-"`
	UID       int       `json:"-"` // real UID
	EUID      int       `json:"-"` // effective UID
	GID       int       `json:"-"` // real GID
	EGID      int       `json:"-"` // effective GID
	Username  string    `json:"-"`
	Groupname string    `json:"-"`
}

// ProcessInfo is an example of a structure to pass to
// instance.ProcessStatus, using a field number for `stat` and a line
// prefix for `status` tags. OpenFiles and OpenSockets fields are counts
// of their respective names.
type ProcessInfoMinimal struct {
	PID     int      `stat:"0" json:"-"`
	Exe     string   `json:"-"`
	Cmdline []string `json:"-"`
}

// PID returns the PID of the process started with executable name and
// all args (in any order) on host h. If not found then an err of
// os.ErrProcessDone is returned.
//
// A CustomChecker() function can be passed as an option to validate the
// args against the each process found. If the function returns true
// then the process is a match.
//
// By default a process cache is used to avoid repeated calls to the
// host to get the process entries, which can be expensive. The cache is
// updated every 5 seconds, or when the cache is empty. The cache can be
// reset/refreshed by passing the RefreshCache() option.
func PID(h host.Host, executable string, args []string, options ...ProcessOption) (pid int, err error) {
	if h == nil {
		return 0, fmt.Errorf("host cannot be nil")
	}
	if executable == "" {
		return 0, fmt.Errorf("executable must not be empty")
	}

	opts := evalProcessOptions(options...)
	c, ok := getProcesses[*ProcessInfoMinimal](h, opts.refreshCache)
	if !ok {
		return 0, fmt.Errorf("host %s does not support process lookups", h.ServerVersion())
	}

	for _, pc := range c {
		// remove Linux specific suffixes
		exe := strings.TrimSuffix(pc.Exe, " (deleted)")

		if filepath.Base(exe) != executable {
			continue
		}

		if len(args) > len(pc.Cmdline)-1 {
			// not enough arguments to test, so it can't match all args
			continue
		}

		if opts.checkFunc != nil {
			if opts.checkFunc(opts.checkArg, pc.Cmdline) {
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

// GetProcessInfo returns information about the process pid on host h.
func GetProcessInfo(h host.Host, pid int, resetcache bool) (pi *ProcessInfo, err error) {
	// c, ok := getProcCache(h, resetcache)
	c, ok := getProcesses[*ProcessInfo](h, resetcache)
	if !ok {
		return pi, fmt.Errorf("host %s does not support process lookups", h.ServerVersion())
	}

	if pc, ok := c[pid]; ok {
		// check and fill cache for lazy fields
		checkAndFillCache(h, pid, pc)
		return pc, nil
	}
	return pi, os.ErrProcessDone
}
