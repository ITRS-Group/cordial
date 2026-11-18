//go:build !windows

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

package host

import (
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"syscall"

	"github.com/pkg/sftp"
)

// GetFileOwner returns the numeric owner and group of the file described
// by info, which can be from a local or remote (SFTP) host. If the
// owner cannot be determined, for example because info has no system
// details, then both are -1.
func GetFileOwner(h Host, info fs.FileInfo) (uid, gid int) {
	uid, gid = -1, -1
	if info == nil {
		return
	}
	switch sys := info.Sys().(type) {
	case *syscall.Stat_t:
		uid, gid = int(sys.Uid), int(sys.Gid)
	case *sftp.FileStat:
		uid, gid = int(sys.UID), int(sys.GID)
	}
	return
}

// procSetupOS is called before the process is started, and can be used
// to perform any OS-specific setup, such as setting process attributes
// or resource limits.
func procSetupOS(cmd *exec.Cmd, out *os.File, options ...ProcessOption) (err error) {
	po := evalProcessOptions(options...)

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// if we've set-up privs at all, set the redirection output file to
	// the same, but never change the ownership of the null device
	if cmd.SysProcAttr.Credential != nil && out.Name() != os.DevNull {
		if err = out.Chown(int(cmd.SysProcAttr.Credential.Uid), int(cmd.SysProcAttr.Credential.Gid)); err != nil {
			return
		}
	}

	if po.detach {
		// detach process by creating a session (fixed start + log)
		cmd.SysProcAttr.Setsid = true
	}

	closeInheritedFDs(cmd)

	return
}

// closeInheritedFDs un-inherits all FDs explicitly, as leaving
// ExtraFiles nil doesn't work. Any files the caller has set are kept and
// the remainder, up to the highest open descriptor, are marked as closed
// in the child. If the open descriptors cannot be listed (there is no
// /proc) then only the close-on-exec flags set by Go apply.
func closeInheritedFDs(cmd *exec.Cmd) {
	fds, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return
	}
	var maxfd int
	for _, f := range fds {
		if i, err := strconv.Atoi(f.Name()); err == nil {
			maxfd = max(maxfd, i)
		}
	}

	// ExtraFiles[i] is file descriptor 3+i
	if pad := maxfd - 2 - len(cmd.ExtraFiles); pad > 0 {
		cmd.ExtraFiles = slices.Concat(cmd.ExtraFiles, make([]*os.File, pad))
	}
}

// Symlink creates newname as a symbolic link to oldname.
func (h *Local) Symlink(oldname, newname string) (err error) {
	return os.Symlink(oldname, newname)
}
