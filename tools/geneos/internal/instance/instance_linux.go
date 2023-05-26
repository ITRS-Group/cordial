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

package instance

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/rs/zerolog/log"
)

// GetPID returns the PID of the process running for the instance. If
// not found then an err of os.ErrProcessDone is returned.
//
// The process is identified by checking the conventions used to start
// Geneos processes.
//
// the component type must be part of the basename of the executable and
// the component name must be on the command line as an exact and
// standalone args
//
// walk the /proc directory (local or remote) and find the matching pid.
// This is subject to races, but not much we can do
func GetPID(c geneos.Instance) (pid int, err error) {
	if fn := c.Type().GetPID; fn != nil {
		return fn(c)
	}

	return process.GetPID(c.Host(), c.Config().GetString("binary"), c.Name())
}

func GetPIDInfo(c geneos.Instance) (pid int, uid uint32, gid uint32, mtime time.Time, err error) {
	pid, err = GetPID(c)
	if err == nil {
		var st os.FileInfo
		st, err = c.Host().Stat(fmt.Sprintf("/proc/%d", pid))
		s := c.Host().GetFileOwner(st)
		return pid, s.Uid, s.Gid, st.ModTime(), err
	}
	return 0, 0, 0, time.Time{}, os.ErrProcessDone
}

func allTCPListenPorts(c geneos.Instance, source string, ports map[int]int) (err error) {
	tcp, err := c.Host().Open(source)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(tcp)
	if scanner.Scan() {
		// skip headers
		_ = scanner.Text()
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 10 || fields[3] != "0A" {
				break
			}
			s := strings.SplitN(fields[1], ":", 2)
			if len(s) != 2 {
				continue
			}
			port, err := strconv.ParseInt(s[1], 16, 32)
			if err != nil {
				continue
			}
			inode, _ := strconv.Atoi(fields[9])
			ports[inode] = int(port)
		}
	}
	return
}

// TCPListenPorts returns all TCP ports currently open for the process
// running as the instance. An empty slice is returned if the process
// cannot be found. The instance may be on a remote host.
func TCPListenPorts(c geneos.Instance) (ports []int) {
	var err error

	if !IsRunning(c) {
		return
	}

	sockets := Sockets(c)
	if len(sockets) == 0 {
		return
	}

	tcpports := make(map[int]int)
	if err = allTCPListenPorts(c, "/proc/net/tcp", tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error().Err(err).Msg("continuing")
	}

	if err = allTCPListenPorts(c, "/proc/net/tcp6", tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error().Err(err).Msg("continuing")
	}

	for _, s := range sockets {
		if port, ok := tcpports[s]; ok {
			ports = append(ports, port)
			log.Debug().Msgf("process listening on %v", port)
		}
	}
	return
}

type OpenFiles struct {
	Path   string
	Stat   fs.FileInfo
	FD     string
	FDMode fs.FileMode
}

// Files returns a map of file descriptor (int) to file details
// (InstanceProcFiles) for all open, real, files for the process running
// as the instance. All paths that are not absolute paths are ignored.
// An empty map is returned if the process cannot be found.
func Files(c geneos.Instance) (openfiles map[int]OpenFiles) {
	pid, err := GetPID(c)
	if err != nil {
		return
	}

	file := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := c.Host().ReadDir(file)
	if err != nil {
		return
	}

	openfiles = make(map[int]OpenFiles, len(fds))

	for _, ent := range fds {
		fd := ent.Name()
		dest, err := c.Host().Readlink(path.Join(file, fd))
		if err != nil {
			continue
		}
		if !filepath.IsAbs(dest) {
			continue
		}
		n, _ := strconv.Atoi(fd)

		fdPath := path.Join(file, fd)
		fdMode, err := c.Host().Lstat(fdPath)
		if err != nil {
			continue
		}

		s, err := c.Host().Stat(dest)
		if err != nil {
			continue
		}

		openfiles[n] = OpenFiles{
			Path:   dest,
			Stat:   s,
			FD:     fdPath,
			FDMode: fdMode.Mode(),
		}

		log.Debug().Msgf("\tfd %s points to %q", fd, dest)
	}
	return
}

// Sockets returns a map[int]int of file descriptor to socket inode for all open
// files for the process running as the instance. An empty map is
// returned if the process cannot be found.
func Sockets(c geneos.Instance) (links map[int]int) {
	var inode int
	links = make(map[int]int)
	pid, err := GetPID(c)
	if err != nil {
		return
	}
	file := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := c.Host().ReadDir(file)
	if err != nil {
		return
	}
	for _, ent := range fds {
		fd := ent.Name()
		dest, err := c.Host().Readlink(path.Join(file, fd))
		if err != nil {
			continue
		}
		if n, err := fmt.Sscanf(dest, "socket:[%d]", &inode); err == nil && n == 1 {
			f, _ := strconv.Atoi(fd)
			links[f] = inode
			log.Debug().Msgf("\tfd %s points to socket %q", fd, inode)
		}
	}
	return
}
