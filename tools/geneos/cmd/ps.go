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
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

type psType struct {
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
	// Live      bool   `json:"live,omitempty"`

	// Extra fields, when `--long` is used
	Extra *instance.ProcessStats `json:"extra,omitempty"`
}

var psCmdLong, psCmdShowFiles, psCmdJSON, psCmdIndent, psCmdCSV, psCmdNoLookups, psCmdToolkit bool

func init() {
	GeneosCmd.AddCommand(psCmd)

	psCmd.Flags().BoolVarP(&psCmdShowFiles, "files", "f", false, "Show open files")
	psCmd.Flags().MarkHidden("files")

	psCmd.Flags().BoolVarP(&psCmdLong, "long", "l", false, "Show more output (remote ports etc.)")
	psCmd.Flags().BoolVarP(&psCmdNoLookups, "nolookup", "n", false, "No lookups for user/groups")

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
//
// XXX relies on global flags
func CommandPS(ct *geneos.Component, names []string, params []string) {
	switch {
	case psCmdJSON, psCmdIndent:
		instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceJSON).Write(os.Stdout, instance.WriterIndent(psCmdIndent))
	case psCmdToolkit:
		psCSVWriter := csv.NewWriter(os.Stdout)
		columns := []string{
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
				"residentSetSize",
				"residentSetSizeMax",
				"totalUserTime",
				"totalKernelTime",
				"totalChildUserTime",
				"totalChildKernelTime",
			)
		}
		psCSVWriter.Write(columns)
		instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceCSV).Write(psCSVWriter)
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
				"RSS",
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
		if psCmdLong {
			fmt.Fprintf(psTabWriter, "Type\tName\tHost\tPID\tPorts\tUser\tGroup\tStarttime\tVersion\tHome\tState\tThreads\tRSS\tRSSMax\tTotalUserTime\tTotalKernelTime\tTotalChildUserTime\tTotalChildKernelTime\n")
		} else {
			fmt.Fprintf(psTabWriter, "Type\tName\tHost\tPID\tPorts\tUser\tGroup\tStarttime\tVersion\tHome\n")
		}
		instance.Do(geneos.GetHost(Hostname), ct, names, psInstancePlain).Write(psTabWriter)
	}
}

func psInstanceCommon(i geneos.Instance) (pid int, username, groupname string, mtime time.Time, base, actual, uptodate string, ports []int, err error) {
	var u *user.User
	var g *user.Group

	if instance.IsDisabled(i) {
		return
	}
	pid, uid, gid, mtime, err := instance.GetPIDInfo(i)
	if err != nil {
		return
	}

	username = fmt.Sprint(uid)
	groupname = fmt.Sprint(gid)

	if !psCmdNoLookups {
		if u, err = user.LookupId(username); err == nil {
			username = u.Username
		}
		if g, err = user.LookupGroupId(groupname); err == nil {
			groupname = g.Name
		}
	}

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

	var portlist string
	portsString := []string{}
	for _, p := range ports {
		portsString = append(portsString, fmt.Sprint(p))
	}
	portlist = strings.Join(portsString, " ")
	if !i.Host().IsLocal() && portlist == "" {
		portlist = "..."
	}

	p := &instance.ProcessStats{}
	if err := instance.ProcessStatus(i, p); err == nil && psCmdLong {
		resp.Line = fmt.Sprintf("%s\t%s\t%s\t%d\t[%s]\t%s\t%s\t%s\t%s%s%s\t%s\t%s\t%d\t%.2f MiB\t%.2f MiB\t%.2f s\t%.2f s\t%.2f s\t%.2f s",
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
			float64(p.VmRSS)/(1024*1024),
			float64(p.VmHWM)/(1024*1024),
			p.Utime.Seconds(),
			p.Stime.Seconds(),
			p.CUtime.Seconds(),
			p.CStime.Seconds(),
		)
	} else {
		resp.Line = fmt.Sprintf("%s\t%s\t%s\t%d\t[%s]\t%s\t%s\t%s\t%s%s%s\t%s",
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

	if psCmdShowFiles {
		resp.Lines = listOpenFiles(i)
	}
	return
}

func psInstanceCSV(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)
	pid, username, groupname, mtime, base, actual, uptodate, ports, err := psInstanceCommon(i)
	if err != nil {
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
				fmt.Sprintf("%.2f MiB", float64(p.VmRSS)/(1024*1024)),
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

func psInstanceJSON(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)
	pid, username, groupname, mtime, base, actual, uptodate, ports, err := psInstanceCommon(i)
	if err != nil {
		return
	}

	psData := psType{
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
		roots := config.ReadCertChain(h, chain)

		client.Transport = &http.Transport{
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
