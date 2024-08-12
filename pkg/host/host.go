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

// Provides remote file and command functions
package host

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/pkg/sftp"
	"github.com/spf13/afero"
)

// Host encapsulates all the methods required by callers to manage Geneos
// installs on a host.
//
// This should have been based on (and extending) something like Afero, but this
// was quicker for the moment. This interface also provides process handling
// etc.
type Host interface {
	// informational
	String() string
	GetFs() afero.Fs
	HostPath(p string) string // return the path as a string, prefixed with "host:" if not local
	Hostname() string
	ServerVersion() string
	IsAvailable() (bool, error)
	IsLocal() bool
	LastError() error
	Username() string

	// file operations
	Abs(name string) (string, error)
	Getwd() (dir string, err error)
	Chown(name string, uid, gid int) (err error)
	Glob(pattern string) (paths []string, err error)
	Lchown(name string, uid, gid int) (err error)
	Lstat(name string) (f fs.FileInfo, err error)
	MkdirAll(p string, perm os.FileMode) (err error)
	ReadDir(name string) (dirs []os.DirEntry, err error)
	ReadFile(name string) (b []byte, err error)
	Readlink(file string) (link string, err error)
	Remove(name string) (err error)
	RemoveAll(name string) (err error)
	Rename(oldpath, newpath string) (err error)
	Stat(name string) (f fs.FileInfo, err error)
	Symlink(oldname, newname string) (err error)
	TempDir() string
	WriteFile(name string, data []byte, perm os.FileMode) (err error)
	// these two do not conform to the afero / os interface
	Open(name string) (f io.ReadSeekCloser, err error)
	Create(p string, perms fs.FileMode) (out io.WriteCloser, err error)

	// process control
	Signal(pid int, signal syscall.Signal) (err error)
	Start(cmd *exec.Cmd, errfile string) (err error)
	Run(cmd *exec.Cmd, errfile string) (stdout []byte, err error)
}

var (
	ErrInvalidArgs  = errors.New("invalid arguments")
	ErrNotSupported = errors.New("not supported")
	ErrNotAvailable = errors.New("not available")
	ErrExist        = errors.New("already exists")
	ErrNotExist     = errors.New("does not exist")
)

// CopyFile copies a file between two locations. Destination can be a
// directory or a file. Parent directories will be created as required.
// Any existing files will be overwritten.
func CopyFile(srcHost Host, srcPath string, dstHost Host, dstPath string) (err error) {
	ss, err := srcHost.Stat(srcPath)
	if err != nil {
		return err
	}
	if ss.IsDir() {
		return fs.ErrInvalid
	}

	sf, err := srcHost.Open(srcPath)
	if err != nil {
		return err
	}
	defer sf.Close()

	ds, err := dstHost.Stat(dstPath)
	if err == nil {
		if ds.IsDir() {
			dstPath = path.Join(dstPath, path.Base(srcPath))
		}
	} else {
		dstHost.MkdirAll(path.Dir(dstPath), 0775)
	}

	df, err := dstHost.Create(dstPath, ss.Mode())
	if err != nil {
		return err
	}
	defer df.Close()
	if _, err = io.Copy(df, sf); err != nil {
		return err
	}
	return
}

// CopyAll copies a directory between any combination of local or remote locations
func CopyAll(srcHost Host, srcDir string, dstHost Host, dstDir string) (err error) {
	if srcHost.IsLocal() {
		filesystem := os.DirFS(srcDir)
		fs.WalkDir(filesystem, ".", func(file string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			fi, err := d.Info()
			if err != nil {
				return nil
			}
			dstPath := path.Join(dstDir, file)
			srcPath := path.Join(srcDir, file)
			return processDirEntry(fi, srcHost, srcPath, dstHost, dstPath)
		})
		return
	}

	if sf, ok := srcHost.(*SSHRemote); ok {
		var s *sftp.Client
		s, err = sf.DialSFTP()
		if err != nil {
			return
		}
		w := s.Walk(srcDir)
		for w.Step() {
			if w.Err() != nil {
				return
			}
			fi := w.Stat()
			srcPath := w.Path()
			dstPath := path.Join(dstDir, strings.TrimPrefix(w.Path(), srcDir))
			if err = processDirEntry(fi, srcHost, srcPath, dstHost, dstPath); err != nil {
				return
			}
		}
	}

	return
}

func processDirEntry(fi fs.FileInfo, srcHost Host, srcPath string, dstHost Host, dstPath string) (err error) {
	switch {
	case fi.IsDir():
		ds, err := srcHost.Stat(srcPath)
		if err != nil {
			return err
		}
		if err = dstHost.MkdirAll(dstPath, ds.Mode()); err != nil {
			return err
		}
	case fi.Mode()&fs.ModeSymlink != 0:
		link, err := srcHost.Readlink(srcPath)
		if err != nil {
			return err
		}
		if err = dstHost.Symlink(link, dstPath); err != nil {
			return err
		}
	default:
		ss, err := srcHost.Stat(srcPath)
		if err != nil {
			return err
		}
		sf, err := srcHost.Open(srcPath)
		if err != nil {
			return err
		}
		defer sf.Close()
		df, err := dstHost.Create(dstPath, ss.Mode())
		if err != nil {
			return err
		}
		defer df.Close()
		if _, err = io.Copy(df, sf); err != nil {
			return err
		}
	}
	return nil
}
