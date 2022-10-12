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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/rs/zerolog/log"
)

// locate a process instance
//
// the component type must be part of the basename of the executable and
// the component name must be on the command line as an exact and
// standalone args
//
// walk the /proc directory (local or remote) and find the matching pid
// this is subject to races, but not much we can do
func GetPID(c geneos.Instance) (pid int, err error) {
	var pids []int
	binary := c.Config().GetString("binary")

	// safe to ignore error as it can only be bad pattern,
	// which means no matches to range over
	dirs, _ := c.Host().Glob("/proc/[0-9]*")

	for _, dir := range dirs {
		p, _ := strconv.Atoi(filepath.Base(dir))
		pids = append(pids, p)
	}

	sort.Ints(pids)

	var data []byte
	for _, pid = range pids {
		if data, err = c.Host().ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err != nil {
			// process may disappear by this point, ignore error
			continue
		}
		args := bytes.Split(data, []byte("\000"))
		execfile := filepath.Base(string(args[0]))
		switch c.Type() {
		case geneos.ParseComponentName("webserver"):
			var wdOK, jarOK bool
			if execfile != "java" {
				continue
			}
			for _, arg := range args[1:] {
				if string(arg) == "-Dworking.directory="+c.Home() {
					wdOK = true
				}
				if strings.HasSuffix(string(arg), "geneos-web-server.jar") {
					jarOK = true
				}
				if wdOK && jarOK {
					return
				}
			}
		default:
			if strings.HasPrefix(execfile, binary) {
				for _, arg := range args[1:] {
					// very simplistic - we look for a bare arg that matches the instance name
					if string(arg) == c.Name() {
						// found
						return
					}
				}
			}
		}
	}
	return 0, os.ErrProcessDone
}

func GetPIDInfo(c geneos.Instance) (pid int, uid uint32, gid uint32, mtime int64, err error) {
	pid, err = GetPID(c)
	if err == nil {
		var s host.FileStat
		s, err = c.Host().StatX(fmt.Sprintf("/proc/%d", pid))
		return pid, s.Uid, s.Gid, s.Mtime, err
	}
	return 0, 0, 0, 0, os.ErrProcessDone
}

func Ports(c geneos.Instance) (ports []int) {
	links := Files(c)

	tcp, err := c.Host().Open("/proc/net/tcp")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// udp, _ := c.Host().ReadFile("/proc/net/udp")

	tcpports := make(map[int]int)
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
			var ip [4]int
			var port int
			fmt.Sscanf(fields[1], "%2X%2X%2X%2X:%X", &ip[0], &ip[1], &ip[2], &ip[3], &port)
			inode, _ := strconv.Atoi(fields[9])
			log.Debug().Msgf("ip %v port %v inode %v", ip, port, inode)
			tcpports[inode] = port
		}
	}

	var inode int
	for _, l := range links {
		if n, err := fmt.Sscanf(l, "socket:[%d]", &inode); err == nil && n == 1 {
			if port, ok := tcpports[inode]; ok {
				ports = append(ports, port)
				log.Debug().Msgf("process listening on %v", port)
			}
		}
	}
	return
}

func Files(c geneos.Instance) (links map[int]string) {
	links = make(map[int]string)
	pid, err := GetPID(c)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	path := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := c.Host().ReadDir(path)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	for _, ent := range fds {
		fd := ent.Name()
		dest, err := c.Host().Readlink(filepath.Join(path, fd))
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		n, _ := strconv.Atoi(fd)
		links[n] = dest
		log.Debug().Msgf("\tfd %s points to %q", fd, dest)
	}
	return
}

func procSetupOS(cmd *exec.Cmd, out *os.File) {
	var err error

	// if we've set-up privs at all, set the redirection output file to the same
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Credential != nil {
		if err = out.Chown(int(cmd.SysProcAttr.Credential.Uid), int(cmd.SysProcAttr.Credential.Gid)); err != nil {
			log.Error().Err(err).Msg("chown")
		}
	}
	// detach process by creating a session (fixed start + log)
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
