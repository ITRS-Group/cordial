/*
Copyright © 2022 ITRS Group

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

package instance

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var tcpfiles = []string{
	"/proc/net/tcp",
	"/proc/net/tcp6",
}

var udpfiles = []string{
	"/proc/net/udp",
	"/proc/net/udp6",
}

// from linux net/tcp_states.h
const (
	_ = iota
	TCP_ESTABLISHED
	TCP_SYN_SENT
	TCP_SYN_RECV
	TCP_FIN_WAIT1
	TCP_FIN_WAIT2
	TCP_TIME_WAIT
	TCP_CLOSE
	TCP_CLOSE_WAIT
	TCP_LAST_ACK
	TCP_LISTEN
	TCP_CLOSING
	TCP_NEW_SYN_RECV
)

var stateNames = map[int]string{
	TCP_ESTABLISHED:  "ESTABLISHED",
	TCP_SYN_SENT:     "SYN-SENT",
	TCP_SYN_RECV:     "SYN-RECEIVED",
	TCP_FIN_WAIT1:    "FIN-WAIT-1",
	TCP_FIN_WAIT2:    "FIN-WAIT-2",
	TCP_TIME_WAIT:    "TIME-WAIT",
	TCP_CLOSE:        "CLOSED",
	TCP_CLOSE_WAIT:   "CLOSE-WAIT",
	TCP_LAST_ACK:     "LAST-ACK",
	TCP_LISTEN:       "LISTEN",
	TCP_CLOSING:      "CLOSING",
	TCP_NEW_SYN_RECV: "SYN-RECEIVED",
}

// GetAllPorts gets all used ports in config files on a specific remote
// and also all listening ports on the same host. Returns a map of port
// to bool "true" for each lookup.
func GetAllPorts(h *geneos.Host) (ports map[uint16]bool) {
	if h == geneos.ALL {
		log.Fatal().Msg("getports() called with all hosts")
	}
	ports = make(map[uint16]bool)
	for _, c := range Instances(h, nil) {
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

// NextFreePort returns the next available (unallocated and unused) TCP
// listening port for component ct on host h.
//
// The range of ports available for a component is defined in the
// configuration for the user and for each component type. A port is
// available if it is neither allocated to any other instance on the
// same host (of any component type) and also is not in use by any other
// process which may not be a Geneos instance.
//
// Each range is a comma separated list of single port number, e.g.
// "7036", a min-max inclusive range, e.g. "7036-8036" or a 'start-'
// open ended range, e.g. "7041-". Ranges can also be denoted by
// double-dot in addition to single dashes '-'.
//
// some limits based on
// https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers
//
// not concurrency safe at this time
func NextFreePort(h *geneos.Host, ct *geneos.Component) uint16 {
	log.Debug().Msgf("looking for %s, default %s", ct.PortRange, ct.ConfigAliases[ct.PortRange])
	from := config.GetString(ct.PortRange, config.Default(ct.ConfigAliases[ct.PortRange]))
	used := GetAllPorts(h)
	for p := range strings.SplitSeq(from, ",") {
		// split on dash or ".."
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
	if strings.Contains(h.ServerVersion(), "windows") {
		return errors.ErrUnsupported
	}

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
				if len(fields) < 10 || fields[3] != "0A" { // TCP_LISTEN
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
	intports := ListeningPorts(i)
	if len(intports) == 0 {
		return
	}
	for _, p := range intports {
		ports = append(ports, fmt.Sprint(p))
	}
	return
}

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

var sockRE = regexp.MustCompile(`socket:\[(\d+)\]`)

// SocketToConn takes the name of a socket from destination of a
// `/proc/.../fd` link and locates the corresponding connection in one
// of `/proc/net/tcp[6]` or `/proc/net/ucp[6]`. socket should be of the
// form `socket:[17126174]`
func SocketToConn(h *geneos.Host, socket string) (sc *SocketConnection, err error) {
	lx := sockRE.FindStringSubmatch(socket)
	if len(lx) < 2 {
		return
	}
	sockInode := lx[1]

	sc = &SocketConnection{}

	for _, source := range tcpfiles {
		tcp, err := h.Open(source)
		if err != nil {
			continue
		}

		var found bool
		var fields []string

		scanner := bufio.NewScanner(tcp)
		if scanner.Scan() {
			// skip headers
			_ = scanner.Text()
			for scanner.Scan() {
				line := scanner.Text()
				fields = strings.Fields(line)
				if len(fields) > 8 && fields[9] == sockInode {
					// found
					found = true
					break
				}
			}
		}

		if !found {
			continue
		}

		sc.Protocol = path.Base(source)

		l, r, st, q := fields[1], fields[2], fields[3], fields[4]

		laddr, lport, _ := strings.Cut(l, ":")
		raddr, rport, _ := strings.Cut(r, ":")

		lx, err2 := hex.DecodeString(laddr)
		for i, j := 0, len(lx)-1; i < j; i, j = i+1, j-1 {
			lx[i], lx[j] = lx[j], lx[i]
		}

		if err2 != nil {
			log.Error().Err(err2).Msg("")
			return sc, err2
		}
		sc.LocalAddr = net.IP(lx)
		fmt.Sscanf(lport, "%X", &sc.LocalPort)

		rx, err2 := hex.DecodeString(raddr)
		for i, j := 0, len(rx)-1; i < j; i, j = i+1, j-1 {
			rx[i], rx[j] = rx[j], rx[i]
		}

		if err2 != nil {
			log.Error().Err(err2).Msg("")
			return sc, err2
		}
		sc.RemoteAddr = net.IP(rx)
		fmt.Sscanf(rport, "%X", &sc.RemotePort)

		fmt.Sscanf(q, "%X:%X", &sc.TxQueue, &sc.RxQueue)
		var state int
		fmt.Sscanf(st, "%X", &state)
		sc.Status = stateNames[state]

		return sc, nil
	}

	return
}

type SocketConnection struct {
	Protocol   string
	LocalAddr  net.IP
	LocalPort  uint16
	RemoteAddr net.IP
	RemotePort uint16
	TxQueue    int64
	RxQueue    int64
	Status     string
}
