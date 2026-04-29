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

package cmd

import (
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

var psCmdLong, psCmdShowFiles, psCmdShowNet, psCmdJSON, psCmdIndent, psCmdCSV, psCmdToolkit bool

func init() {
	GeneosCmd.AddCommand(psCmd)

	psCmd.Flags().BoolVarP(&psCmdShowFiles, "files", "f", false, "Show open files")
	psCmd.Flags().BoolVarP(&psCmdShowNet, "network", "n", false, "Show TCP sockets")

	psCmd.Flags().BoolVarP(&psCmdLong, "long", "l", false, "Show more output (remote ports etc.)")

	psCmd.Flags().BoolVarP(&psCmdJSON, "json", "j", false, "Output JSON")
	psCmd.Flags().BoolVarP(&psCmdIndent, "pretty", "i", false, "Output indented JSON")
	psCmd.Flags().BoolVarP(&psCmdCSV, "csv", "c", false, "Output CSV")
	psCmd.Flags().BoolVarP(&psCmdToolkit, "toolkit", "t", false, "Output Toolkit formatted CSV")

	psCmd.Flags().SortFlags = false
}

//go:embed _docs/ps.md
var psCmdDescription string

var psCmd = &cobra.Command{
	Use:          "ps [flags] [TYPE] [NAMES...]",
	GroupID:      CommandGroupView,
	Short:        "List Running Instance Details",
	Long:         psCmdDescription,
	Aliases:      []string{"status"},
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:               "true",
		CmdRequireHome:          "true",
		CmdWildcardNames:        "true",
		CmdAllowRoot:            "true",
		CmdNonInstanceArgsError: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names, params, err := FetchArgs(cmd)
		if err != nil {
			return
		}
		CommandPS(ct, names, params)
	},
}

type psInstanceNetwork struct {
	Type       *geneos.Component `json:"type"`
	Name       string            `json:"name"`
	Host       *geneos.Host      `json:"host"`
	PID        int               `json:"pid"`
	FD         int               `json:"fd"`
	Protocol   string            `json:"protocol"`
	LocalAddr  net.IP            `json:"local_addr"`
	LocalPort  uint16            `json:"local_port"`
	RemoteAddr net.IP            `json:"remote_addr"`
	RemotePort uint16            `json:"remote_port"`
	Status     string            `json:"status"`
	TXQueue    int64             `json:"tx_queue"`
	RXQueue    int64             `json:"rx_queue"`
}

var netToolkitColumns = []string{
	"ID",
	"type",
	"name",
	"host",
	"pid",
	"fd",
	"protocol",
	"localaddr",
	"localport",
	"remoteaddr",
	"remoteport",
	"status",
	"txqueue",
	"rxqueue",
}

var netCSVColumns = []string{
	"Type",
	"Name",
	"Host",
	"PID",
	"FD",
	"Protocol",
	"Local Addr",
	"Local Port",
	"Remote Addr",
	"Remote Port",
	"Status",
	"TXQueue",
	"RXQueue",
}

type psInstanceFiles struct {
	Type     *geneos.Component `json:"type"`
	Name     string            `json:"name"`
	Host     *geneos.Host      `json:"host"`
	PID      int               `json:"pid"`
	FD       int               `json:"fd"`
	FDPerms  string            `json:"fd_perms"`
	Perms    fs.FileMode       `json:"perms"`
	Username string            `json:"username"`
	Group    string            `json:"group"`
	Size     int64             `json:"size"`
	ModTime  time.Time         `json:"mod_time"`
	Path     string            `json:"path"`
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

type psInstance struct {
	Type      *geneos.Component `json:"type"`
	Name      string            `json:"name"`
	Host      *geneos.Host      `json:"host"`
	PID       string            `json:"pid"`
	Ports     []int             `json:"ports,omitempty"`
	User      string            `json:"user,omitempty"`
	Group     string            `json:"group,omitempty"`
	Starttime time.Time         `json:"starttime,omitempty"`
	Version   string            `json:"version,omitempty"`
	Home      string            `json:"home,omitempty"`

	// Extra fields, when `--long` is used
	Extra *process.ProcessStats `json:"extra,omitempty"`
}

var instanceToolkitColumns = []string{
	"ID",
	"type",
	"name",
	"host",
	"pid",
	"ports",
	"user",
	"group",
	"startTime",
	"version",
	"home",
}

var instanceCSVColumns = []string{
	"Type",
	"Name",
	"Host",
	"PID",
	"Ports",
	"User",
	"Group",
	"Starttime",
	"Version",
	"Home",
}

var instanceToolkitExtraColumns = []string{
	"state",
	"threads",
	"openfiles",
	"opensockets",
	"residentSetSize",
	"residentSetSizeAnon",
	"residentSetSizeMax",
	"totalUserTime",
	"totalKernelTime",
	"totalChildUserTime",
	"totalChildKernelTime",
}

var instanceCSVExtraColumns = []string{
	"State",
	"Threads",
	"Open Files",
	"Open Sockets",
	"RSS",
	"RSSAnon",
	"RSSMax",
	"TotalUserTime",
	"TotalKernelTime",
	"TotalChildUserTime",
	"TotalChildKernelTime",
}

// CommandPS writes running instance information to STDOUT
func CommandPS(ct *geneos.Component, names []string, params []string) {
	switch {
	case psCmdJSON, psCmdIndent:
		instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceJSON).Report(os.Stdout, responses.IndentJSON(psCmdIndent))
	case psCmdToolkit:
		psCSVWriter := csv.NewWriter(os.Stdout)

		var columns []string

		switch {
		case psCmdShowNet:
			columns = netToolkitColumns
		case psCmdShowFiles:
			columns = fileToolkitColumns
		default:
			columns = instanceToolkitColumns
			if psCmdLong {
				columns = append(columns, instanceToolkitExtraColumns...)
			}
		}

		psCSVWriter.Write(columns)
		resp := instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceCSV)
		resp.Report(psCSVWriter, responses.IgnoreErr(geneos.ErrDisabled))
		switch {
		case psCmdShowNet:
		case psCmdShowFiles:
		default:
			var notRunning int
			var disabled int
			for _, r := range resp {
				if errors.Is(r.Err, os.ErrProcessDone) {
					notRunning++
				}
				if errors.Is(r.Err, geneos.ErrDisabled) {
					disabled++
				}
			}
			fmt.Printf("<!>instances,%d\n", len(resp))
			fmt.Printf("<!>running,%d\n", len(resp)-notRunning-disabled)
			fmt.Printf("<!>notRunning,%d\n", notRunning)
			fmt.Printf("<!>disabled,%d\n", disabled)
		}
	case psCmdCSV:
		var columns []string
		switch {
		case psCmdShowNet:
			columns = netCSVColumns
		case psCmdShowFiles:
			columns = fileCSVColumns
		default:
			columns = instanceCSVColumns
			if psCmdLong {
				columns = append(columns, instanceCSVExtraColumns...)
			}
		}

		psCSVWriter := csv.NewWriter(os.Stdout)
		psCSVWriter.Write(columns)
		instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceCSV).Report(psCSVWriter)
	default:
		psTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		if psCmdShowNet {
			fmt.Fprintln(psTabWriter, strings.Join(netCSVColumns, "\t"))
		} else if psCmdShowFiles {
			fmt.Fprintf(psTabWriter,
				"Type\tName\tHost\tPID\tFD\tPerms\tUser:Group\tSize\tLast Modified\tPath\n",
			)
		} else if psCmdLong {
			fmt.Fprintf(psTabWriter, "Type\tName\tHost\tPID\tPorts\tUser\tGroup\tStarttime\tVersion\tHome\tState\tThreads\tOpen Files\tOpen Sockets\tRSS\tRSSAnon\tRSSMax\tTotalUserTime\tTotalKernelTime\tTotalChildUserTime\tTotalChildKernelTime\n")
		} else {
			fmt.Fprintln(psTabWriter, strings.Join(instanceCSVColumns, "\t"))
		}
		instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceTable).Report(psTabWriter)
	}
}

func psInstanceCommon(i geneos.Instance) (pid int, username, groupname string, mtime time.Time, base, actual, uptodate string, ports []int, children []int, err error) {
	h := i.Host()

	if instance.IsDisabled(i) {
		err = geneos.ErrDisabled
		return
	}
	pi, err := instance.GetProcessInfo(i)
	if err != nil {
		if !errors.Is(err, os.ErrProcessDone) {
			log.Debug().Err(err).Msgf("failed to get PID info for instance %s", i.Name())
		}
		return
	}

	pid = pi.PID
	username = process.GetUsername(pi.UID)
	groupname = process.GetGroupname(pi.GID)
	mtime = pi.CreationTime
	children = pi.Children

	base, underlying, actual, _ := instance.LiveVersion(i, pid)
	if pkgtype := config.Get[string](i.Config(), "pkgtype"); pkgtype != "" {
		base = path.Join(pkgtype, base)
	}
	uptodate = "="
	if underlying != actual {
		uptodate = "<>"
	}

	if h.IsLocal() || psCmdLong {
		ports = process.ListeningPorts(h, pid)
	}

	return
}

func checkCA(h *geneos.Host, pid int) (pi process.ProcessInfo, ok bool, err error) {
	pi, err = process.GetProcessInfo(h, pid, false)
	if err != nil {
		return
	}
	if len(pi.Cmdline) == 0 {
		err = fmt.Errorf("no cmdline for PID %d", pid)
		return
	}

	if path.Base(pi.Cmdline[0]) != "java" {
		return
	}

	for _, arg := range pi.Cmdline[1:] {
		if strings.Contains(arg, "collection-agent") {
			ok = true
			return
		}
	}

	return
}

func psInstanceTable(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	h := i.Host()
	ct := i.Type()
	name := i.Name()

	pid, username, groupname, mtime, base, actual, uptodate, ports, children, err := psInstanceCommon(i)
	if err != nil {
		return
	}

	if psCmdShowNet {
		for _, fd := range process.OpenFiles(h, pid) {
			if fd.Conn == nil {
				continue
			}

			// socket
			c := fd.Conn
			if !(strings.HasPrefix(c.Protocol, "tcp") || strings.HasPrefix(c.Protocol, "udp")) {
				continue
			}
			remAddr := "-"
			if !c.RemoteAddr.Equal(net.IPv4(0, 0, 0, 0)) {
				remAddr = fmt.Sprint(c.RemoteAddr)
			}
			remPort := "-"
			if c.RemotePort != 0 {
				remPort = fmt.Sprint(c.RemotePort)
			}
			resp.Details = append(resp.Details,
				fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%s\t%s\t%d\t%s\t%s\t%s\t%d\t%d",
					ct,
					name,
					h,
					pid,
					fd.FD,
					c.Protocol,
					c.LocalAddr,
					c.LocalPort,
					remAddr,
					remPort,
					c.Status,
					c.TxQueue,
					c.RxQueue,
				))
		}
		return
	}

	// file
	if psCmdShowFiles {
		homedir := i.Home()
		hs, err := h.Stat(homedir)
		if err != nil {
			resp.Err = err
			return
		}
		uid, gid := host.GetFileOwner(h, hs)
		resp.Details = append(resp.Details,
			fmt.Sprintf("%s\t%s\t%s\t%d\tcwd\t%s\t%s:%s\t%d\t%s\t%s",
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
				fmt.Sprintf("%s\t%s\t%s\t%d\t%d:%s\t%s\t%s:%s\t%d\t%s\t%s",
					ct,
					name,
					h,
					pid,
					fd.FD,
					fdPerm,
					fd.Stat.Mode().Perm().String(),
					process.GetUsername(uid),
					process.GetGroupname(gid),
					fd.Stat.Size(),
					fd.Stat.ModTime().Local().Format(time.RFC3339),
					path,
				))
		}

		return
	}

	var portlist string
	portsString := []string{}
	for _, p := range ports {
		portsString = append(portsString, fmt.Sprint(p))
	}
	portlist = strings.Join(portsString, ",")
	if !h.IsLocal() && portlist == "" {
		portlist = "..."
	}

	p := &process.ProcessStats{}
	if err := process.ProcessStatus(h, pid, p); err == nil && psCmdLong {
		resp.Summary = fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s%s%s\t%s\t%s\t%d\t%d\t%d\t%.2f MiB\t%.2f MiB\t%.2f MiB\t%.2f s\t%.2f s\t%.2f s\t%.2f s",
			ct,
			name,
			h,
			pid,
			portlist,
			username,
			groupname,
			mtime.Local().Format(time.RFC3339),
			base,
			uptodate,
			actual,
			i.Home(),
			p.State,
			p.Threads,
			p.OpenFiles,
			p.OpenSockets,
			float64(p.VmRSS)/(1024*1024),
			float64(p.RssAnon)/(1024*1024),
			float64(p.VmHWM)/(1024*1024),
			p.Utime.Seconds(),
			p.Stime.Seconds(),
			p.CUtime.Seconds(),
			p.CStime.Seconds(),
		)
	} else {
		resp.Summary = fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s%s%s\t%s",
			ct,
			name,
			h,
			pid,
			portlist,
			username,
			groupname,
			mtime.Local().Format(time.RFC3339),
			base,
			uptodate,
			actual,
			i.Home(),
		)

		// look for a collection agent child process
		if len(children) > 0 && ct.IsA("netprobe") {
			// check for CAs and list, but ignore other child processes (for now)
			for _, childPID := range children {
				if pi, ok, err := checkCA(h, childPID); err == nil && ok {
					// list it
					caports := process.ListeningPorts(h, childPID)
					var portlist string
					portsString := []string{}
					for _, p := range caports {
						portsString = append(portsString, fmt.Sprint(p))
					}
					portlist = strings.Join(portsString, ",")
					if !h.IsLocal() && portlist == "" {
						portlist = "..."
					}
					resp.Details = append(resp.Details,
						fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s%s%s\t%s",
							ct.String()+"/ca",
							name,
							h,
							pi.PID,
							portlist,
							username,
							groupname,
							pi.CreationTime.Local().Format(time.RFC3339),
							base,
							uptodate,
							actual,
							i.Home(),
						),
					)
				}
			}
		}
	}

	return
}

func psInstanceCSV(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	h := i.Host()
	ct := i.Type()
	name := i.Name()

	pid, username, groupname, mtime, base, actual, uptodate, ports, _, err := psInstanceCommon(i)
	if err != nil {
		resp.Err = err
		return
	}

	if psCmdShowNet {
		var row []string
		if psCmdToolkit {
			row = append(row, instance.IDString(i))
		}

		row = append(row,
			ct.String(),
			name,
			h.String(),
			fmt.Sprint(pid),
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
		)
		resp.Rows = append(resp.Rows, row)

		for _, fd := range process.OpenFiles(h, pid) {
			if fd.Conn != nil {
				// socket
				c := fd.Conn
				if !(strings.HasPrefix(c.Protocol, "tcp") || strings.HasPrefix(c.Protocol, "udp")) {
					continue
				}
				remAddr := "-"
				if !c.RemoteAddr.Equal(net.IPv4(0, 0, 0, 0)) {
					remAddr = fmt.Sprint(c.RemoteAddr)
				}
				remPort := "-"
				if c.RemotePort != 0 {
					remPort = fmt.Sprint(c.RemotePort)
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
					fmt.Sprint(fd.FD),
					c.Protocol,
					c.LocalAddr.String(),
					fmt.Sprint(c.LocalPort),
					remAddr,
					remPort,
					c.Status,
					fmt.Sprint(c.TxQueue),
					fmt.Sprint(c.RxQueue),
				)
				resp.Rows = append(resp.Rows, row)
			}
		}
		return
	}

	// file
	if psCmdShowFiles {
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

	var portlist string
	portsString := []string{}
	for _, p := range ports {
		portsString = append(portsString, fmt.Sprint(p))
	}
	portlist = strings.Join(portsString, " ")
	if !h.IsLocal() && portlist == "" {
		portlist = "..."
	}

	var row []string

	if psCmdToolkit {
		row = append(row, instance.IDString(i))
	}

	row = append(row,
		ct.String(),
		name,
		h.String(),
		fmt.Sprint(pid),
		portlist,
		username,
		groupname,
		mtime.Local().Format(time.RFC3339),
		fmt.Sprintf("%s%s%s", base, uptodate, actual),
		i.Home(),
	)

	if psCmdLong {
		p := &process.ProcessStats{}
		if err := process.ProcessStatus(h, pid, p); err == nil {
			row = append(row,
				p.State,
				fmt.Sprint(p.Threads),
				fmt.Sprint(p.OpenFiles),
				fmt.Sprint(p.OpenSockets),
				fmt.Sprintf("%.2f MiB", float64(p.VmRSS)/(1024*1024)),
				fmt.Sprintf("%.2f MiB", float64(p.RssAnon)/(1024*1024)),
				fmt.Sprintf("%.2f MiB", float64(p.VmHWM)/(1024*1024)),
				fmt.Sprintf("%.2f s", p.Utime.Seconds()),
				fmt.Sprintf("%.2f s", p.Stime.Seconds()),
				fmt.Sprintf("%.2f s", p.CUtime.Seconds()),
				fmt.Sprintf("%.2f s", p.CStime.Seconds()),
			)
		}
	}

	resp.Rows = append(resp.Rows, row)
	return
}

func psInstanceJSON(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	h := i.Host()
	ct := i.Type()
	name := i.Name()

	pid, username, groupname, mtime, base, actual, uptodate, ports, _, err := psInstanceCommon(i)
	if err != nil {
		return
	}

	if psCmdShowNet {
		conns := []psInstanceNetwork{}
		for _, fd := range process.OpenFiles(h, pid) {
			if fd.Conn != nil {
				// socket
				c := fd.Conn
				if !(strings.HasPrefix(c.Protocol, "tcp") || strings.HasPrefix(c.Protocol, "udp")) {
					continue
				}
				conns = append(conns, psInstanceNetwork{
					Type:       ct,
					Name:       name,
					Host:       h,
					PID:        pid,
					FD:         fd.FD,
					Protocol:   c.Protocol,
					LocalAddr:  c.LocalAddr,
					LocalPort:  c.LocalPort,
					RemoteAddr: c.RemoteAddr,
					RemotePort: c.RemotePort,
					Status:     c.Status,
					TXQueue:    c.TxQueue,
					RXQueue:    c.RxQueue,
				})
			}
		}
		resp.Value = conns
		return
	}

	if psCmdShowFiles {
		homedir := i.Home()
		hs, err := h.Stat(homedir)
		if err != nil {
			resp.Err = err
			return
		}
		uid, gid := host.GetFileOwner(h, hs)

		files := []psInstanceFiles{}

		files = append(files, psInstanceFiles{
			Type:     ct,
			Name:     name,
			Host:     h,
			PID:      pid,
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
					Type:     ct,
					Name:     name,
					Host:     h,
					PID:      pid,
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

	psData := psInstance{
		Type:      ct,
		Name:      name,
		Host:      h,
		PID:       fmt.Sprint(pid),
		Ports:     ports,
		User:      username,
		Group:     groupname,
		Starttime: mtime,
		Version:   fmt.Sprintf("%s%s%s", base, uptodate, actual),
		Home:      i.Home(),
	}

	if psCmdLong {
		psData.Extra = &process.ProcessStats{}
		process.ProcessStatus(h, pid, psData.Extra)
	}

	resp.Value = psData
	return
}
