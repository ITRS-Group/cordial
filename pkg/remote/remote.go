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

// Provides remote file and command functions
package remote

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/afero"
)

// Remote encapsulates all the methods required by callers to manage
// Geneos installs.
//
// This should have been based on (and extending) something like Afero,
// but this was quicker for the moment.
type Remote interface {
	IsLocal() bool
	IsAvailable() bool
	String() string
	Symlink(oldname, newname string) (err error)
	Readlink(file string) (link string, err error)
	MkdirAll(path string, perm os.FileMode) (err error)
	Chown(name string, uid, gid int) (err error)
	Lchown(name string, uid, gid int) (err error)
	Create(path string, perms fs.FileMode) (out io.WriteCloser, err error)
	Remove(name string) (err error)
	RemoveAll(name string) (err error)
	Rename(oldpath, newpath string) (err error)
	Stat(name string) (f fs.FileInfo, err error)
	Lstat(name string) (f fs.FileInfo, err error)
	Glob(pattern string) (paths []string, err error)
	WriteFile(name string, data []byte, perm os.FileMode) (err error)
	ReadFile(name string) (b []byte, err error)
	ReadDir(name string) (dirs []os.DirEntry, err error)
	Open(name string) (f io.ReadSeekCloser, err error)
	Run(name string, args ...string) (output []byte, err error)
	Start(cmd *exec.Cmd, env []string, username, home, errfile string) (err error)
	Path(path string) string
	Failed() bool
	ServerVersion() string
	NewAferoFS() afero.Fs
	Signal(pid int, signal syscall.Signal) (err error)
}

var (
	ErrInvalidArgs  = errors.New("invalid arguments")
	ErrNotSupported = errors.New("not supported")
)
