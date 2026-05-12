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
	"errors"
	"fmt"
	"os"
	"path"
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
	cmd.Cmd.AddCommand(psCmd)

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
	PID  int               `json:"pid"`
}

// CommandPS writes running instance information to STDOUT
func CommandPS(ct *geneos.Component, names []string, params []string) {
	switch {
	case psCmdJSON, psCmdIndent:
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, psInstanceJSON).Formatted(os.Stdout, "json", nil, nil,
			responses.IndentJSON(psCmdIndent),
			responses.IgnoreErr(geneos.ErrDisabled),
			responses.IgnoreErr(os.ErrProcessDone),
		)

	case psCmdCSV, psCmdToolkit:
		var columns []string
		format := "csv"
		if psCmdToolkit {
			format = "toolkit"
		}

		switch {
		case psCmdShowNet:
			if psCmdToolkit {
				columns = netToolkitColumns
			} else {
				columns = netCSVColumns
			}
		case psCmdShowFiles:
			if psCmdToolkit {
				columns = fileToolkitColumns
			} else {
				columns = fileCSVColumns
			}
		default:
			if psCmdToolkit {
				columns = instanceToolkitColumns
				if psCmdLong {
					columns = append(columns, instanceToolkitExtraColumns...)
				}
			} else {
				columns = instanceCSVColumns
				if psCmdLong {
					columns = append(columns, instanceCSVExtraColumns...)
				}
			}
		}

		resps := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, psInstanceCSV)

		headlines := map[string]string{}

		if psCmdToolkit {
			switch {
			case psCmdShowNet:
				// no headlines yet
			case psCmdShowFiles:
				// no headlines yet
			default:
				var notRunning int
				var disabled int
				for _, r := range resps {
					if errors.Is(r.Err, os.ErrProcessDone) {
						notRunning++
					}
					if errors.Is(r.Err, geneos.ErrDisabled) {
						disabled++
					}
				}
				headlines = map[string]string{
					"instances":  fmt.Sprint(len(resps)),
					"running":    fmt.Sprint(len(resps) - notRunning - disabled),
					"notRunning": fmt.Sprint(notRunning),
					"disabled":   fmt.Sprint(disabled),
				}
			}
		}

		resps.Formatted(os.Stdout, format, columns, nil,
			responses.IgnoreErr(geneos.ErrDisabled),
			responses.AddHeadlines(headlines),
		)

	default:
		var columns []string

		// psTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		if psCmdShowNet {
			columns = netCSVColumns
		} else if psCmdShowFiles {
			columns = fileCSVColumns
		} else if psCmdLong {
			columns = append(instanceCSVColumns, instanceCSVExtraColumns...)
		} else {
			columns = instanceCSVColumns
		}

		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, psInstanceTable).Formatted(os.Stdout, "column", columns, nil)

		// Report(psTabWriter, responses.IgnoreErr(geneos.ErrDisabled))
	}
}

func psInstanceCommon(i geneos.Instance) (pi *process.ProcessInfo, base, actual, uptodate string, err error) {
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

	return
}

func psInstanceTable(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.NewResponse(i)

	h := i.Host()
	ct := i.Type()
	name := i.Name()

	pi, base, actual, uptodate, err := psInstanceCommon(i)
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

	row := []string{}
	row = append(row,
		ct.String(),
		name,
		h.String(),
		fmt.Sprint(pi.PID),
		pi.ListeningPorts,
		pi.Username,
		pi.Groupname,
		pi.StartTime.Local().Format(time.RFC3339),
		fmt.Sprintf("%s%s%s", base, uptodate, actual),
		i.Home(),
	)

	if psCmdLong {
		row = append(row,
			pi.State,
			fmt.Sprint(pi.Threads),
			fmt.Sprint(len(pi.OpenFiles)),
			fmt.Sprint(pi.OpenSockets),
			fmt.Sprintf("%.2f MiB", float64(pi.VmRSS)/(1024*1024)),
			fmt.Sprintf("%.2f MiB", float64(pi.RssAnon)/(1024*1024)),
			fmt.Sprintf("%.2f MiB", float64(pi.VmHWM)/(1024*1024)),
			fmt.Sprintf("%.2f s", pi.Utime.Seconds()),
			fmt.Sprintf("%.2f s", pi.Stime.Seconds()),
			fmt.Sprintf("%.2f s", pi.CUtime.Seconds()),
			fmt.Sprintf("%.2f s", pi.CStime.Seconds()),
		)
	}

	resp.Dataview.Table = append(resp.Dataview.Table, row)

	if capi, ok, err := checkCA(h, ct, pi.Children); err == nil && ok {
		// if this is a netprobe and has a CA child process then we want to list it, but ignore other child processes for now
		log.Debug().Msgf("pid %d has CA child process with pid %d", pi.PID, capi.PID)

		row := []string{}
		row = append(row,
			ct.String()+"/ca",
			name,
			h.String(),
			fmt.Sprint(capi.PID),
			capi.ListeningPorts,
			capi.Username,
			capi.Groupname,
			capi.StartTime.Local().Format(time.RFC3339),
			fmt.Sprintf("%s%s%s", base, uptodate, actual),
			i.Home(),
		)

		if psCmdLong {
			row = append(row,
				capi.State,
				fmt.Sprint(capi.Threads),
				fmt.Sprint(len(capi.OpenFiles)),
				fmt.Sprint(capi.OpenSockets),
				fmt.Sprintf("%.2f MiB", float64(capi.VmRSS)/(1024*1024)),
				fmt.Sprintf("%.2f MiB", float64(capi.RssAnon)/(1024*1024)),
				fmt.Sprintf("%.2f MiB", float64(capi.VmHWM)/(1024*1024)),
				fmt.Sprintf("%.2f s", capi.Utime.Seconds()),
				fmt.Sprintf("%.2f s", capi.Stime.Seconds()),
				fmt.Sprintf("%.2f s", capi.CUtime.Seconds()),
				fmt.Sprintf("%.2f s", capi.CStime.Seconds()),
			)
		}
		resp.Dataview.Table = append(resp.Dataview.Table, row)
	}

	return
}

func psInstanceCSV(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.NewResponse(i)

	h := i.Host()
	ct := i.Type()
	name := i.Name()

	pi, base, actual, uptodate, err := psInstanceCommon(i)
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

	var row []string

	if psCmdToolkit {
		row = append(row, instance.IDString(i))
	}

	row = append(row,
		ct.String(),
		name,
		h.String(),
		fmt.Sprint(pi.PID),
		pi.ListeningPorts,
		pi.Username,
		pi.Groupname,
		pi.StartTime.Local().Format(time.RFC3339),
		fmt.Sprintf("%s%s%s", base, uptodate, actual),
		i.Home(),
	)

	if psCmdLong {
		// p, _ := process.ProcessStatus[*process.ProcessInfo](h, pi.PID)
		p, _ := process.GetProcessInfo(h, pi.PID, false)
		if p != nil {
			row = append(row,
				p.State,
				fmt.Sprint(p.Threads),
				fmt.Sprint(len(p.OpenFiles)),
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

	resp.Dataview.Table = append(resp.Dataview.Table, row)

	if capi, ok, err := checkCA(h, ct, pi.Children); err == nil && ok {
		row := []string{}
		// if this is a netprobe and has a CA child process then we want to list it, but ignore other child processes for now
		log.Debug().Msgf("pid %d has CA child process with pid %d", pi.PID, capi.PID)
		if psCmdToolkit {
			row = append(row, instance.IDString(i)+" # ca")
		}

		row = append(row,
			ct.String()+"/ca",
			name,
			h.String(),
			fmt.Sprint(capi.PID),
			capi.ListeningPorts,
			capi.Username,
			capi.Groupname,
			capi.StartTime.Local().Format(time.RFC3339),
			fmt.Sprintf("%s%s%s", base, uptodate, actual),
			i.Home(),
		)

		if psCmdLong {
			p, _ := process.GetProcessInfo(h, capi.PID, false)
			if p != nil {
				row = append(row,
					p.State,
					fmt.Sprint(p.Threads),
					fmt.Sprint(len(p.OpenFiles)),
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
		resp.Dataview.Table = append(resp.Dataview.Table, row)
	}

	return
}

func psInstanceJSON(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.NewResponse(i)

	pid, err := instance.GetPID(i)
	if err != nil {
		resp.Err = err
		return
	}

	if psCmdShowNet {
		if conns, err := psNetworkJSON(i, pid); err != nil {
			resp.Err = err
		} else {
			for _, c := range conns {
				resp.Values = append(resp.Values, c)
			}
		}
		return
	}

	if psCmdShowFiles {
		if files, err := psFilesJSON(i, pid); err != nil {
			resp.Err = err
		} else {
			for _, f := range files {
				resp.Values = append(resp.Values, f)
			}
		}
		return
	}

	if psData, err := getInstanceData(i); err != nil {
		resp.Err = err
	} else {
		resp.Values = append(resp.Values, psData)
	}
	return
}
