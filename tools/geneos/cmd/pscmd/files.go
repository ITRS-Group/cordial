package pscmd

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

type psInstanceFiles struct {
	psCommon
	FD       int         `json:"fd"`
	FDPerms  string      `json:"fd_perms"`
	Perms    fs.FileMode `json:"perms"`
	Username string      `json:"username"`
	Group    string      `json:"group"`
	Size     int64       `json:"size"`
	ModTime  time.Time   `json:"mod_time"`
	Path     string      `json:"path"`
}

var fileToolkitColumns = []string{
	"ID",
	"type",
	"name",
	"host",
	"pid",
	"fd",
	"permissions",
	"user",
	"group",
	"size",
	"lastModified",
	"path",
}

//				"Type\tName\tHost\tPID\tFD\tPerms\tUser:Group\tSize\tLast Modified\tPath\n",

var fileCSVColumns = []string{
	"Type",
	"Name",
	"Host",
	"PID",
	"FD",
	"Permissions",
	"User",
	"Group",
	"Size",
	"Last Modified",
	"Path",
}

var fileCSVHeader = strings.Join(fileCSVColumns, "\t")

func psFilesJSON(i geneos.Instance, pid int) (files []psInstanceFiles, err error) {
	ct := i.Type()
	h := i.Host()
	name := i.Name()

	homedir := i.Home()
	hs, err := h.Stat(homedir)
	if err != nil {
		return
	}
	uid, gid := host.GetFileOwner(h, hs)

	openFiles := process.OpenFiles(h, pid)
	files = make([]psInstanceFiles, 0, len(openFiles)+1)

	files = append(files, psInstanceFiles{
		psCommon: psCommon{
			Type: ct,
			Name: name,
			Host: h,
			PID:  pid,
		},
		FD:       -1,
		Perms:    hs.Mode().Perm(),
		Username: process.GetUsername(uid),
		Group:    process.GetGroupname(gid),
		Size:     hs.Size(),
		ModTime:  hs.ModTime(),
		Path:     homedir,
	})

	for _, fd := range openFiles {
		if path.IsAbs(fd.Path) {
			uid, gid := host.GetFileOwner(h, fd.Stat)
			path := fd.Path
			fdPerm := ""
			m := fd.Lstat.Mode().Perm()
			if m&0400 == 0400 {
				fdPerm += "r"
			}
			if m&0200 == 0200 {
				fdPerm += "w"
			}
			files = append(files, psInstanceFiles{
				psCommon: psCommon{
					Type: ct,
					Name: name,
					Host: h,
					PID:  pid,
				},
				FD:       fd.FD,
				FDPerms:  fdPerm,
				Perms:    fd.Stat.Mode().Perm(),
				Username: process.GetUsername(uid),
				Group:    process.GetGroupname(gid),
				Size:     fd.Stat.Size(),
				ModTime:  fd.Stat.ModTime(),
				Path:     path,
			})
		}
	}

	return
}

func psFilesCSV(i geneos.Instance, pid int, resp *responses.General) (err error) {
	ct := i.Type()
	h := i.Host()
	name := i.Name()

	homedir := i.Home()
	hs, err := h.Stat(homedir)
	if err != nil {
		resp.Err = err
		return
	}
	uid, gid := host.GetFileOwner(h, hs)

	var row []string
	if psCmdToolkit {
		row = append(row, instance.IDString(i))
	}

	row = append(row,
		ct.String(),
		name,
		h.String(),
		fmt.Sprint(pid),
		"cwd",
		hs.Mode().Perm().String(),
		process.GetUsername(uid),
		process.GetGroupname(gid),
		fmt.Sprint(hs.Size()),
		hs.ModTime().Local().Format(time.RFC3339),
		homedir,
	)
	resp.Dataview.Table = append(resp.Dataview.Table, row)

	for _, fd := range process.OpenFiles(h, pid) {
		if path.IsAbs(fd.Path) {
			uid, gid := host.GetFileOwner(h, fd.Stat)
			path := fd.Path
			fdPerm := ""
			m := fd.Lstat.Mode().Perm()
			if m&0400 == 0400 {
				fdPerm += "r"
			}
			if m&0200 == 0200 {
				fdPerm += "w"
			}
			var row []string
			if psCmdToolkit {
				row = append(row, instance.IDString(i)+" # "+fmt.Sprint(fd.FD))
			}

			row = append(row,
				ct.String(),
				name,
				h.String(),
				fmt.Sprint(pid),
				fmt.Sprintf("%d:%s", fd.FD, fdPerm),
				fd.Stat.Mode().Perm().String(),
				process.GetUsername(uid),
				process.GetGroupname(gid),
				fmt.Sprint(fd.Stat.Size()),
				fd.Stat.ModTime().Local().Format(time.RFC3339),
				path,
			)
			resp.Dataview.Table = append(resp.Dataview.Table, row)
		}
	}

	return
}

func psFilesTable(i geneos.Instance, pid int, resp *responses.General) (err error) {
	ct := i.Type()
	h := i.Host()
	name := i.Name()

	pi, _, _, _, err := psInstanceCommon(i)
	if err != nil {
		return
	}

	homedir := pi.Cwd
	hs, err := h.Stat(homedir)
	if err != nil {
		resp.Err = err
		return
	}
	uid, gid := host.GetFileOwner(h, hs)
	resp.Dataview.Table = append(resp.Dataview.Table, []string{
		ct.String(),
		name,
		h.String(),
		fmt.Sprint(pid),
		"cwd",
		hs.Mode().Perm().String(),
		process.GetUsername(uid),
		process.GetGroupname(gid),
		fmt.Sprint(hs.Size()),
		hs.ModTime().Local().Format(time.RFC3339),
		homedir,
	})

	for _, fd := range process.OpenFiles(h, pid) {
		if !path.IsAbs(fd.Path) {
			continue
		}

		uid, gid := host.GetFileOwner(h, fd.Stat)
		path := fd.Path
		fdPerm := ""
		m := fd.Lstat.Mode().Perm()
		if m&0400 == 0400 {
			fdPerm += "r"
		}
		if m&0200 == 0200 {
			fdPerm += "w"
		}
		resp.Dataview.Table = append(resp.Dataview.Table, []string{
			ct.String(),
			name,
			h.String(),
			fmt.Sprint(pid),
			fmt.Sprintf("%d:%s", fd.FD, fdPerm),
			fd.Stat.Mode().Perm().String(),
			process.GetUsername(uid),
			process.GetGroupname(gid),
			fmt.Sprint(fd.Stat.Size()),
			fd.Stat.ModTime().Local().Format(time.RFC3339),
			path,
		})
	}

	if capi, ok, err := checkCA(i, pi.Children); err == nil && ok {
		i.Log().Debug("pid has CA child process", slog.Int("parent_pid", pi.PID), slog.Int("child_pid", capi.PID))
		homedir := capi.Cwd
		hs, err := h.Stat(homedir)
		if err != nil {
			resp.Err = err
			return err
		}
		uid, gid := host.GetFileOwner(h, hs)
		resp.Dataview.Table = append(resp.Dataview.Table, []string{
			ct.String() + "/ca",
			name,
			h.String(),
			fmt.Sprint(capi.PID),
			"cwd",
			hs.Mode().Perm().String(),
			process.GetUsername(uid),
			process.GetGroupname(gid),
			fmt.Sprint(hs.Size()),
			hs.ModTime().Local().Format(time.RFC3339),
			homedir,
		})

		for _, fd := range capi.OpenFiles {
			if !path.IsAbs(fd.Path) {
				continue
			}

			uid, gid := host.GetFileOwner(h, fd.Stat)
			path := fd.Path
			fdPerm := ""
			m := fd.Lstat.Mode().Perm()
			if m&0400 == 0400 {
				fdPerm += "r"
			}
			if m&0200 == 0200 {
				fdPerm += "w"
			}
			resp.Dataview.Table = append(resp.Dataview.Table, []string{
				ct.String() + "/ca",
				name,
				h.String(),
				fmt.Sprint(capi.PID),
				fmt.Sprintf("%d:%s", fd.FD, fdPerm),
				fd.Stat.Mode().Perm().String(),
				process.GetUsername(uid),
				process.GetGroupname(gid),
				fmt.Sprint(fd.Stat.Size()),
				fd.Stat.ModTime().Local().Format(time.RFC3339),
				path,
			})
		}
	}

	return
}
