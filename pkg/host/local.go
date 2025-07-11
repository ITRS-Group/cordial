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

	"github.com/spf13/afero"
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
	var cf *os.File
	cf, err = os.Create(p)
	if err != nil {
		return
	}
	out = cf
	err = cf.Chmod(perms)
	return
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

func (h *Local) Start(cmd *exec.Cmd, errfile string) (err error) {
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

	if err = procSetupOS(cmd, out, true); err != nil {
		return
	}

	cmd.Stdout = out
	cmd.Stderr = out

	if err = cmd.Start(); err != nil {
		return
	}

	if cmd.Process != nil {
		// detach from control
		cmd.Process.Release()
	}
	return
}

// Run starts a program, waits for completion and returns the output
// and/or any error. errfile is either absolute or relative to home.
func (h *Local) Run(cmd *exec.Cmd, errfile string) (output []byte, err error) {
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

	if err = procSetupOS(cmd, out, false); err != nil {
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
