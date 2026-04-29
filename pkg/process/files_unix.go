//go:build !windows

package process

import (
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
func OpenFiles(h host.Host, pid int) (files []ProcessFDs) {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := h.ReadDir(fdDir)
	if err != nil {
		return
	}
	slices.SortFunc(fds, func(a, b fs.DirEntry) int {
		an, _ := strconv.Atoi(a.Name())
		bn, _ := strconv.Atoi(b.Name())
		if an < bn {
			return -1
		}
		if an == bn {
			return 0
		}
		return 1
	})

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

		if !path.IsAbs(linkVal) {
			// check for socket
			sc, err := SocketToConn(h, linkVal)
			if err != nil {
				continue
			}

			files = append(files, ProcessFDs{
				PID:  pid,
				FD:   n,
				Path: linkVal,
				Stat: linkStat,
				Conn: sc,
			})

			continue
		}

		destStat, err := h.Stat(fdPath)
		if err != nil {
			continue
		}

		files = append(files, ProcessFDs{
			PID:   pid,
			FD:    n,
			Path:  linkVal,
			Lstat: linkStat,
			Stat:  destStat,
		})
	}
	return
}
