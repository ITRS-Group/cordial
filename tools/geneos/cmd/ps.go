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
	"crypto/tls"
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names, params := ParseTypeNamesParams(cmd)
		CommandPS(ct, names, params)
	},
}

// CommandPS writes running instance information to STDOUT
func CommandPS(ct *geneos.Component, names []string, params []string) {
	switch {
	case psCmdJSON, psCmdIndent:
		instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceJSON).Write(os.Stdout, instance.WriterIndent(psCmdIndent))
	case psCmdToolkit:
		psCSVWriter := csv.NewWriter(os.Stdout)

		var columns []string

		switch {
		case psCmdShowNet:
			columns = []string{
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
		case psCmdShowFiles:
			columns = []string{
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
		default:
			columns = []string{
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
			if psCmdLong {
				columns = append(columns,
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
				)
			}
		}
		psCSVWriter.Write(columns)
		resp := instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceCSV)
		resp.Write(psCSVWriter, instance.WriterIgnoreErr(geneos.ErrDisabled))
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
		psCSVWriter := csv.NewWriter(os.Stdout)
		columns := []string{
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
		if psCmdLong {
			columns = append(columns,
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
			)
		}
		psCSVWriter.Write(columns)
		instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceCSV).Write(psCSVWriter)
	default:
		psTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		if psCmdShowNet {
			fmt.Fprintf(psTabWriter,
				"Type\tName\tHost\tPID\tFD\tProtocol\tLocal Addr\tLocal Port\tRemote Addr\tRemote Port\tStatus\tTXQueue\tRxQueue\n",
			)
		} else if psCmdShowFiles {
			fmt.Fprintf(psTabWriter,
				"Type\tName\tHost\tPID\tFD\tPerms\tUser:Group\tSize\tLast Modified\tPath\n",
			)
		} else if psCmdLong {
			fmt.Fprintf(psTabWriter, "Type\tName\tHost\tPID\tPorts\tUser\tGroup\tStarttime\tVersion\tHome\tState\tThreads\tOpen Files\tOpen Sockets\tRSS\tRSSAnon\tRSSMax\tTotalUserTime\tTotalKernelTime\tTotalChildUserTime\tTotalChildKernelTime\n")
		} else {
			fmt.Fprintf(psTabWriter, "Type\tName\tHost\tPID\tPorts\tUser\tGroup\tStarttime\tVersion\tHome\n")
		}
		instance.Do(geneos.GetHost(Hostname), ct, names, psInstancePlain).Write(psTabWriter)
	}
}

func psInstanceCommon(i geneos.Instance) (pid int, username, groupname string, mtime time.Time, base, actual, uptodate string, ports []int, err error) {
	if instance.IsDisabled(i) {
		err = geneos.ErrDisabled
		return
	}
	pid, uid, gid, mtime, err := instance.GetPIDInfo(i)
	if err != nil {
		if !errors.Is(err, os.ErrProcessDone) {
			log.Debug().Err(err).Msgf("failed to get PID info for instance %s", i.Name())
		}
		return
	}

	username = geneos.GetUsername(uid)
	groupname = geneos.GetGroupname(gid)

	base, underlying, actual, _ := instance.LiveVersion(i, pid)
	if pkgtype := i.Config().GetString("pkgtype"); pkgtype != "" {
		base = path.Join(pkgtype, base)
	}
	uptodate = "="
	if underlying != actual {
		uptodate = "<>"
	}

	if i.Host().IsLocal() || psCmdLong {
		ports = instance.ListeningPorts(i)
	}

	return
}

func psInstancePlain(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	pid, username, groupname, mtime, base, actual, uptodate, ports, err := psInstanceCommon(i)
	if err != nil {
		return
	}

	if psCmdShowNet {
		for _, fd := range instance.Files(i) {
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
				resp.Lines = append(resp.Lines,
					fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%s\t%s\t%d\t%s\t%s\t%s\t%d\t%d",
						i.Type(),
						i.Name(),
						i.Host(),
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
		}
		return
	}

	// file
	if psCmdShowFiles {
		homedir := i.Home()
		hs, err := i.Host().Stat(homedir)
		if err != nil {
			resp.Err = err
			return
		}
		uid, gid := i.Host().GetFileOwner(hs)
		resp.Lines = append(resp.Lines,
			fmt.Sprintf("%s\t%s\t%s\tcwd\t%d\t%s\t%s:%s\t%d\t%s\t%s",
				i.Type(),
				i.Name(),
				i.Host(),
				pid,
				hs.Mode().Perm().String(),
				geneos.GetUsername(uid),
				geneos.GetGroupname(gid),
				hs.Size(),
				hs.ModTime().Local().Format(time.RFC3339),
				homedir,
			))
		homedir += "/"
		for _, fd := range instance.Files(i) {
			if path.IsAbs(fd.Path) {
				uid, gid := i.Host().GetFileOwner(fd.Stat)
				path := fd.Path
				if strings.HasPrefix(path, homedir) {
					path = strings.Replace(path, homedir, "", 1)
				}
				fdPerm := ""
				m := fd.Lstat.Mode().Perm()
				if m&0400 == 0400 {
					fdPerm += "r"
				}
				if m&0200 == 0200 {
					fdPerm += "w"
				}
				resp.Lines = append(resp.Lines,
					fmt.Sprintf("%s\t%s\t%s\t%d\t%d:%s\t%s\t%s:%s\t%d\t%s\t%s",
						i.Type(),
						i.Name(),
						i.Host(),
						pid,
						fd.FD,
						fdPerm,
						fd.Stat.Mode().Perm().String(),
						geneos.GetUsername(uid),
						geneos.GetGroupname(gid),
						fd.Stat.Size(),
						fd.Stat.ModTime().Local().Format(time.RFC3339),
						path,
					))
			}
		}

		return
	}

	var portlist string
	portsString := []string{}
	for _, p := range ports {
		portsString = append(portsString, fmt.Sprint(p))
	}
	portlist = strings.Join(portsString, ",")
	if !i.Host().IsLocal() && portlist == "" {
		portlist = "..."
	}

	p := &instance.ProcessStats{}
	if err := instance.ProcessStatus(i, p); err == nil && psCmdLong {
		resp.Line = fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s%s%s\t%s\t%s\t%d\t%d\t%d\t%.2f MiB\t%.2f MiB\t%.2f MiB\t%.2f s\t%.2f s\t%.2f s\t%.2f s",
			i.Type(),
			i.Name(),
			i.Host(),
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
		resp.Line = fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s%s%s\t%s",
			i.Type(),
			i.Name(),
			i.Host(),
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
	}

	return
}

func psInstanceCSV(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)
	pid, username, groupname, mtime, base, actual, uptodate, ports, err := psInstanceCommon(i)
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
			i.Type().String(),
			i.Name(),
			i.Host().String(),
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

		for _, fd := range instance.Files(i) {
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
					i.Type().String(),
					i.Name(),
					i.Host().String(),
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
		hs, err := i.Host().Stat(homedir)
		if err != nil {
			resp.Err = err
			return
		}
		uid, gid := i.Host().GetFileOwner(hs)

		var row []string
		if psCmdToolkit {
			row = append(row, instance.IDString(i))
		}

		row = append(row,
			i.Type().String(),
			i.Name(),
			i.Host().String(),
			fmt.Sprint(pid),
			"cwd",
			hs.Mode().Perm().String(),
			geneos.GetUsername(uid),
			geneos.GetGroupname(gid),
			fmt.Sprint(hs.Size()),
			hs.ModTime().Local().Format(time.RFC3339),
			homedir,
		)
		resp.Rows = append(resp.Rows, row)

		homedir += "/"
		for _, fd := range instance.Files(i) {
			if path.IsAbs(fd.Path) {
				uid, gid := i.Host().GetFileOwner(fd.Stat)
				path := fd.Path
				if strings.HasPrefix(path, homedir) {
					path = strings.Replace(path, homedir, "", 1)
				}
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
					i.Type().String(),
					i.Name(),
					i.Host().String(),
					fmt.Sprint(pid),
					fmt.Sprintf("%d:%s", fd.FD, fdPerm),
					fd.Stat.Mode().Perm().String(),
					geneos.GetUsername(uid),
					geneos.GetGroupname(gid),
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
	if !i.Host().IsLocal() && portlist == "" {
		portlist = "..."
	}

	var row []string

	if psCmdToolkit {
		row = append(row, instance.IDString(i))
	}

	row = append(row,
		i.Type().String(),
		i.Name(),
		i.Host().String(),
		fmt.Sprint(pid),
		portlist,
		username,
		groupname,
		mtime.Local().Format(time.RFC3339),
		fmt.Sprintf("%s%s%s", base, uptodate, actual),
		i.Home(),
	)

	if psCmdLong {
		p := &instance.ProcessStats{}
		if err := instance.ProcessStatus(i, p); err == nil {
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

type psInstance struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	PID       string `json:"pid"`
	Ports     []int  `json:"ports,omitempty"`
	User      string `json:"user,omitempty"`
	Group     string `json:"group,omitempty"`
	Starttime string `json:"starttime,omitempty"`
	Version   string `json:"version,omitempty"`
	Home      string `json:"home,omitempty"`

	// Extra fields, when `--long` is used
	Extra *instance.ProcessStats `json:"extra,omitempty"`
}

type psInstanceFiles struct {
	Type     string      `json:"type"`
	Name     string      `json:"name"`
	Host     string      `json:"host"`
	PID      int         `json:"pid"`
	FD       int         `json:"fd"`
	FDPerms  string      `json:"fd_perms"`
	Perms    fs.FileMode `json:"perms"`
	Username string      `json:"username"`
	Group    string      `json:"group"`
	Size     int64       `json:"size"`
	ModTime  time.Time   `json:"mod_time"`
	Path     string      `json:"path"`
}

type psInstanceNetwork struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	PID        int    `json:"pid"`
	FD         int    `json:"fd"`
	Protocol   string `json:"protocol"`
	LocalAddr  net.IP `json:"local_addr"`
	LocalPort  uint16 `json:"local_port"`
	RemoteAddr net.IP `json:"remote_addr"`
	RemotePort uint16 `json:"remote_port"`
	Status     string `json:"status"`
	TXQueue    int64  `json:"tx_queue"`
	RXQueue    int64  `json:"rx_queue"`
}

func psInstanceJSON(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)
	pid, username, groupname, mtime, base, actual, uptodate, ports, err := psInstanceCommon(i)
	if err != nil {
		return
	}

	if psCmdShowNet {
		conns := []psInstanceNetwork{}
		for _, fd := range instance.Files(i) {
			if fd.Conn != nil {
				// socket
				c := fd.Conn
				if !(strings.HasPrefix(c.Protocol, "tcp") || strings.HasPrefix(c.Protocol, "udp")) {
					continue
				}
				conns = append(conns, psInstanceNetwork{
					Type:       i.Type().String(),
					Name:       i.Name(),
					Host:       i.Host().String(),
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
		hs, err := i.Host().Stat(homedir)
		if err != nil {
			resp.Err = err
			return
		}
		uid, gid := i.Host().GetFileOwner(hs)

		files := []psInstanceFiles{}

		files = append(files, psInstanceFiles{
			Type:     i.Type().String(),
			Name:     i.Name(),
			Host:     i.Host().String(),
			PID:      pid,
			FD:       -1,
			Perms:    hs.Mode().Perm(),
			Username: geneos.GetUsername(uid),
			Group:    geneos.GetGroupname(gid),
			Size:     hs.Size(),
			ModTime:  hs.ModTime(),
			Path:     homedir,
		})

		homedir += "/"
		for _, fd := range instance.Files(i) {
			if path.IsAbs(fd.Path) {
				uid, gid := i.Host().GetFileOwner(fd.Stat)
				path := fd.Path
				if strings.HasPrefix(path, homedir) {
					path = strings.Replace(path, homedir, "", 1)
				}
				fdPerm := ""
				m := fd.Lstat.Mode().Perm()
				if m&0400 == 0400 {
					fdPerm += "r"
				}
				if m&0200 == 0200 {
					fdPerm += "w"
				}
				files = append(files, psInstanceFiles{
					Type:     i.Type().String(),
					Name:     i.Name(),
					Host:     i.Host().String(),
					PID:      pid,
					FD:       fd.FD,
					FDPerms:  fdPerm,
					Perms:    fd.Stat.Mode().Perm(),
					Username: geneos.GetUsername(uid),
					Group:    geneos.GetGroupname(gid),
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
		Type:      i.Type().String(),
		Name:      i.Name(),
		Host:      i.Host().String(),
		PID:       fmt.Sprint(pid),
		Ports:     ports,
		User:      username,
		Group:     groupname,
		Starttime: mtime.Local().Format(time.RFC3339),
		Version:   fmt.Sprintf("%s%s%s", base, uptodate, actual),
		Home:      i.Home(),
	}

	if psCmdLong {
		psData.Extra = &instance.ProcessStats{}
		instance.ProcessStatus(i, psData.Extra)
	}

	resp.Value = psData
	return
}

// live is unused for now
func live(i geneos.Instance) bool {
	cf := i.Config()
	h := i.Host()
	port := cf.GetInt("port")
	cert := cf.GetString("certificate")
	chain := cf.GetString("certchain", config.Default(h.PathTo("tls", geneos.ChainCertFile)))

	scheme := "http"
	client := http.DefaultClient

	if cert != "" {
		scheme = "https"
		roots := config.ReadCertPool(h, chain)

		client.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				RootCAs: roots,
			},
		}
	}

	resp, err := client.Get(fmt.Sprintf("%s://%s:%d/liveness", scheme, h.Hostname(), port))
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			return true
		}
	}
	return false
}
