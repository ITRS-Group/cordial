package process

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
)

var tcpfiles = []string{
	"/proc/net/tcp",
	"/proc/net/tcp6",
}

// TODO: Revisit unused declaration udpfiles; restore it only when needed.
// var udpfiles = []string{
// 	"/proc/net/udp",
// 	"/proc/net/udp6",
// }

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

// SocketConnection TODO: Describe its purpose and intended use.
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

// socketInode returns the inode from the destination of a
// `/proc/.../fd` link of the form `socket:[17126174]`
func socketInode(link string) (inode string, ok bool) {
	inode, ok = strings.CutPrefix(link, "socket:[")
	if !ok {
		return "", false
	}
	inode, ok = strings.CutSuffix(inode, "]")
	if !ok || inode == "" {
		return "", false
	}
	return inode, true
}

// procNetEntry is one row of a `/proc/net/tcp[6]` file
type procNetEntry struct {
	SocketConnection
	State int
	Inode string
}

// decodeProcNetAddr decodes the `ADDRESS:PORT` columns of
// `/proc/net/tcp[6]`. The kernel prints each 32-bit word of an address
// in host byte order, so each word is reversed separately, for IPv4 and
// IPv6, which is for the little-endian hosts that are supported.
func decodeProcNetAddr(s string) (ip net.IP, port uint16, err error) {
	addr, p, found := strings.Cut(s, ":")
	if !found {
		return nil, 0, fmt.Errorf("invalid address %q", s)
	}
	b, err := hex.DecodeString(addr)
	if err != nil {
		return nil, 0, err
	}
	if len(b) != net.IPv4len && len(b) != net.IPv6len {
		return nil, 0, fmt.Errorf("invalid address length in %q", s)
	}
	for word := range slices.Chunk(b, 4) {
		slices.Reverse(word)
	}
	pv, err := strconv.ParseUint(p, 16, 16)
	if err != nil {
		return nil, 0, err
	}
	return net.IP(b), uint16(pv), nil
}

// parseProcNet reads the rows of a `/proc/net/tcp[6]` file, skipping the
// header and any row that cannot be understood. protocol is the name
// given to each entry.
func parseProcNet(r io.Reader, protocol string) (entries []procNetEntry, err error) {
	scanner := bufio.NewScanner(r)
	scanner.Scan() // skip headers
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		// sl, local, remote, st, tx:rx, tr:tm, retrnsmt, uid, timeout, inode
		if len(fields) < 10 {
			continue
		}

		e := procNetEntry{Inode: fields[9]}
		e.Protocol = protocol
		var err error
		if e.LocalAddr, e.LocalPort, err = decodeProcNetAddr(fields[1]); err != nil {
			continue
		}
		if e.RemoteAddr, e.RemotePort, err = decodeProcNetAddr(fields[2]); err != nil {
			continue
		}
		state, err := strconv.ParseUint(fields[3], 16, 8)
		if err != nil {
			continue
		}
		e.State = int(state)
		e.Status = stateNames[e.State]
		tx, rx, _ := strings.Cut(fields[4], ":")
		e.TxQueue, _ = strconv.ParseInt(tx, 16, 64)
		e.RxQueue, _ = strconv.ParseInt(rx, 16, 64)

		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

// readProcNet reads and parses source on host h and always closes it
func readProcNet(h host.Host, source string) (entries []procNetEntry, err error) {
	f, err := h.Open(source)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseProcNet(f, path.Base(source))
}

// socketTable returns the connections of all TCP sockets on host h, by
// socket inode. If none of the files could be read then the error is
// returned.
func socketTable(h host.Host) (table map[string]*SocketConnection, err error) {
	table = make(map[string]*SocketConnection)
	var errs []error
	var read int
	for _, source := range tcpfiles {
		entries, err := readProcNet(h, source)
		if err != nil && entries == nil {
			errs = append(errs, err)
			continue
		}
		read++
		for _, e := range entries {
			table[e.Inode] = &e.SocketConnection
		}
	}
	if read == 0 {
		return table, errors.Join(errs...)
	}
	return table, nil
}

// SocketToConn takes the name of a socket from destination of a
// `/proc/.../fd` link and locates the corresponding connection in one
// of `/proc/net/tcp[6]`. socket should be of the form
// `socket:[17126174]`. If socket is not of that form or there is no TCP
// connection for it then the returned connection is nil and the error
// is nil, unless the connection tables could not be read at all.
func SocketToConn(h host.Host, socket string) (sc *SocketConnection, err error) {
	inode, ok := socketInode(socket)
	if !ok {
		return nil, nil
	}
	table, err := socketTable(h)
	return table[inode], err
}

// AllTCPListenPorts returns a map of inodes to ports for all listening
// TCP ports from `/proc/net/tcp` and `/proc/net/tcp6` on host h. Will
// only work on Linux hosts. A host without IPv6 does not have the
// second of these, which is not an error.
func AllTCPListenPorts(h host.Host, ports map[int]int) (err error) {
	if h.OS() == "windows" {
		return errors.ErrUnsupported
	}

	var errs []error
	var read int
	for _, source := range tcpfiles {
		entries, err := readProcNet(h, source)
		if err != nil && entries == nil {
			errs = append(errs, err)
			continue
		}
		read++
		for _, e := range entries {
			if e.State != TCP_LISTEN {
				continue
			}
			if inode, err := strconv.Atoi(e.Inode); err == nil {
				ports[inode] = int(e.LocalPort)
			}
		}
	}
	if read == 0 {
		return errors.Join(errs...)
	}
	return nil
}

// ListeningPorts returns all TCP ports currently open for the process
// running as the instance. An empty slice is returned if the process
// cannot be found. The instance may be on remote host h.
func ListeningPorts(h host.Host, pid int) (ports []int) {
	if pid == 0 {
		return
	}

	sockets := sockets(h, pid)
	if len(sockets) == 0 {
		return
	}

	tcpports := make(map[int]int) // key = socket inode
	_ = AllTCPListenPorts(h, tcpports)

	for _, s := range sockets {
		if port, ok := tcpports[s]; ok {
			ports = append(ports, port)
		}
	}
	slices.Sort(ports)
	return
}

// sockets returns a map[int]int of file descriptor to socket inode for
// all open files for the process pid on host h. An empty map is
// returned if the process cannot be found.
func sockets(h host.Host, pid int) (links map[int]int) {
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
		if inode, ok := socketInode(dest); ok {
			if n, err := strconv.Atoi(inode); err == nil {
				f, _ := strconv.Atoi(fd)
				links[f] = n
			}
		}
	}
	return
}
