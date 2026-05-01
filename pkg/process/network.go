package process

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

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
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

var sockRE = regexp.MustCompile(`socket:\[(\d+)\]`)

// SocketToConn takes the name of a socket from destination of a
// `/proc/.../fd` link and locates the corresponding connection in one
// of `/proc/net/tcp[6]` or `/proc/net/ucp[6]`. socket should be of the
// form `socket:[17126174]`
func SocketToConn(h host.Host, socket string) (sc *SocketConnection, err error) {
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

// AllTCPListenPorts returns a map of inodes to ports for all listening
// TCP ports from the source (typically /proc/net/tcp or /proc/net/tcp6)
// on host h. Will only work on Linux hosts.
func AllTCPListenPorts(h host.Host, ports map[int]int) (err error) {
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
// cannot be found. The instance may be on remote host h.
func ListeningPorts(h host.Host, pid int) (ports []int) {
	var err error

	if pid == 0 {
		return
	}

	sockets := sockets(h, pid)
	if len(sockets) == 0 {
		return
	}

	tcpports := make(map[int]int) // key = socket inode
	if err = AllTCPListenPorts(h, tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error().Err(err).Msg("continuing")
	}

	for _, s := range sockets {
		if port, ok := tcpports[s]; ok {
			ports = append(ports, port)
		}
	}
	sort.Ints(ports)
	return
}

// sockets returns a map[int]int of file descriptor to socket inode for
// all open files for the process pid on host h. An empty map is
// returned if the process cannot be found.
func sockets(h host.Host, pid int) (links map[int]int) {
	var inode int
	links = make(map[int]int)
	if pid == 0 {
		return
	}
	file := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := h.ReadDir(file)
	if err != nil {
		return
	}
	for _, ent := range fds {
		fd := ent.Name()
		dest, err := h.Readlink(path.Join(file, fd))
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
