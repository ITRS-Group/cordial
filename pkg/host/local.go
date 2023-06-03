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
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
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
	u, _ := user.Current()
	return u.Username
}

// IsLocal returns true if h is local, which for Local it is
func (h *Local) IsLocal() bool {
	return true
}

// IsAvailable returns true for Local
func (h *Local) IsAvailable() bool {
	return true
}

func (h *Local) String() string {
	return "localhost"
}

func (h *Local) Abs(dir string) (string, error) {
	return filepath.Abs(dir)
}

func (h *Local) Getwd() (dir string, err error) {
	return os.Getwd()
}

func (h *Local) Symlink(oldname, newname string) (err error) {
	return os.Symlink(oldname, newname)
}

func (h *Local) Readlink(file string) (link string, err error) {
	return os.Readlink(file)
}

func (h *Local) MkdirAll(path string, perm os.FileMode) (err error) {
	return os.MkdirAll(path, perm)
}

func (h *Local) Chown(name string, uid, gid int) (err error) {
	return os.Chown(name, uid, gid)
}

// change the symlink ownership on local system, issue chown for remotes
func (h *Local) Lchown(name string, uid, gid int) (err error) {
	return os.Lchown(name, uid, gid)
}

func (h *Local) Create(path string, perms fs.FileMode) (out io.WriteCloser, err error) {
	var cf *os.File
	cf, err = os.Create(path)
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
	return filepath.Glob(pattern)
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

func (h *Local) Path(path string) string {
	return path
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

func (h *Local) Signal(pid int, signal syscall.Signal) (err error) {
	proc, _ := os.FindProcess(pid)
	if err = proc.Signal(signal); err != nil && !errors.Is(err, syscall.EEXIST) {
		return
	}
	return nil
}

func (h *Local) Start(cmd *exec.Cmd, env []string, home, errfile string) (err error) {
	cmd.Env = append(os.Environ(), env...)

	out, err := os.OpenFile(errfile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}

	procSetupOS(cmd, out, true)

	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Dir = home

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
// and/or any error
func (h *Local) Run(cmd *exec.Cmd, env []string, home, errfile string) (output []byte, err error) {
	cmd.Env = append(os.Environ(), env...)

	if errfile == "" {
		errfile = os.DevNull
	}

	out, err := os.OpenFile(errfile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}

	procSetupOS(cmd, out, false)

	cmd.WaitDelay = 200 * time.Millisecond

	var buf bytes.Buffer
	cmd.Stdout = &buf
	// cmd.Stdout = out

	cmd.Stderr = out
	cmd.Dir = home

	if err = cmd.Start(); err != nil {
		return
	}

	if err = cmd.Wait(); err != nil {
		return
	}

	return buf.Bytes(), err
}
