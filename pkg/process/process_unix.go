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
	"strconv"
	"syscall"
)

func prepareCmd(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
	} else {
		cmd.SysProcAttr.Setsid = true
	}
}

func setCredentials(cmd *exec.Cmd, user, group any) {
	uid, _ := strconv.ParseUint(fmt.Sprint(user), 10, 32)
	gid, _ := strconv.ParseUint(fmt.Sprint(group), 10, 32)
	creds := &syscall.Credential{
		Uid: uint32(uid),
		Gid: uint32(gid),
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: creds,
		}
	} else {
		cmd.SysProcAttr.Credential = creds
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
