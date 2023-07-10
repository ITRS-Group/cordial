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

// GetPorts gets all used ports in config files on a specific remote
// this will not work for ports assigned in component config files, such
// as gateway setup or netprobe collection agent
//
// returns a map
func GetPorts(r *geneos.Host) (ports map[uint16]*geneos.Component) {
	if r == geneos.ALL {
		log.Fatal().Msg("getports() call with all hosts")
	}
	ports = make(map[uint16]*geneos.Component)
	for _, c := range GetAll(r, nil) {
		if !c.Loaded() {
			log.Error().Msgf("cannot load configuration for %s", c)
			continue
		}
		if port := c.Config().GetInt("port"); port != 0 {
			ports[uint16(port)] = c.Type()
		}
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
func NextPort(r *geneos.Host, ct *geneos.Component) uint16 {
	from := config.GetString(ct.PortRange)
	used := GetPorts(r)
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
func ListeningPorts(c geneos.Instance) (ports []int) {
	var err error

	if !IsRunning(c) {
		return
	}

	sockets := sockets(c)
	if len(sockets) == 0 {
		return
	}

	tcpports := make(map[int]int) // key = socket inode
	if err = allTCPListenPorts(c.Host(), tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
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
func ListeningPortsStrings(c geneos.Instance) (ports []string) {
	var err error

	if !IsRunning(c) {
		return
	}

	sockets := sockets(c)
	if len(sockets) == 0 {
		return
	}

	tcpports := make(map[int]int) // key = socket inode
	if err = allTCPListenPorts(c.Host(), tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
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
func AllListeningPorts(h *geneos.Host, min, max int) (ports []int) {
	var err error

	tcpports := make(map[int]int) // key = socket inode, value port
	if err = allTCPListenPorts(h, tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Debug().Err(err).Msg("continuing")
	}
	for v := range tcpports {
		if min == -1 || v >= min {
			if max == -1 || v <= max {
				ports = append(ports, v)
			}
		}
	}
	sort.Ints(ports)

	return
}

// sockets returns a map[int]int of file descriptor to socket inode for all open
// files for the process running as the instance. An empty map is
// returned if the process cannot be found.
func sockets(c geneos.Instance) (links map[int]int) {
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
		}
	}
	return
}
