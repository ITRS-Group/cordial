//go:build !windows

/*
Copyright © 2023 ITRS Group

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

package host

import (
	"cmp"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"syscall"

	"github.com/rs/zerolog/log"
)

func procSetupOS(cmd *exec.Cmd, out *os.File, detach bool) {
	var err error

	// if we've set-up privs at all, set the redirection output file to the same
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Credential != nil {
		if err = out.Chown(int(cmd.SysProcAttr.Credential.Uid), int(cmd.SysProcAttr.Credential.Gid)); err != nil {
			log.Error().Err(err).Msg("chown")
		}
	}
	if detach {
		// detach process by creating a session (fixed start + log)
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		cmd.SysProcAttr.Setsid = true
	}

	// mark all fds unshared
	fds, _ := os.ReadDir("/proc/self/fd")
	maxdir := slices.MaxFunc(fds, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	maxfd, _ := strconv.ParseInt(maxdir.Name(), 10, 64)
	maxfd -= 3
	for fd := int64(0); fd < maxfd; fd++ {
		cmd.ExtraFiles = append(cmd.ExtraFiles, nil)
	}
}
