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
	Entries map[int]ProcessInfo
}

var procCacheTTL = 5 * time.Second

// ProcessInfo is a struct that holds information about a process.
// It is used to return information about a process on a host.
type ProcessInfo struct {
	PID          int
	PPID         int
	Children     []int
	Exe          string
	Cmdline      []string
	CreationTime time.Time
	UID          int
	GID          int
	TCPPorts     []int
	UDPPorts     []int
	status       map[string]string // save the status key/value pairs for later use if required
}

// ProcessStats is an example of a structure to pass to
// instance.ProcessStatus, using a field number for `stat` and a line
// prefix for `status` tags. OpenFiles and OpenSockets are counts of
// their respective names.
type ProcessStats struct {
	Pid         int64         `stat:"0"`
	Utime       time.Duration `stat:"13"`
	Stime       time.Duration `stat:"14"`
	CUtime      time.Duration `stat:"15"`
	CStime      time.Duration `stat:"16"`
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

func getProcCache(h host.Host, resetcache bool) (c procCache, ok bool) {
	if h.IsLocal() {
		return getLocalProcCache(resetcache)
	}

	// remote support for Linux only for now
	if h.ServerVersion() == "windows" {
		return
	}

	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()

	if !resetcache {
		if c, ok = procCacheMap[h]; ok {
			if time.Since(c.LastUpdate) < procCacheTTL {
				return
			}
		}
	}

	// cache is empty or expired, update it
	dirs, err := h.Glob("/proc/[0-9]*")
	if err != nil {
		return c, false
	}
	c.Entries = make(map[int]ProcessInfo, len(dirs))

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
		mtime := st.ModTime()

		var ppid int
		pstat, err := h.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			log.Debug().Err(err).Msgf("failed to read stat for pid %d", pid)
		} else {
			fields := strings.Fields(string(pstat))
			ppid, err = strconv.Atoi(fields[3])
			if err != nil {
				log.Debug().Err(err).Msgf("failed to parse ppid for pid %d", pid)
				// leave ppid as zero if we cannot parse it
			}
		}

		exe, _ := h.Readlink(fmt.Sprintf("/proc/%d/exe", pid))

		b, err := h.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			log.Debug().Err(err).Msgf("failed to read cmdline for pid %d", pid)
			continue
		}
		cmdline := strings.Split(strings.TrimSuffix(string(b), "\000"), "\000")

		status, err := readProcessStatus(h, pid)
		if err != nil {
			log.Debug().Err(err).Msgf("failed to read status for pid %d", pid)
			continue
		}

		var uid, gid = -1, -1
		uids := strings.Fields(status["uid"])
		gids := strings.Fields(status["gid"])

		if len(uids) > 0 {
			uid, _ = strconv.Atoi(uids[0])
		}
		if len(gids) > 0 {
			gid, _ = strconv.Atoi(gids[0])
		}

		c.Entries[pid] = ProcessInfo{
			PID:          pid,
			PPID:         ppid,
			Exe:          exe,
			Cmdline:      cmdline,
			CreationTime: mtime,
			UID:          uid,
			GID:          gid,
			status:       status,
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

// readProcessStatus reads the /proc/<pid>/status file for the process
// with the given pid on host h and returns a map of the entries in the
// file. The keys in the map are the lowercase names of the entries in
// the file, and the values are the corresponding values. If there is an
// error reading the file then an error is returned.
func readProcessStatus(h host.Host, pid int) (entries map[string]string, err error) {
	b, err := h.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return
	}
	entries = make(map[string]string)

	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		name, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		entries[strings.ToLower(strings.TrimSpace(name))] = strings.TrimSpace(value)
	}
	return entries, nil
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
func GetPID(h host.Host, binary string, resetcache bool, customCheckFunc func(checkarg any, cmdline []string) bool, checkarg any, args ...string) (int, error) {
	if h == nil {
		return 0, fmt.Errorf("host cannot be nil")
	}
	if binary == "" {
		return 0, fmt.Errorf("binaryPrefix must not be empty")
	}

	c, ok := getProcCache(h, resetcache)
	if !ok {
		return 0, fmt.Errorf("host %s does not support process lookups", h.ServerVersion())
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

// GetProcessInfo returns information about the process pid on host h.
func GetProcessInfo(h host.Host, pid int, resetcache bool) (pi ProcessInfo, err error) {
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
