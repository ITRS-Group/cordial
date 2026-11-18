//go:build !windows && !linux

/*
Copyright © 2026 ITRS Group

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
	"time"

	"golang.org/x/sys/unix"
)

// setCPUAffinity is not supported on this platform, so any requested
// affinity is ignored and the returned restore function does nothing.
func setCPUAffinity(cpus []int) (restore func() error, err error) {
	return func() error { return nil }, nil
}

// postStart is called after the process has started. The only setup
// supported on this platform is to allow core dumps for the calling
// process, as there is no way to change the limits of another process.
func postStart(cmd *exec.Cmd, options ...ProcessOption) (err error) {
	po := evalProcessOptions(options...)

	if po.allowCoreDumps {
		var rlim unix.Rlimit
		if err := unix.Getrlimit(unix.RLIMIT_CORE, &rlim); err == nil && rlim.Cur == 0 {
			rlim.Cur = unix.RLIM_INFINITY
			_ = unix.Setrlimit(unix.RLIMIT_CORE, &rlim)
		}
	}

	return nil
}

// Lchtimes sets the access and modification times of path without
// following a symbolic link. Without support for leaving one time
// unchanged on this platform, a zero mtime is replaced with the current
// modification time of path and a zero atime with the resulting mtime.
func (h *Local) Lchtimes(path string, atime time.Time, mtime time.Time) (err error) {
	if mtime.IsZero() {
		fi, err := os.Lstat(path)
		if err != nil {
			return err
		}
		mtime = fi.ModTime()
	}
	if atime.IsZero() {
		atime = mtime
	}
	tv := []unix.Timeval{
		unix.NsecToTimeval(atime.UnixNano()),
		unix.NsecToTimeval(mtime.UnixNano()),
	}
	return unix.Lutimes(path, tv)
}
