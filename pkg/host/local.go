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
	"sync"
	"syscall"
	"time"

	"github.com/spf13/afero"
)

// Local implements Host operations against the current machine and filesystem.
type Local struct {
}

// Localhost is a Host backed by the current machine.
var Localhost = NewLocal()

// NewLocal returns a Host that performs operations on the current machine.
func NewLocal() Host {
	return &Local{}
}

// Username returns the current operating-system user's name.
func (h *Local) Username() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return os.Getenv("USER")
}

// Hostname returns the current machine's hostname.
func (h *Local) Hostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// IsLocalhost reports whether h represents the current machine.
func (h *Local) IsLocalhost() bool {
	return true
}

// IsAvailable reports that the local host is available.
func (h *Local) IsAvailable() (bool, error) {
	return true, nil
}

// String returns the host identifier "localhost".
func (h *Local) String() string {
	return "localhost"
}

// Abs returns dir as an absolute local path with slash separators.
func (h *Local) Abs(dir string) (abs string, err error) {
	abs, err = filepath.Abs(dir)
	if err != nil {
		return
	}
	abs = filepath.ToSlash(abs)
	return
}

// Base returns the final path element using the local operating system's rules.
func (h *Local) Base(path string) string {
	return filepath.Base(path)
}

// Dir returns the directory portion of path using local path rules.
func (h *Local) Dir(path string) string {
	return filepath.Dir(path)
}

// Ext returns path's file extension using local path rules.
func (h *Local) Ext(path string) string {
	return filepath.Ext(path)
}

// Join joins path elements using local path rules.
func (h *Local) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// ToSlash converts path separators to slash characters.
func (h *Local) ToSlash(path string) string {
	return filepath.ToSlash(path)
}

// VolumeName returns the volume component of path, if the operating system has one.
func (h *Local) VolumeName(path string) string {
	return filepath.VolumeName(path)
}

// Split separates path into its directory and final element.
func (h *Local) Split(path string) (dir, file string) {
	return filepath.Split(path)
}

// Getwd returns the current working directory.
func (h *Local) Getwd() (dir string, err error) {
	return os.Getwd()
}

// IsAbs reports whether name is an absolute path on the local operating system.
func (h *Local) IsAbs(name string) bool {
	return filepath.IsAbs(name)
}

// Readlink returns the target of file if it is a symbolic link.
func (h *Local) Readlink(file string) (link string, err error) {
	return os.Readlink(file)
}

// Mkdir creates one local directory with the supplied permissions.
func (h *Local) Mkdir(p string, perm os.FileMode) (err error) {
	return os.Mkdir(p, perm)
}

// MkdirAll creates p and any missing parent directories with the supplied permissions.
func (h *Local) MkdirAll(p string, perm os.FileMode) (err error) {
	return os.MkdirAll(p, perm)
}

// Chown changes the owner and group of a local filesystem path.
func (h *Local) Chown(name string, uid, gid int) (err error) {
	return os.Chown(name, uid, gid)
}

// Chtimes changes the access and modification times of a local filesystem path.
func (h *Local) Chtimes(path string, atime time.Time, mtime time.Time) (err error) {
	return os.Chtimes(path, atime, mtime)
}

// Lchown changes the owner and group of a local path without following a symlink.
func (h *Local) Lchown(name string, uid, gid int) (err error) {
	return os.Lchown(name, uid, gid)
}

// Create opens p for writing, creating or truncating it with the supplied permissions.
func (h *Local) Create(p string, perms fs.FileMode) (out io.WriteCloser, err error) {
	return os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perms)
}

// Remove deletes the named local file or empty directory.
func (h *Local) Remove(name string) (err error) {
	return os.Remove(name)
}

// RemoveAll deletes name and any children it contains.
func (h *Local) RemoveAll(name string) (err error) {
	return os.RemoveAll(name)
}

// Rename moves a local path from oldpath to newpath.
func (h *Local) Rename(oldpath, newpath string) (err error) {
	return os.Rename(oldpath, newpath)
}

// Link creates a hard link from oldname to newname.
func (h *Local) Link(oldname, newname string) (err error) {
	return os.Link(oldname, newname)
}

// Stat returns file information for name, following symbolic links.
func (h *Local) Stat(name string) (f fs.FileInfo, err error) {
	return os.Stat(name)
}

// Lstat returns file information for name without following symbolic links.
func (h *Local) Lstat(name string) (f fs.FileInfo, err error) {
	return os.Lstat(name)
}

// Glob returns local paths matching pattern, with slash-separated path elements.
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

// WriteFile writes data to name, creating or truncating the file with perm.
func (h *Local) WriteFile(name string, data []byte, perm os.FileMode) (err error) {
	return os.WriteFile(name, data, perm)
}

// ReadFile reads and returns the complete contents of name.
func (h *Local) ReadFile(name string) (b []byte, err error) {
	return os.ReadFile(name)
}

// ReadDir reads the named directory and returns all its directory
// entries sorted by name.
func (h *Local) ReadDir(name string) (dirs []os.DirEntry, err error) {
	return os.ReadDir(name)
}

// Open opens name for reading and returns a seekable read-closer.
func (h *Local) Open(name string) (f io.ReadSeekCloser, err error) {
	return os.Open(name)
}

// HostPath returns p unchanged because local paths need no host translation.
func (h *Local) HostPath(p string) string {
	return p
}

// LastError returns nil because local host operations do not cache connection errors.
func (h *Local) LastError() error {
	return nil
}

// ServerVersion returns the local operating-system identifier.
func (h *Local) ServerVersion() string {
	return runtime.GOOS
}

// OS returns the local operating-system identifier.
func (h *Local) OS() string {
	return runtime.GOOS
}

// GetFs returns an afero filesystem backed by the local operating system.
func (h *Local) GetFs() afero.Fs {
	return afero.NewOsFs()
}

// TempDir returns the local operating system's temporary directory.
func (h *Local) TempDir() string {
	return os.TempDir()
}

// Signal sends signal to the local process identified by pid.
func (h *Local) Signal(pid int, signal syscall.Signal) (err error) {
	proc, _ := os.FindProcess(pid)
	if err := proc.Signal(signal); err != nil && !errors.Is(err, syscall.EEXIST) {
		return err
	}
	return nil
}

var startMutex sync.Mutex

// Start launches cmd on the local host, detaching it unless options override
// process setup. Standard output and error are written to the configured error log.
func (h *Local) Start(cmd *exec.Cmd, options ...ProcessOption) (pid int, err error) {
	po := evalProcessOptions(options...)

	errfile := po.errfile
	if errfile == "" {
		errfile = os.DevNull
	} else if !h.IsAbs(errfile) {
		errfile = path.Join(cmd.Dir, errfile)
	}

	// Start is not thread-safe due to the use of cmd.Process.Release()
	// and options CPU affinity settings
	startMutex.Lock()
	defer startMutex.Unlock()

	out, err := os.OpenFile(errfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer out.Close()

	// TODO: check the logic for hardwiring detach
	if err = procSetupOS(cmd, out, append(options, ProcessDetach())...); err != nil {
		return
	}

	cmd.Stdout = out
	cmd.Stderr = out

	if err = cmd.Start(); err != nil {
		return
	}

	if err = postStart(cmd, options...); err != nil {
		return
	}

	if cmd.Process != nil {
		// get the PID before releasing control, otherwise
		// cmd.Process.Pid will be zero
		pid = cmd.Process.Pid

		// detach from control
		cmd.Process.Release()
	}

	return
}

// Run starts a program, waits for completion and returns the output
// and/or any error. errfile is either absolute or relative to home.
func (h *Local) Run(cmd *exec.Cmd, options ...ProcessOption) (output []byte, err error) {
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

	if err = procSetupOS(cmd, out, options...); err != nil {
		return
	}

	cmd.Stderr = out

	return cmd.Output()
}

// Uname TODO: Document its behavior, inputs, and effects.
func (h *Local) Uname() (os, arch string, err error) {
	return runtime.GOOS, runtime.GOARCH, nil
}

// WalkDir TODO: Document its behavior, inputs, and effects.
func (h *Local) WalkDir(dir string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(os.DirFS(dir), ".", fn)
}
