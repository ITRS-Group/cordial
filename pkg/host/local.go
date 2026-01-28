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
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

// Localhost operations
type Local struct {
}

var Localhost = NewLocal()

func NewLocal() Host {
	return &Local{}
}

func (h *Local) Username() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return os.Getenv("USER")
}

func (h *Local) Hostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// IsLocal returns true if h is local, which for Local it is
func (h *Local) IsLocal() bool {
	return true
}

// IsAvailable returns true for Local
func (h *Local) IsAvailable() (bool, error) {
	return true, nil
}

func (h *Local) String() string {
	return "localhost"
}

func (h *Local) Abs(dir string) (abs string, err error) {
	abs, err = filepath.Abs(dir)
	if err != nil {
		return
	}
	abs = filepath.ToSlash(abs)
	return
}

func (h *Local) Getwd() (dir string, err error) {
	return os.Getwd()
}

// IsAbs on a Windows host will always use filepath.IsAbs()
func (h *Local) IsAbs(name string) bool {
	return filepath.IsAbs(name)
}

func (h *Local) Readlink(file string) (link string, err error) {
	return os.Readlink(file)
}

func (h *Local) Mkdir(p string, perm os.FileMode) (err error) {
	return os.Mkdir(p, perm)
}

func (h *Local) MkdirAll(p string, perm os.FileMode) (err error) {
	return os.MkdirAll(p, perm)
}

func (h *Local) Chown(name string, uid, gid int) (err error) {
	return os.Chown(name, uid, gid)
}

func (h *Local) Chtimes(path string, atime time.Time, mtime time.Time) (err error) {
	return os.Chtimes(path, atime, mtime)
}

// change the symlink ownership on local system, issue chown for remotes
func (h *Local) Lchown(name string, uid, gid int) (err error) {
	return os.Lchown(name, uid, gid)
}

func (h *Local) Create(p string, perms fs.FileMode) (out io.WriteCloser, err error) {
	return os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perms)
}

func (h *Local) Remove(name string) (err error) {
	return os.Remove(name)
}

func (h *Local) RemoveAll(name string) (err error) {
	return os.RemoveAll(name)
}

func (h *Local) Rename(oldpath, newpath string) (err error) {
	return os.Rename(oldpath, newpath)
}

func (h *Local) Link(oldname, newname string) (err error) {
	return os.Link(oldname, newname)
}

// Stat wraps the os.Stat and sftp.Stat functions
func (h *Local) Stat(name string) (f fs.FileInfo, err error) {
	return os.Stat(name)
}

// Lstat wraps the os.Lstat and sftp.Lstat functions
func (h *Local) Lstat(name string) (f fs.FileInfo, err error) {
	return os.Lstat(name)
}

func (h *Local) Glob(pattern string) (paths []string, err error) {
	paths, err = filepath.Glob(pattern)
	if err != nil {
		return
	}
	for i := range paths {
		paths[i] = filepath.ToSlash(paths[i])
	}
	return
}

func (h *Local) WriteFile(name string, data []byte, perm os.FileMode) (err error) {
	return os.WriteFile(name, data, perm)
}

func (h *Local) ReadFile(name string) (b []byte, err error) {
	return os.ReadFile(name)
}

// ReadDir reads the named directory and returns all its directory
// entries sorted by name.
func (h *Local) ReadDir(name string) (dirs []os.DirEntry, err error) {
	return os.ReadDir(name)
}

func (h *Local) Open(name string) (f io.ReadSeekCloser, err error) {
	return os.Open(name)
}

func (h *Local) HostPath(p string) string {
	return p
}

func (h *Local) LastError() error {
	return nil
}

func (h *Local) ServerVersion() string {
	return runtime.GOOS
}

func (h *Local) GetFs() afero.Fs {
	return afero.NewOsFs()
}

func (h *Local) TempDir() string {
	return os.TempDir()
}

func (h *Local) Signal(pid int, signal syscall.Signal) (err error) {
	proc, _ := os.FindProcess(pid)
	if err = proc.Signal(signal); err != nil && !errors.Is(err, syscall.EEXIST) {
		return
	}
	return nil
}

func (h *Local) Start(cmd *exec.Cmd, options ...ProcessOptions) (err error) {
	po := evalProcessOptions(options...)
	errfile := po.errfile
	if errfile == "" {
		errfile = os.DevNull
	} else if !h.IsAbs(errfile) {
		errfile = path.Join(cmd.Dir, errfile)
	}

	out, err := os.OpenFile(errfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer out.Close()

	if err = procSetupOS(cmd, out, ProcessDetach()); err != nil {
		return
	}

	cmd.Stdout = out
	cmd.Stderr = out

	if err = cmd.Start(); err != nil {
		return
	}

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

	if cmd.Process != nil {
		// detach from control
		cmd.Process.Release()
	}
	return
}

// Run starts a program, waits for completion and returns the output
// and/or any error. errfile is either absolute or relative to home.
func (h *Local) Run(cmd *exec.Cmd, options ...ProcessOptions) (output []byte, err error) {
	po := evalProcessOptions(options...)
	errfile := po.errfile
	if errfile == "" {
		errfile = os.DevNull
	} else if !h.IsAbs(errfile) {
		errfile = path.Join(cmd.Dir, errfile)
	}

	out, err := os.OpenFile(errfile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer out.Close()

	if err = procSetupOS(cmd, out); err != nil {
		return
	}

	cmd.Stderr = out

	return cmd.Output()
}

func (h *Local) Uname() (os, arch string, err error) {
	return runtime.GOOS, runtime.GOARCH, nil
}

func (h *Local) WalkDir(dir string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(os.DirFS(dir), ".", fn)
}
