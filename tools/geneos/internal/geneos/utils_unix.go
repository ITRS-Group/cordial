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

package geneos

import (
	"fmt"
	"io/fs"
	"os/user"
	"sync"
	"syscall"

	"github.com/pkg/sftp"
)

// FileOwner is only available on Linux localhost
type FileOwner struct {
	Uid int
	Gid int
}

func (h *Host) GetFileOwner(info fs.FileInfo) (s FileOwner) {
	switch h.GetString("name") {
	case LOCALHOST:
		s.Uid = int(info.Sys().(*syscall.Stat_t).Uid)
		s.Gid = int(info.Sys().(*syscall.Stat_t).Gid)
	default:
		s.Uid = int(info.Sys().(*sftp.FileStat).UID)
		s.Gid = int(info.Sys().(*sftp.FileStat).GID)
	}
	return
}

var usernames sync.Map

func GetUsername(s FileOwner) (username string) {
	if u, ok := usernames.Load(s.Uid); ok {
		username = u.(string)
		return
	}

	username = fmt.Sprint(s.Uid)
	u, err := user.LookupId(username)
	if err == nil && u.Name != "" {
		username = u.Username
	}

	usernames.Store(s.Uid, username)

	return
}

var groupnames sync.Map

func GetGroupname(s FileOwner) (groupname string) {
	if g, ok := usernames.Load(s.Gid); ok {
		groupname = g.(string)
		return
	}

	groupname = fmt.Sprint(s.Gid)
	g, err := user.LookupGroupId(groupname)
	if err == nil && g.Name != "" {
		groupname = g.Name
	}

	groupnames.Store(s.Gid, groupname)

	return
}
