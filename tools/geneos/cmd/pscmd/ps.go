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
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

var psCmdLong, psCmdShowFiles, psCmdShowNet, psCmdJSON, psCmdIndent, psCmdCSV, psCmdToolkit bool

func init() {
	cmd.Cmd.AddCommand(psCmd)

	psCmd.Flags().BoolVarP(&psCmdShowFiles, "files", "f", false, "Show open files")
	psCmd.Flags().BoolVarP(&psCmdShowNet, "network", "n", false, "Show TCP sockets")
	psCmd.MarkFlagsMutuallyExclusive("files", "network")

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
		cmd.CmdAuditCommand:         "never",
		cmd.CmdAuditActions:         "never",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		ct, names, params, err := cmd.FetchArgs(command)
		if err != nil {
			return err
		}
		return CommandPS(command, ct, names, params)
	},
}

type psCommon struct {
	Type *geneos.Component `json:"type"`
	Name string            `json:"name"`
	Host *geneos.Host      `json:"host"`
	PID  int               `json:"pid"`
}

// CommandPS writes running instance information to STDOUT. Instances
// that are not running or are disabled are not listed, but a name that
// is not an instance, or an instance that cannot be inspected, is an
// error.
func CommandPS(command *cobra.Command, ct *geneos.Component, names []string, params []string) error {
	// always start with a clean cache, in case this is called from init commands
	process.ClearCache()

	switch {
	case psCmdJSON, psCmdIndent:
		return instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, psInstanceJSON).Formatted(os.Stdout, "json", nil, nil,
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

		resps := instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, psInstanceCSV)

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
				var total int
				for _, r := range resps {
					if r.Instance == nil {
						continue
					}
					total++
					if errors.Is(r.Err, os.ErrProcessDone) {
						notRunning++
					}
					if errors.Is(r.Err, geneos.ErrDisabled) {
						disabled++
					}
				}
				headlines = map[string]string{
					"instances":  fmt.Sprint(total),
					"running":    fmt.Sprint(total - notRunning - disabled),
					"notRunning": fmt.Sprint(notRunning),
					"disabled":   fmt.Sprint(disabled),
				}
			}
		}

		return resps.Formatted(os.Stdout, format, columns, nil,
			responses.IgnoreErr(geneos.ErrDisabled),
			responses.AddHeadlines(headlines),
		)

	default:
		var columns []string

		if psCmdShowNet {
			columns = netCSVColumns
		} else if psCmdShowFiles {
			columns = fileCSVColumns
		} else if psCmdLong {
			columns = append(instanceCSVColumns, instanceCSVExtraColumns...)
		} else {
			columns = instanceCSVColumns
		}

		return instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, psInstanceTable).Formatted(os.Stdout, "column", columns, nil,
			responses.IgnoreErr(geneos.ErrDisabled),
		)
	}
}

func psInstanceCommon(i geneos.Instance) (pi *instance.ProcessInfo, base, actual, uptodate string, err error) {
	if instance.IsDisabled(i) {
		err = geneos.ErrDisabled
		return
	}
	pi, err = instance.GetProcessInfo(i)
	if err != nil {
		if !errors.Is(err, os.ErrProcessDone) {
			i.Log().Debug("failed to get PID info for instance", slog.Any("error", err))
		}
		return
	}

	base, underlying, actual, _ := instance.LiveVersion(i, pi)
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
	resp = responses.New[responses.General](i)

	pi, base, actual, uptodate, err := psInstanceCommon(i)
	if err != nil {
		resp.Err = err
		return
	}

	if psCmdShowNet {
		if err = psNetworkTable(i, pi.PID, pi, resp); err != nil {
			resp.Err = err
		}
		return
	}

	// file
	if psCmdShowFiles {
		if err := psFilesTable(i, pi.PID, pi, resp); err != nil {
			resp.Err = err
		}
		return
	}

	version := base + uptodate + actual

	row := psRow(i, i.Type().String(), pi, version)
	if psCmdLong {
		row = append(row, psInstanceLongColumns(pi)...)
	}
	resp.Dataview.Table = append(resp.Dataview.Table, row)

	if capi, ok, err := checkCA(i, pi.Children); err == nil && ok {
		// if this is a netprobe and has a CA child process then we want to list it, but ignore other child processes for now
		i.Log().Debug("pid has CA child process", slog.Int("parent_pid", pi.PID), slog.Int("child_pid", capi.PID))

		row := psRow(i, i.Type().String()+"/ca", capi, version)
		if psCmdLong {
			row = append(row, psInstanceLongColumns(capi)...)
		}
		resp.Dataview.Table = append(resp.Dataview.Table, row)
	}

	return
}

// psRow returns the common columns for the process pi of the instance i,
// listed with the type typ
func psRow(i geneos.Instance, typ string, pi *instance.ProcessInfo, version string) []string {
	return []string{
		typ,
		i.Name(),
		i.Host().String(),
		fmt.Sprint(pi.PID),
		pi.ListeningPorts,
		pi.Username,
		pi.Groupname,
		pi.StartTime.Local().Format(time.RFC3339),
		version,
		i.Home(),
	}
}

func psInstanceCSV(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	h := i.Host()

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

	version := base + uptodate + actual

	var row []string
	if psCmdToolkit {
		row = append(row, instance.IDString(i))
	}
	row = append(row, psRow(i, i.Type().String(), pi, version)...)
	if psCmdLong {
		p, _ := process.GetProcessInfo[*instance.ProcessInfo](h, pi.PID, process.FetchLazyFields())
		if p != nil {
			row = append(row, psInstanceLongColumns(p)...)
		}
	}
	resp.Dataview.Table = append(resp.Dataview.Table, row)

	if capi, ok, err := checkCA(i, pi.Children); err == nil && ok {
		// if this is a netprobe and has a CA child process then we want to list it, but ignore other child processes for now
		i.Log().Debug("pid has CA child process", slog.Int("parent_pid", pi.PID), slog.Int("child_pid", capi.PID))

		var row []string
		if psCmdToolkit {
			row = append(row, instance.IDString(i)+" # ca")
		}
		row = append(row, psRow(i, i.Type().String()+"/ca", capi, version)...)
		if psCmdLong {
			p, _ := process.GetProcessInfo[*instance.ProcessInfo](h, capi.PID, process.FetchLazyFields())
			if p != nil {
				row = append(row, psInstanceLongColumns(p)...)
			}
		}
		resp.Dataview.Table = append(resp.Dataview.Table, row)
	}

	return
}

func psInstanceJSON(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	if psCmdShowNet || psCmdShowFiles {
		pid, err := instance.GetLivePID(i)
		if err != nil {
			resp.Err = err
			return
		}
		psInstanceDetailJSON(i, pid, resp)
		return
	}

	if psData, err := getInstanceData(i); err != nil {
		resp.Err = err
	} else {
		resp.Values = append(resp.Values, psData)
	}
	return
}

// psInstanceDetailJSON adds the network or open file details for the
// process with PID pid to resp
func psInstanceDetailJSON(i geneos.Instance, pid int, resp *responses.General) {
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
	}
}
