package remote

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/afero"
)

// Localhost operations
type Local struct {
}

func NewLocal() Remote {
	return &Local{}
}

func (h *Local) IsLocal() bool {
	return true
}

func (h *Local) IsAvailable() bool {
	return true
}

func (h *Local) String() string {
	return "localhost"
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

func (h *Local) Run(name string, args ...string) (output []byte, err error) {
	// run locally
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

func (h *Local) Path(path string) string {
	return path
}

func (h *Local) Failed() bool {
	return false
}

func (h *Local) ServerVersion() string {
	return runtime.GOOS
}

func (h *Local) NewAferoFS() afero.Fs {
	return afero.NewOsFs()
}

func (h *Local) Signal(pid int, signal syscall.Signal) (err error) {
	proc, _ := os.FindProcess(pid)
	if err = proc.Signal(signal); err != nil && !errors.Is(err, syscall.EEXIST) {
		return
	}
	return nil
}

func (h *Local) Start(cmd *exec.Cmd, env []string, username, home, errfile string) (err error) {
	cmd.Env = append(os.Environ(), env...)

	out, err := os.OpenFile(errfile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}

	procSetupOS(cmd, out)

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
