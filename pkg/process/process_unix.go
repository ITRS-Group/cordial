//go:build !windows

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

package process

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// cache lookups, including fails
const notfound = "[NOT FOUND]"

var usernames sync.Map
var groupnames sync.Map

func GetUsername(uid int) (username string) {
	if u, ok := usernames.Load(uid); ok {
		username = u.(string)
		if username == notfound {
			username = fmt.Sprint(uid)
		}
		return
	}

	username = fmt.Sprint(uid)
	u, err := user.LookupId(username)
	if err != nil || u.Username == "" {
		usernames.Store(uid, notfound)
		return
	}
	username = u.Username
	usernames.Store(uid, username)

	return
}

func GetGroupname(gid int) (groupname string) {
	if g, ok := groupnames.Load(gid); ok {
		groupname = g.(string)
		if groupname == notfound {
			groupname = fmt.Sprint(gid)
		}
		return
	}

	groupname = fmt.Sprint(gid)
	g, err := user.LookupGroupId(groupname)
	if err != nil || g.Name == "" {
		groupnames.Store(gid, notfound)
		return
	}
	groupname = g.Name
	groupnames.Store(gid, groupname)

	return
}

func prepareCmd(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
	} else {
		cmd.SysProcAttr.Setsid = true
	}
}

func setCredentialsFromUsername(cmd *exec.Cmd, username string) (err error) {
	// setting the Credential struct causes errors and confusion if you
	// are not privileged
	if os.Getuid() != 0 && os.Geteuid() != 0 {
		return os.ErrPermission
	}

	u, err := user.Lookup(username)
	if err != nil {
		return
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return
	}
	groups := []uint32{}
	gids, _ := u.GroupIds()
	for _, g := range gids {
		var gid uint64
		gid, err = strconv.ParseUint(g, 10, 32)
		if err != nil {
			return
		}
		groups = append(groups, uint32(gid))
	}
	creds := &syscall.Credential{
		Uid:    uint32(uid),
		Gid:    uint32(gid),
		Groups: groups,
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: creds,
		}
	} else {
		cmd.SysProcAttr.Credential = creds
	}
	return
}

func getLocalProcCache(resetcache bool) (c procCache, ok bool) {
	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()

	if !resetcache {
		if c, ok = procCacheMap[nil]; ok {
			if time.Since(c.LastUpdate) < procCacheTTL {
				return
			}
		}
	}

	// cache is empty or expired, update it
	dirs, err := filepath.Glob("/proc/[0-9]*")
	if err != nil {
		return
	}
	c.Entries = make(map[int]ProcessInfo, len(dirs))

	for _, dir := range dirs {
		st, err := os.Stat(dir)
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
		pstat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
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

		exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if err != nil {
			continue
		}
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			continue
		}
		cmdline := strings.Split(strings.TrimSuffix(string(b), "\000"), "\000")

		uid, gid := ownerOfFile(st)

		// status, err := readProcessStatus(nil, pid)
		// if err != nil {
		// 	log.Debug().Err(err).Msgf("failed to read status for pid %d", pid)
		// 	continue
		// 	// leave status as nil if we cannot read it
		// }
		c.Entries[pid] = ProcessInfo{
			PID:          pid,
			PPID:         ppid,
			Exe:          exe,
			Cmdline:      cmdline,
			CreationTime: mtime,
			UID:          uid,
			GID:          gid,
			// status:       status, // not required locally
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
	procCacheMap[nil] = c

	return c, true
}

func ownerOfFile(st os.FileInfo) (uid, gid int) {
	if stat, ok := st.Sys().(*syscall.Stat_t); ok {
		return int(stat.Uid), int(stat.Gid)
	}
	return -1, -1
}
