//go:build linux

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
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/sys/unix"
)

// setCPUAffinity restricts the calling OS thread to cpus, so that a
// child process started from it inherits the mask, and returns a
// function that puts the original mask back. The restore function must
// always be called, but is a no-op if cpus is empty. The calling
// goroutine is locked to its thread until restore is called.
func setCPUAffinity(cpus []int) (restore func() error, err error) {
	restore = func() error { return nil }
	if len(cpus) == 0 {
		return
	}

	var newMask unix.CPUSet
	for _, cpu := range cpus {
		if cpu < 0 {
			return restore, fmt.Errorf("invalid CPU %d for affinity", cpu)
		}
		newMask.Set(cpu)
	}

	runtime.LockOSThread()

	var originalMask unix.CPUSet
	if err = unix.SchedGetaffinity(0, &originalMask); err != nil {
		runtime.UnlockOSThread()
		return
	}
	if err = unix.SchedSetaffinity(0, &newMask); err != nil {
		runtime.UnlockOSThread()
		return
	}

	return func() error {
		// if the mask cannot be put back then leave the thread locked,
		// so that it is discarded when the goroutine ends rather than
		// being reused with the wrong mask
		if err := unix.SchedSetaffinity(0, &originalMask); err != nil {
			return err
		}
		runtime.UnlockOSThread()
		return nil
	}, nil
}

// postStart is called after the process has started, and can be used to
// perform any additional setup that requires the process to be running,
// such as setting resource limits or signal handlers.
func postStart(cmd *exec.Cmd, options ...ProcessOption) (err error) {
	po := evalProcessOptions(options...)

	if po.allowCoreDumps {
		// wait a bit to ensure process has started
		time.Sleep(100 * time.Millisecond)

		// failures to change limits are not fatal to starting the
		// process, so they are deliberately ignored

		// first, my limits
		var rlim unix.Rlimit
		if err := unix.Getrlimit(unix.RLIMIT_CORE, &rlim); err == nil && rlim.Cur == 0 {
			rlim.Cur = unix.RLIM_INFINITY
			_ = unix.Setrlimit(unix.RLIMIT_CORE, &rlim)
		}

		// then the child's, which is raised to the hard limit unless
		// core dumps are disabled
		if err := unix.Prlimit(cmd.Process.Pid, unix.RLIMIT_CORE, nil, &rlim); err == nil && rlim.Max != 0 {
			rlim.Cur = rlim.Max
			_ = unix.Prlimit(cmd.Process.Pid, unix.RLIMIT_CORE, &rlim, nil)
		}
	}

	return nil
}

// Lchtimes sets the access and modification times of path without
// following a symbolic link. A zero time leaves the corresponding time
// unchanged.
func (h *Local) Lchtimes(path string, atime time.Time, mtime time.Time) (err error) {
	ts := []unix.Timespec{timespecOrOmit(atime), timespecOrOmit(mtime)}
	return unix.UtimesNanoAt(unix.AT_FDCWD, path, ts, unix.AT_SYMLINK_NOFOLLOW)
}

func timespecOrOmit(t time.Time) unix.Timespec {
	if t.IsZero() {
		return unix.Timespec{Nsec: unix.UTIME_OMIT}
	}
	return unix.NsecToTimespec(t.UnixNano())
}
