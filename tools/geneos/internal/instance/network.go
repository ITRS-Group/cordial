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
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/rs/zerolog/log"
)

var tcpfiles = []string{
	"/proc/net/tcp",
	"/proc/net/tcp6",
}

// GetAllPorts gets all used ports in config files on a specific remote
// and also all listening ports on the same host. Returns a map of port
// to bool "true" for each lookup.
func GetAllPorts(h *geneos.Host) (ports map[uint16]bool) {
	if h == geneos.ALL {
		log.Fatal().Msg("getports() call with all hosts")
	}
	ports = make(map[uint16]bool)
	for _, c := range GetAll(h, nil) {
		if c.Loaded().IsZero() {
			log.Error().Msgf("cannot load configuration for %s", c)
			continue
		}
		if port := c.Config().GetInt("port"); port != 0 {
			ports[uint16(port)] = true
		}
	}

	// add all listening ports
	listening := make(map[int]int)
	if err := allTCPListenPorts(h, listening); err != nil {
		return
	}
	for _, v := range listening {
		ports[uint16(v)] = true
	}
	return
}

// syntax of ranges of ints:
// x,y,a-b,c..d m n o-p
// also open ended A,N-,B
// command or space seperated?
// - or .. = inclusive range
//
// how to represent
// split, for range, check min-max -> max > min
// repeats ignored
// special ports? - nah
//

// given a range, find the first unused port
//
// range is comma or two-dot separated list of
// single number, e.g. "7036"
// min-max inclusive range, e.g. "7036-8036"
// start- open ended range, e.g. "7041-"
//
// some limits based on https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers
//
// not concurrency safe at this time
func NextPort(h *geneos.Host, ct *geneos.Component) uint16 {
	from := config.GetString(ct.PortRange)
	used := GetAllPorts(h)
	ps := strings.Split(from, ",")
	for _, p := range ps {
		// split on comma or ".."
		m := strings.SplitN(p, "-", 2)
		if len(m) == 1 {
			m = strings.SplitN(p, "..", 2)
		}

		if len(m) > 1 {
			var min uint16
			mn, err := strconv.Atoi(m[0])
			if err != nil {
				continue
			}
			if mn < 0 || mn > 65534 {
				min = 65535
			} else {
				min = uint16(mn)
			}
			if m[1] == "" {
				m[1] = "49151"
			}
			max, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			if int(min) >= max {
				continue
			}
			for i := min; int(i) <= max; i++ {
				if _, ok := used[i]; !ok {
					// found an unused port
					return i
				}
			}
		} else {
			var p1 uint16
			p, err := strconv.Atoi(m[0])
			if err != nil {
				continue
			}
			if p < 0 || p > 65534 {
				p1 = 65535
			} else {
				p1 = uint16(p)
			}
			if _, ok := used[p1]; !ok {
				return p1
			}
		}
	}
	return 0
}

// allTCPListenPorts returns a map of inodes to ports for all listening
// TCP ports from the source (typically /proc/net/tcp or /proc/net/tcp6)
// on host h. Will only work on Linux hosts.
func allTCPListenPorts(h *geneos.Host, ports map[int]int) (err error) {
	for _, source := range tcpfiles {
		tcp, err := h.Open(source)
		if err != nil {
			return err
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
	}
	return
}

// ListeningPorts returns all TCP ports currently open for the process
// running as the instance. An empty slice is returned if the process
// cannot be found. The instance may be on a remote host.
func ListeningPorts(i geneos.Instance) (ports []int) {
	var err error

	if !IsRunning(i) {
		return
	}

	sockets := sockets(i)
	if len(sockets) == 0 {
		return
	}

	tcpports := make(map[int]int) // key = socket inode
	if err = allTCPListenPorts(i.Host(), tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error().Err(err).Msg("continuing")
	}

	for _, s := range sockets {
		if port, ok := tcpports[s]; ok {
			ports = append(ports, port)
			log.Debug().Msgf("process listening on %v", port)
		}
	}
	sort.Ints(ports)
	return
}

// ListeningPorts returns all TCP ports currently open for the process
// running as the instance. An empty slice is returned if the process
// cannot be found. The instance may be on a remote host.
func ListeningPortsStrings(i geneos.Instance) (ports []string) {
	var err error

	if !IsRunning(i) {
		return
	}

	sockets := sockets(i)
	if len(sockets) == 0 {
		return
	}

	tcpports := make(map[int]int) // key = socket inode
	if err = allTCPListenPorts(i.Host(), tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error().Err(err).Msg("continuing")
	}

	for _, s := range sockets {
		if port, ok := tcpports[s]; ok {
			ports = append(ports, fmt.Sprint(port))
		}
	}
	sort.Strings(ports)
	return
}

// AllListeningPorts returns a sorted list of all listening TCP ports on
// host h between min and max (inclusive). If min or max is -1 then no
// limit is imposed.
// func AllListeningPorts(h *geneos.Host, min, max int) (ports []int) {
// 	var err error

// 	tcpports := make(map[int]int) // key = socket inode, value port
// 	if err = allTCPListenPorts(h, tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
// 		log.Debug().Err(err).Msg("continuing")
// 	}
// 	for v := range tcpports {
// 		if min == -1 || v >= min {
// 			if max == -1 || v <= max {
// 				ports = append(ports, v)
// 			}
// 		}
// 	}
// 	sort.Ints(ports)

// 	return
// }

// sockets returns a map[int]int of file descriptor to socket inode for all open
// files for the process running as the instance. An empty map is
// returned if the process cannot be found.
func sockets(i geneos.Instance) (links map[int]int) {
	var inode int
	links = make(map[int]int)
	pid, err := GetPID(i)
	if err != nil {
		return
	}
	file := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := i.Host().ReadDir(file)
	if err != nil {
		return
	}
	for _, ent := range fds {
		fd := ent.Name()
		dest, err := i.Host().Readlink(path.Join(file, fd))
		if err != nil {
			continue
		}
		if n, err := fmt.Sscanf(dest, "socket:[%d]", &inode); err == nil && n == 1 {
			f, _ := strconv.Atoi(fd)
			links[f] = inode
		}
	}
	return
}
