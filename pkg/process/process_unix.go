//go:build linux

package process

import (
	"fmt"
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
