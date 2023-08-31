//go:build !windows

/*
Copyright © 2022 ITRS Group

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
