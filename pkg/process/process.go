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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/host"
)

type procCache struct {
	// LastUpdate is the time when the cache was last updated
	LastUpdate time.Time

	// Entries is the map of process entries, indexed by PID
	Entries map[int]*ProcessInfo
}

var procCacheTTL = 5 * time.Second

// ProcessInfo is an example of a structure to pass to
// instance.ProcessStatus, using a field number for `stat` and a line
// prefix for `status` tags. OpenFiles and OpenSockets fields are counts
// of their respective names.
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
	OpenFiles   int64 `json:"-"` // calculated from /proc/PID/fd
	OpenSockets int64 `json:"-"` // calculated from /proc/PID/fd and /proc/PID/net/tcp and /proc/PID/net/udp

	// these fields are not filled by ProcessStatus but are included in ProcessInfo for convenience
	TCPPorts     []int     `json:"-"` // calculated from /proc/PID/net/tcp
	UDPPorts     []int     `json:"-"` // calculated from /proc/PID/net/udp
	Exe          string    `json:"-"`
	Cmdline      []string  `json:"-"`
	CreationTime time.Time `json:"-"`
	Children     []int     `json:"-"`
	UID          int       `json:"-"` // real UID
	EUID         int       `json:"-"` // effective UID
	GID          int       `json:"-"` // real GID
	EGID         int       `json:"-"` // effective GID
	Username     string    `json:"-"`
	Groupname    string    `json:"-"`
}

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

// getProcCache returns the procCache for the given host h. If
// refreshCache is true then the cache is refreshed even if it is not
// expired. If the host does not support process lookups then ok will be
// false.
//
// this assumes host h is Linux with an accessible /proc filesystem
func getProcCache(h host.Host, refreshCache bool) (c procCache, ok bool) {
	// remote support for Linux only for now
	if h.ServerVersion() == "windows" {
		if h.IsLocal() {
			// local windows only, for now
			return getWindowsProcCache(refreshCache)
		}
		return
	}

	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()

	if !refreshCache {
		if c, ok = procCacheMap[h]; ok {
			if time.Since(c.LastUpdate) < procCacheTTL {
				return
			}
		}
		// if not found, try to get live data by refreshing the cache,
		// drop through
	}

	// cache is empty or expired, update it
	dirs, err := h.Glob("/proc/[0-9]*")
	if err != nil {
		return c, false
	}
	c.Entries = make(map[int]*ProcessInfo, len(dirs))

	for _, dir := range dirs {
		st, err := h.Stat(dir)
		if err != nil {
			log.Debug().Err(err).Msgf("failed to stat %s", dir)
			continue
		}
		if !st.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(st.Name())
		if err != nil {
			log.Debug().Err(err).Msgf("failed to parse pid from %s", dir)
			continue
		}

		psexe, _ := h.Readlink(fmt.Sprintf("/proc/%d/exe", pid))

		b, err := h.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			log.Debug().Err(err).Msgf("failed to read cmdline for pid %d", pid)
			continue
		}

		pstatus := &ProcessInfo{
			PID:          pid,
			UID:          -1,
			GID:          -1,
			EUID:         -1,
			EGID:         -1,
			CreationTime: st.ModTime(),
			Exe:          strings.TrimSuffix(psexe, " (deleted)"),
			Cmdline:      strings.Split(strings.TrimSuffix(string(b), "\000"), "\000"),
		}

		if err = ProcessStatus(h, pid, pstatus); err != nil {
			log.Debug().Err(err).Msgf("failed to get process status for pid %d", pid)
			continue
		}

		if len(pstatus.UIDs) > 0 {
			pstatus.UID, _ = strconv.Atoi(pstatus.UIDs[0])
		}
		if len(pstatus.GIDs) > 0 {
			pstatus.GID, _ = strconv.Atoi(pstatus.GIDs[0])
		}

		pstatus.Username = GetUsername(pstatus.UID)
		pstatus.Groupname = GetGroupname(pstatus.GID)

		c.Entries[pid] = pstatus
	}

	// build child lists
	for _, p := range c.Entries {
		if parent, ok := c.Entries[p.PPID]; ok {
			parent.Children = append(parent.Children, p.PID)
			c.Entries[p.PPID] = parent
		}
	}

	c.LastUpdate = time.Now()
	procCacheMap[h] = c

	return c, true
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
	c, ok := getProcCache(h, opts.refreshCache)
	if !ok {
		return 0, fmt.Errorf("host %s does not support process lookups", h.ServerVersion())
	}

	for _, pc := range c.Entries {
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
	c, ok := getProcCache(h, resetcache)
	if !ok {
		return pi, fmt.Errorf("host %s does not support process lookups", h.ServerVersion())
	}

	if pc, ok := c.Entries[pid]; ok {
		log.Debug().Msgf("matched pid %d exe %s cmdline %v", pc.PID, pc.Exe, pc.Cmdline)
		return pc, nil
	}
	return pi, os.ErrProcessDone
}
