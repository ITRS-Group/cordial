//go:build !windows

package process

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"

	"github.com/itrs-group/cordial/pkg/host"
)

// OpenFiles returns a map of file descriptor to file details for all files
// for the instance. An empty map is returned if the process cannot be
// found.
//
// For descriptors that do not link to a path, such as pipes and
// sockets, Lstat describes the link and Stat describes what it refers to
// if that can be determined, otherwise it is the same as Lstat.
// Sockets that are TCP connections have Conn set.
func OpenFiles(h host.Host, pid int) (files []ProcessFDs) {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := h.ReadDir(fdDir)
	if err != nil {
		return
	}
	slices.SortFunc(fds, func(a, b fs.DirEntry) int {
		an, _ := strconv.Atoi(a.Name())
		bn, _ := strconv.Atoi(b.Name())
		return cmp.Compare(an, bn)
	})

	// the connection tables are read once, and only if needed, rather
	// than for each socket
	var sockets map[string]*SocketConnection
	var socketsRead bool

	for _, ent := range fds {
		fd := ent.Name()
		n, _ := strconv.Atoi(fd)

		fdPath := path.Join(fdDir, fd)

		linkVal, err := h.Readlink(fdPath)
		if err != nil {
			continue
		}

		linkStat, err := h.Lstat(fdPath)
		if err != nil {
			continue
		}

		destStat, err := h.Stat(fdPath)
		if err != nil {
			if path.IsAbs(linkVal) {
				continue
			}
			// pipes, sockets and the like, which may not be statable
			destStat = linkStat
		}

		pfd := ProcessFDs{
			PID:   pid,
			FD:    n,
			Path:  linkVal,
			Lstat: linkStat,
			Stat:  destStat,
		}

		if !path.IsAbs(linkVal) {
			if inode, ok := socketInode(linkVal); ok {
				if !socketsRead {
					// if the tables cannot be read there are no connections to show
					sockets, _ = socketTable(h)
					socketsRead = true
				}
				pfd.Conn = sockets[inode]
			}
		}

		files = append(files, pfd)
	}
	return
}
