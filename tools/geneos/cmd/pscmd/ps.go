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

package pscmd

import (
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

var psCmdLong, psCmdShowFiles, psCmdShowNet, psCmdJSON, psCmdIndent, psCmdCSV, psCmdToolkit bool

func init() {
	cmd.GeneosCmd.AddCommand(psCmd)

	psCmd.Flags().BoolVarP(&psCmdShowFiles, "files", "f", false, "Show open files")
	psCmd.Flags().BoolVarP(&psCmdShowNet, "network", "n", false, "Show TCP sockets")

	psCmd.Flags().BoolVarP(&psCmdLong, "long", "l", false, "Show more output (remote ports etc.)")

	psCmd.Flags().BoolVarP(&psCmdJSON, "json", "j", false, "Output JSON")
	psCmd.Flags().BoolVarP(&psCmdIndent, "pretty", "i", false, "Output indented JSON")
	psCmd.Flags().BoolVarP(&psCmdCSV, "csv", "c", false, "Output CSV")
	psCmd.Flags().BoolVarP(&psCmdToolkit, "toolkit", "t", false, "Output Toolkit formatted CSV")

	psCmd.Flags().SortFlags = false
}

//go:embed README.md
var psCmdDescription string

var psCmd = &cobra.Command{
	Use:          "ps [flags] [TYPE] [NAMES...]",
	GroupID:      cmd.CommandGroupView,
	Short:        "List Running Instance Details",
	Long:         psCmdDescription,
	Aliases:      []string{"status"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:               "true",
		cmd.CmdRequireHome:          "true",
		cmd.CmdWildcardNames:        "true",
		cmd.CmdAllowRoot:            "true",
		cmd.CmdNonInstanceArgsError: "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, names, params, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}
		CommandPS(ct, names, params)
	},
}

type psCommon struct {
	Type *geneos.Component `json:"type"`
	Name string            `json:"name"`
	Host *geneos.Host      `json:"host"`
	PID  int64             `json:"pid"`
}

// CommandPS writes running instance information to STDOUT
func CommandPS(ct *geneos.Component, names []string, params []string) {
	switch {
	case psCmdJSON, psCmdIndent:
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, psInstanceJSON).Report(os.Stdout, responses.IndentJSON(psCmdIndent))

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
		resp := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, psInstanceCSV)
		resp.Report(psCSVWriter, responses.IgnoreErr(geneos.ErrDisabled))
		switch {
		case psCmdShowNet:
			// no headlines yet
		case psCmdShowFiles:
			// no headlines yet
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
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, psInstanceCSV).Report(psCSVWriter)

	default:
		psTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		if psCmdShowNet {
			fmt.Fprintln(psTabWriter, netCSVHeader)
		} else if psCmdShowFiles {
			fmt.Fprintln(psTabWriter, fileCSVHeader)
		} else if psCmdLong {
			fmt.Fprintln(psTabWriter, instanceCSVLongHeader)
		} else {
			fmt.Fprintln(psTabWriter, instanceCSVHeader)
		}

		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, psInstanceTable).Report(psTabWriter)
	}
}

func psInstanceCommon(i geneos.Instance) (pi process.ProcessInfo, base, actual, uptodate string, ports []int, err error) {
	h := i.Host()

	if instance.IsDisabled(i) {
		err = geneos.ErrDisabled
		return
	}
	pi, err = instance.GetProcessInfo(i)
	if err != nil {
		if !errors.Is(err, os.ErrProcessDone) {
			log.Debug().Err(err).Msgf("failed to get PID info for instance %s", i.Name())
		}
		return
	}

	base, underlying, actual, _ := instance.LiveVersion(i, pi.PID)
	if pkgtype := config.Get[string](i.Config(), "pkgtype"); pkgtype != "" {
		base = path.Join(pkgtype, base)
	}
	uptodate = "="
	if underlying != actual {
		uptodate = "<>"
	}

	if h.IsLocal() || psCmdLong {
		ports = process.ListeningPorts(h, pi.PID)
	}

	return
}

func checkCA(h *geneos.Host, pid int64) (pi process.ProcessInfo, ok bool, err error) {
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

	pi, base, actual, uptodate, ports, err := psInstanceCommon(i)
	if err != nil {
		return
	}

	if psCmdShowNet {
		if err = psNetworkTable(i, pi.PID, resp); err != nil {
			resp.Err = err
		}
		return
	}

	// file
	if psCmdShowFiles {
		if err := psFilesTable(i, pi.PID, resp); err != nil {
			resp.Err = err
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
	if err := process.ProcessStatus(h, pi.PID, p); err == nil && psCmdLong {
		resp.Summary = fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s%s%s\t%s\t%s\t%d\t%d\t%d\t%.2f MiB\t%.2f MiB\t%.2f MiB\t%.2f s\t%.2f s\t%.2f s\t%.2f s",
			ct,
			name,
			h,
			pi.PID,
			portlist,
			pi.Username,
			pi.Groupname,
			pi.CreationTime.Local().Format(time.RFC3339),
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
			pi.PID,
			portlist,
			pi.Username,
			pi.Groupname,
			pi.CreationTime.Local().Format(time.RFC3339),
			base,
			uptodate,
			actual,
			i.Home(),
		)

		// look for a collection agent child process
		if len(pi.Children) > 0 && ct.IsA("netprobe") {
			// check for CAs and list, but ignore other child processes (for now)
			for _, childPID := range pi.Children {
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
							pi.Username,
							pi.Groupname,
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

	pi, base, actual, uptodate, ports, err := psInstanceCommon(i)
	if err != nil {
		resp.Err = err
		return
	}

	if psCmdShowNet {
		if err = psNetworkCSV(i, pi.PID, resp); err != nil {
			resp.Err = err
		}
		return
	}

	// file
	if psCmdShowFiles {
		if err := psFilesCSV(i, pi.PID, resp); err != nil {
			resp.Err = err
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
		fmt.Sprint(pi.PID),
		portlist,
		pi.Username,
		pi.Groupname,
		pi.CreationTime.Local().Format(time.RFC3339),
		fmt.Sprintf("%s%s%s", base, uptodate, actual),
		i.Home(),
	)

	if psCmdLong {
		p := &process.ProcessStats{}
		if err := process.ProcessStatus(h, pi.PID, p); err == nil {
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

	pid, err := instance.GetPID(i)
	if err != nil {
		resp.Err = err
		return
	}

	if psCmdShowNet {
		if err = psNetworkJSON(i, pid, resp); err != nil {
			resp.Err = err
		}
		return
	}

	if psCmdShowFiles {
		if err := psFilesJSON(i, pid, resp); err != nil {
			resp.Err = err
		}
		return
	}

	if err = psInstanceJSON2(i, resp); err != nil {
		resp.Err = err
	}
	return
}
