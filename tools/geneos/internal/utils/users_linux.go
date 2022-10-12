package utils

import (
	"os"
	"os/exec"
	"syscall"
)

// set-up the Cmd to set uid, gid and groups of the username given
// Note: does not change stdout etc. which is done later
func SetUser(cmd *exec.Cmd, username string) (err error) {
	uid, gid, gids, err := GetIDs(username)
	if err != nil {
		return
	}

	// do not set-up credentials if no-change
	if os.Getuid() == uid {
		return nil
	}

	// no point continuing if not root
	if !IsSuperuser() {
		return os.ErrPermission
	}

	// convert gids...
	var ugids []uint32
	for _, g := range gids {
		if g < 0 {
			continue
		}
		ugids = append(ugids, uint32(g))
	}
	cred := &syscall.Credential{
		Uid:         uint32(uid),
		Gid:         uint32(gid),
		Groups:      ugids,
		NoSetGroups: false,
	}
	sys := &syscall.SysProcAttr{Credential: cred}

	cmd.SysProcAttr = sys
	return nil
}
