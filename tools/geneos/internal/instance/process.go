package instance

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func Ports(c geneos.Instance) (ports []int) {
	links := Files(c)

	tcp, err := c.Host().Open("/proc/net/tcp")
	if err != nil {
		log.Fatalln(err)
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
			logDebug.Printf("ip %v port %v inode %v", ip, port, inode)
			tcpports[inode] = port
		}
	}

	var inode int
	for _, l := range links {
		if n, err := fmt.Sscanf(l, "socket:[%d]", &inode); err == nil && n == 1 {
			if port, ok := tcpports[inode]; ok {
				ports = append(ports, port)
				logDebug.Printf("process listening on %v", port)
			}
		}
	}
	return
}

func Files(c geneos.Instance) (links map[int]string) {
	links = make(map[int]string)
	pid, err := GetPID(c)
	if err != nil {
		log.Fatalln(err)
	}
	path := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := c.Host().ReadDir(path)
	if err != nil {
		log.Fatalln(err)
	}
	for _, ent := range fds {
		fd := ent.Name()
		dest, err := c.Host().Readlink(filepath.Join(path, fd))
		if err != nil {
			log.Fatalln(err)
		}
		n, _ := strconv.Atoi(fd)
		links[n] = dest
		logDebug.Printf("\tfd %s points to %q", fd, dest)
	}
	return
}
