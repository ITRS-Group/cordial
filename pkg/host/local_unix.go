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
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func procSetupOS(cmd *exec.Cmd, out *os.File, options ...ProcessOptions) (err error) {
	po := evalProcessOptions(options...)

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// if we've set-up privs at all, set the redirection output file to the same
	if cmd.SysProcAttr.Credential != nil {
		if err = out.Chown(int(cmd.SysProcAttr.Credential.Uid), int(cmd.SysProcAttr.Credential.Gid)); err != nil {
			return
		}
	}

	if po.detach {
		// detach process by creating a session (fixed start + log)
		cmd.SysProcAttr.Setsid = true
	}

	// mark all non-std fds unshared
	cmd.ExtraFiles = []*os.File{}

	// un-inherit all FDs explicitly, leaving Extrafiles nil doesn't work
	fds, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return
	}
	var maxfd int
	for _, f := range fds {
		if i, err := strconv.Atoi(f.Name()); err == nil {
			if i > maxfd {
				maxfd = i
			}
		}
	}

	for i := 0; i < maxfd-3; i++ {
		cmd.ExtraFiles = append(cmd.ExtraFiles, nil)
	}

	return
}

func (h *Local) Lchtimes(path string, atime time.Time, mtime time.Time) (err error) {
	var ua, um int64
	if !atime.IsZero() {
		ua = atime.UnixMicro()
	}
	if !mtime.IsZero() {
		um = mtime.UnixMicro()
	}
	tv := []unix.Timeval{
		{Sec: ua / 1000000, Usec: ua % 1000000},
		{Sec: um / 1000000, Usec: um % 1000000},
	}
	return unix.Lutimes(path, tv)
}

func (h *Local) Symlink(oldname, newname string) (err error) {
	return os.Symlink(oldname, newname)
}
