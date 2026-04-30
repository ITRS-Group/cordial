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
	Entries map[int64]ProcessInfo
}

var procCacheTTL = 5 * time.Second

// ProcessInfo is a struct that holds information about a process.
// It is used to return information about a process on a host.
type ProcessInfo struct {
	PID          int64
	PPID         int64
	Children     []int64
	Exe          string
	Cmdline      []string
	CreationTime time.Time
	UID          int
	GID          int
	Username     string
	Groupname    string
	TCPPorts     []int
	UDPPorts     []int
}

// ProcessStats is an example of a structure to pass to
// instance.ProcessStatus, using a field number for `stat` and a line
// prefix for `status` tags. OpenFiles and OpenSockets fields are counts
// of their respective names.
type ProcessStats struct {
	PID         int64         `stat:"0" json:"-"`
	PPID        int64         `stat:"3"`
	Utime       time.Duration `stat:"13"`
	Stime       time.Duration `stat:"14"`
	CUtime      time.Duration `stat:"15"`
	CStime      time.Duration `stat:"16"`
	UIDs        []string      `status:"Uid" json:"-"`
	GIDs        []string      `status:"Gid" json:"-"`
	State       string        `status:"State"`
	Threads     int64         `status:"Threads"`
	VmRSS       int64         `status:"VmRSS"`
	VmHWM       int64         `status:"VmHWM"`
	RssAnon     int64         `status:"RssAnon"`
	RssFile     int64         `status:"RssFile"`
	RssShmem    int64         `status:"RssShmem"`
	OpenFiles   int64
	OpenSockets int64
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
	if h.IsLocal() {
		return getLocalProcCache(refreshCache)
	}

	// remote support for Linux only for now
	if h.ServerVersion() == "windows" {
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
	c.Entries = make(map[int64]ProcessInfo, len(dirs))

	for _, dir := range dirs {
		st, err := h.Stat(dir)
		if err != nil {
			log.Debug().Err(err).Msgf("failed to stat %s", dir)
			continue
		}
		if !st.IsDir() {
			continue
		}
		pid, err := strconv.ParseInt(st.Name(), 10, 64)
		if err != nil {
			log.Debug().Err(err).Msgf("failed to parse pid from %s", dir)
			continue
		}
		mtime := st.ModTime()

		// this does not truncate the binary name, which can happen in
		// /proc/PID/stat, but we need to remove any " (deleted)" suffix
		// that can be added to the exe name when the binary is deleted
		// after the process starts
		exe, _ := h.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		exe = strings.TrimSuffix(exe, " (deleted)")

		b, err := h.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			log.Debug().Err(err).Msgf("failed to read cmdline for pid %d", pid)
			continue
		}
		cmdline := strings.Split(strings.TrimSuffix(string(b), "\000"), "\000")

		var uid, gid = -1, -1

		pstatus := &ProcessStats{}
		ProcessStatus(h, pid, pstatus)

		if len(pstatus.UIDs) > 0 {
			uid, _ = strconv.Atoi(pstatus.UIDs[0])
		}
		if len(pstatus.GIDs) > 0 {
			gid, _ = strconv.Atoi(pstatus.GIDs[0])
		}

		c.Entries[pid] = ProcessInfo{
			PID:          pstatus.PID,
			PPID:         pstatus.PPID,
			Exe:          exe,
			Cmdline:      cmdline,
			CreationTime: mtime,
			UID:          uid,
			GID:          gid,
			Username:     GetUsername(uid),
			Groupname:    GetGroupname(gid),
		}
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
func PID(h host.Host, executable string, args []string, options ...ProcessOption) (int64, error) {
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
func GetProcessInfo(h host.Host, pid int64, resetcache bool) (pi ProcessInfo, err error) {
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
