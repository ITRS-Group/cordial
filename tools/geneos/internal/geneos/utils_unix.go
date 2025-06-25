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

// cache lookups, including fails
const notfound = "[NOT FOUND]"

var usernames sync.Map
var groupnames sync.Map

func (h *Host) GetFileOwner(info fs.FileInfo) (uid, gid int) {
	switch h.GetString("name") {
	case LOCALHOST:
		uid = int(info.Sys().(*syscall.Stat_t).Uid)
		gid = int(info.Sys().(*syscall.Stat_t).Gid)
	default:
		uid = int(info.Sys().(*sftp.FileStat).UID)
		gid = int(info.Sys().(*sftp.FileStat).GID)
	}
	return
}

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
