//go:build !windows

package process

import (
	"io"
	"io/fs"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
)

// fakeInfo is a minimal fs.FileInfo
type fakeInfo struct {
	name string
	dir  bool
}

func (i fakeInfo) Name() string { return i.name }
func (i fakeInfo) Size() int64  { return 0 }
func (i fakeInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}
func (i fakeInfo) ModTime() time.Time { return time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC) }
func (i fakeInfo) IsDir() bool        { return i.dir }
func (i fakeInfo) Sys() any           { return nil }

// fakeHost is a host with a made up /proc. It embeds the interface, so
// anything that is not implemented panics.
type fakeHost struct {
	host.Host

	files map[string]string   // contents of files by path
	links map[string]string   // targets of symbolic links by path
	dirs  []string            // /proc/PID directories
	lists map[string][]string // names in directories, for ReadDir

	globs atomic.Int32

	mu     sync.Mutex
	opens  map[string]int
	closes int
}

func newFakeHost() *fakeHost {
	return &fakeHost{
		files: map[string]string{},
		links: map[string]string{},
		lists: map[string][]string{},
		opens: map[string]int{},
	}
}

func (f *fakeHost) String() string    { return "fake" }
func (f *fakeHost) IsLocalhost() bool { return false }
func (f *fakeHost) OS() string        { return "linux" }
func (f *fakeHost) Username() string  { return "fake" }

func (f *fakeHost) Glob(pattern string) ([]string, error) {
	f.globs.Add(1)
	if pattern == "/proc/[0-9]*" {
		return f.dirs, nil
	}
	return nil, fs.ErrNotExist
}

func (f *fakeHost) Stat(name string) (fs.FileInfo, error) {
	if _, ok := f.links[name]; ok {
		return fakeInfo{name: path.Base(name)}, nil
	}
	for _, d := range f.dirs {
		if d == name {
			return fakeInfo{name: path.Base(name), dir: true}, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (f *fakeHost) Lstat(name string) (fs.FileInfo, error) {
	return f.Stat(name)
}

func (f *fakeHost) Readlink(name string) (string, error) {
	if l, ok := f.links[name]; ok {
		return l, nil
	}
	return "", fs.ErrNotExist
}

func (f *fakeHost) ReadFile(name string) ([]byte, error) {
	if c, ok := f.files[name]; ok {
		return []byte(c), nil
	}
	return nil, fs.ErrNotExist
}

func (f *fakeHost) ReadDir(name string) ([]fs.DirEntry, error) {
	names, ok := f.lists[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	var entries []fs.DirEntry
	for _, n := range names {
		entries = append(entries, fs.FileInfoToDirEntry(fakeInfo{name: n}))
	}
	return entries, nil
}

type countingFile struct {
	*strings.Reader
	host *fakeHost
}

func (c countingFile) Close() error {
	c.host.mu.Lock()
	defer c.host.mu.Unlock()
	c.host.closes++
	return nil
}

func (f *fakeHost) Open(name string) (io.ReadSeekCloser, error) {
	c, ok := f.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opens[name]++
	return countingFile{strings.NewReader(c), f}, nil
}

func (f *fakeHost) opened(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.opens[name]
}

func (f *fakeHost) totalOpens() (n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.opens {
		n += c
	}
	return
}

func (f *fakeHost) closed() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closes
}
