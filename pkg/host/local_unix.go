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

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

// procSetupOS is called before the process is started, and can be used
// to perform any OS-specific setup, such as setting process attributes
// or resource limits.
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

// postStart is called after the process has started, and can be used to
// perform any additional setup that requires the process to be running,
// such as setting resource limits or signal handlers.
func postStart(cmd *exec.Cmd, options ...ProcessOptions) (err error) {
	po := evalProcessOptions(options...)

	// this doesn't work as memguard lowers the hard limit to 0. There
	// is a bug raised: https://github.com/awnumar/memguard/issues/166
	// and hopefully it can change in the future, than we can have
	// per-instance options to allow core-dumps for diagnostics.
	if po.createCore {
		// wait a bit to ensure process has started
		time.Sleep(100 * time.Millisecond)
		// enable core dumps for the process
		var rlim unix.Rlimit

		// first, my limits
		if err = unix.Getrlimit(unix.RLIMIT_CORE, &rlim); err != nil {
			log.Debug().Err(err).Msg("Failed to get core dump limit")
			err = nil
		} else {
			log.Debug().Uint64("cur", rlim.Cur).Uint64("max", rlim.Max).Msg("Current core dump limits for parent")
		}

		if rlim.Cur == 0 {
			rlim.Cur = unix.RLIM_INFINITY
			// rlim.Max = unix.RLIM_INFINITY
			if err = unix.Setrlimit(unix.RLIMIT_CORE, &rlim); err != nil {
				log.Debug().Err(err).Msg("Failed to set core dump limit for parent")
				err = nil
			} else {
				log.Debug().Uint64("cur", rlim.Cur).Uint64("max", rlim.Max).Msg("Core dumps enabled for parent")
			}
		}

		// first get current limits
		if err = unix.Prlimit(cmd.Process.Pid, unix.RLIMIT_CORE, nil, &rlim); err != nil {
			log.Debug().Err(err).Int("pid", cmd.Process.Pid).Msg("Failed to get core dump limit")
			err = nil
		}

		log.Debug().Uint64("cur", rlim.Cur).Uint64("max", rlim.Max).Int("pid", cmd.Process.Pid).Msg("Current core dump limits")

		switch rlim.Max {
		case 0:
			// core dumps disabled
			log.Debug().Int("pid", cmd.Process.Pid).Msg("Core dumps are disabled for process")
		default:
			rlim.Cur = rlim.Max
			if err = unix.Prlimit(cmd.Process.Pid, unix.RLIMIT_CORE, &rlim, nil); err != nil {
				log.Debug().Err(err).Int("pid", cmd.Process.Pid).Msg("Failed to set core dump limit")
				err = nil
			} else {
				log.Debug().Int("pid", cmd.Process.Pid).Uint64("limit", rlim.Cur).Msg("Core dumps enabled for process")
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
