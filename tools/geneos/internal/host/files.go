/*
Copyright © 2022 ITRS Group

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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/sftp"
)

// file handling

var (
	ErrInvalidArgs  = fmt.Errorf("invalid argument")
	ErrNotSupported = fmt.Errorf("not supported")
)

// shim methods that test Host and direct to ssh / sftp / os
// at some point this should become interface based to allow other
// remote protocols cleanly
func (h *Host) Symlink(oldname, newname string) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Symlink(oldname, newname)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Symlink(oldname, newname)
	}
}

func (h *Host) Readlink(file string) (link string, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Readlink(file)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.ReadLink(file)
	}
}

func (h *Host) MkdirAll(path string, perm os.FileMode) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.MkdirAll(path, perm)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.MkdirAll(path)
	}
}

func (h *Host) Chown(name string, uid, gid int) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Chown(name, uid, gid)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Chown(name, uid, gid)
	}
}

// change the symlink ownership on local system, issue chown for remotes
func (h *Host) Lchown(name string, uid, gid int) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Lchown(name, uid, gid)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Chown(name, uid, gid)
	}
}

func (h *Host) Create(path string, perms fs.FileMode) (out io.WriteCloser, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		var cf *os.File
		cf, err = os.Create(path)
		if err != nil {
			return
		}
		out = cf
		if err = cf.Chmod(perms); err != nil {
			return
		}
	default:
		var cf *sftp.File
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		if cf, err = s.Create(path); err != nil {
			return
		}
		out = cf
		if err = cf.Chmod(perms); err != nil {
			return
		}
	}
	return
}

func (h *Host) Remove(name string) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Remove(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Remove(name)
	}
}

func (h *Host) RemoveAll(name string) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.RemoveAll(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}

		// walk, reverse order by prepending and remove
		// we could also just reverse sort strings...
		files := []string{}
		w := s.Walk(name)
		for w.Step() {
			if w.Err() != nil {
				continue
			}
			files = append([]string{w.Path()}, files...)
		}
		for _, file := range files {
			if err = s.Remove(file); err != nil {
				return
			}
		}
		return
	}
}

func (h *Host) Rename(oldpath, newpath string) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Rename(oldpath, newpath)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		// use PosixRename to overwrite oldpath
		return s.PosixRename(oldpath, newpath)
	}
}

// massaged file stats
type FileOwner struct {
	Uid uint32
	Gid uint32
}

// Stat wraps the os.Stat and sftp.Stat functions
func (h *Host) Stat(name string) (f fs.FileInfo, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Stat(name)
	default:
		var sf *sftp.Client
		if sf, err = h.DialSFTP(); err != nil {
			return
		}
		return sf.Stat(name)
	}
}

// Lstat wraps the os.Lstat and sftp.Lstat functions
func (h *Host) Lstat(name string) (f fs.FileInfo, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Lstat(name)
	default:
		var sf *sftp.Client
		if sf, err = h.DialSFTP(); err != nil {
			return
		}
		return sf.Lstat(name)
	}
}

func (h *Host) Glob(pattern string) (paths []string, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return filepath.Glob(pattern)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Glob(pattern)
	}
}

func (h *Host) WriteFile(name string, data []byte, perm os.FileMode) (err error) {
	var s *sftp.Client
	var f *sftp.File

	if h == LOCAL {
		return os.WriteFile(name, data, perm)
	}
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	if f, err = s.Create(name); err != nil {
		return
	}
	defer f.Close()
	f.Chmod(perm)
	_, err = f.Write(data)
	return
}

func (h *Host) ReadFile(name string) (b []byte, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.ReadFile(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		f, err := s.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		st, err := f.Stat()
		if err != nil {
			return nil, err
		}
		// force a block read as /proc doesn't give sizes
		sz := st.Size()
		if sz == 0 {
			sz = 8192
		}
		return io.ReadAll(f)
	}
}

// ReadDir reads the named directory and returns all its directory
// entries sorted by name.
func (h *Host) ReadDir(name string) (dirs []os.DirEntry, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.ReadDir(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		f, err := s.ReadDir(name)
		if err != nil {
			return nil, err
		}
		sort.Slice(f, func(i, j int) bool {
			return f[i].Name() < f[j].Name()
		})
		for _, d := range f {
			dirs = append(dirs, fs.FileInfoToDirEntry(d))
		}
	}
	return
}

func (h *Host) Open(name string) (f io.ReadSeekCloser, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		f, err = os.Open(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		f, err = s.Open(name)
	}
	return
}
