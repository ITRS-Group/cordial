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
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/sys/unix"
)

func GetFileOwner(h Host, info fs.FileInfo) (uid, gid int) {
	if h.IsLocalhost() {
		uid = int(info.Sys().(*syscall.Stat_t).Uid)
		gid = int(info.Sys().(*syscall.Stat_t).Gid)
	} else {
		uid = int(info.Sys().(*sftp.FileStat).UID)
		gid = int(info.Sys().(*sftp.FileStat).GID)
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

// postStart is called after the process has started, and can be used to
// perform any additional setup that requires the process to be running,
// such as setting resource limits or signal handlers.
func postStart(cmd *exec.Cmd, options ...ProcessOption) (err error) {
	po := evalProcessOptions(options...)

	if po.allowCoreDumps {
		// wait a bit to ensure process has started
		time.Sleep(100 * time.Millisecond)
		// enable core dumps for the process
		var rlim unix.Rlimit

		// first, my limits
		if err = unix.Getrlimit(unix.RLIMIT_CORE, &rlim); err != nil {
			err = nil
		}

		if rlim.Cur == 0 {
			rlim.Cur = unix.RLIM_INFINITY
			// rlim.Max = unix.RLIM_INFINITY
			if err = unix.Setrlimit(unix.RLIMIT_CORE, &rlim); err != nil {
				err = nil
			}
		}

		// first get current limits
		if err = unix.Prlimit(cmd.Process.Pid, unix.RLIMIT_CORE, nil, &rlim); err != nil {
			err = nil
		}

		switch rlim.Max {
		case 0:
			// core dumps disabled
		default:
			rlim.Cur = rlim.Max
			if err = unix.Prlimit(cmd.Process.Pid, unix.RLIMIT_CORE, &rlim, nil); err != nil {
				err = nil
			} else {
			}
		}
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
