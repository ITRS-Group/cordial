package pscmd

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
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

func psFilesJSON(i geneos.Instance, pid int, resp *responses.Response) (err error) {
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

	files := []psInstanceFiles{}

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

	resp.Value = files
	return
}

func psFilesCSV(i geneos.Instance, pid int, resp *responses.Response) (err error) {
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
	resp.Rows = append(resp.Rows, row)

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
			resp.Rows = append(resp.Rows, row)

		}
	}

	return
}

func psFilesTable(i geneos.Instance, pid int, resp *responses.Response) (err error) {
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
	resp.Details = append(resp.Details,
		fmt.Sprintf("%s\t%s\t%s\t%d\tcwd\t%s\t%s\t%s\t%d\t%s\t%s",
			ct,
			name,
			h,
			pid,
			hs.Mode().Perm().String(),
			process.GetUsername(uid),
			process.GetGroupname(gid),
			hs.Size(),
			hs.ModTime().Local().Format(time.RFC3339),
			homedir,
		))
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
		resp.Details = append(resp.Details,
			fmt.Sprintf("%s\t%s\t%s\t%d\t%d:%s\t%s\t%s\t%s\t%d\t%s\t%s",
				ct,
				name,
				h,
				pid,
				fd.FD,
				fdPerm,
				fd.Stat.Mode().Perm(),
				process.GetUsername(uid),
				process.GetGroupname(gid),
				fd.Stat.Size(),
				fd.Stat.ModTime().Local().Format(time.RFC3339),
				path,
			))
	}

	return
}
