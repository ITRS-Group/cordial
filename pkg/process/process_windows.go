//go:build windows

package process

import (
	"os/exec"
	"syscall"
)

func prepareCmd(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}
	} else {
		cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP
	}
}

// user and group are sids
func setCredentials(cmd *exec.Cmd, user, group any) {
	// not implemented
}

func setCredentialsFromUsername(cmd *exec.Cmd, username string) (err error) {
	// not implemented
	return
}
