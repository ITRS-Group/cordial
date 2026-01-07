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
	"fmt"
	"os"
	"path"
	"text/tabwriter"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
	"github.com/spf13/cobra"
)

type listCmdType struct {
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	Host      string `json:"host,omitempty"`
	Running   bool   `json:"running"`
	Disabled  bool   `json:"disabled"`
	Protected bool   `json:"protected"`
	AutoStart bool   `json:"autostart"`
	TLS       bool   `json:"tls"`
	Port      int64  `json:"port,omitempty"`
	Version   string `json:"version,omitempty"`
	Home      string `json:"home,omitempty"`
}

var listCmdJSON, listCmdCSV, listCmdIndent, listCmdToolkit bool

func init() {
	GeneosCmd.AddCommand(listCmd)

	listCmd.PersistentFlags().BoolVarP(&listCmdJSON, "json", "j", false, "Output JSON")
	listCmd.PersistentFlags().BoolVarP(&listCmdIndent, "pretty", "i", false, "Output indented JSON")
	listCmd.PersistentFlags().BoolVarP(&listCmdCSV, "csv", "c", false, "Output CSV")
	listCmd.PersistentFlags().BoolVarP(&listCmdToolkit, "toolkit", "t", false, "Output Toolkit formatted CSV")

	listCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var listCmdDescription string

var listCmd = &cobra.Command{
	Use:          "list [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupView,
	Short:        "List Instances",
	Long:         listCmdDescription,
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names := ParseTypeNames(cmd)
		switch {
		case listCmdJSON, listCmdIndent:
			instance.Do(geneos.GetHost(Hostname), ct, names, listInstanceJSON).Report(os.Stdout, responses.IndentJSON(listCmdIndent))
		case listCmdToolkit:
			listCSVWriter := csv.NewWriter(os.Stdout)
			listCSVWriter.Write([]string{
				"ID",
				"type",
				"name",
				"host",
				"autoStart",
				"disabled",
				"protected",
				"running",
				"tls",
				"port",
				"version",
				"home",
			})
			resp := instance.Do(geneos.GetHost(Hostname), ct, names, listInstanceCSV)
			resp.Report(listCSVWriter)
			fmt.Printf("<!>instances,%d\n", len(resp))
			for _, ct := range geneos.RealComponents() {
				var count int
				for _, r := range resp {
					if r.Instance.Type() == ct {
						count++
					}
				}
				fmt.Printf("<!>%ss,%d\n", ct, count)
			}
		case listCmdCSV:
			listCSVWriter := csv.NewWriter(os.Stdout)
			listCSVWriter.Write([]string{
				"Type",
				"Name",
				"Host",
				"AutoStart",
				"Disabled",
				"Protected",
				"Running",
				"TLS",
				"Port",
				"Version",
				"Home",
			})
			instance.Do(geneos.GetHost(Hostname), ct, names, listInstanceCSV).Report(listCSVWriter)
		default:
			listTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(listTabWriter, "Type\tName\tHost\tFlags\tPort\tVersion\tHome\n")
			instance.Do(geneos.GetHost(Hostname), ct, names, listInstancePlain).Report(listTabWriter)
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func listInstancePlain(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	var flags string

	if instance.IsAutoStart(i) {
		flags += "A"
	}
	if instance.IsDisabled(i) {
		flags += "D"
	}
	if instance.IsProtected(i) {
		flags += "P"
	}
	if instance.IsRunning(i) {
		flags += "R"
	}
	secureArgs, _, _, _ := instance.SecureArgs(i)
	if len(secureArgs) > 0 {
		flags += "T"
	}
	if flags == "" {
		flags = "-"
	}
	base, underlying, _ := instance.Version(i)
	if pkgtype := i.Config().GetString("pkgtype"); pkgtype != "" {
		base = path.Join(pkgtype, base)
	}

	resp.Summary = fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s:%s\t%s", i.Type(), i.Name(), i.Host(), flags, i.Config().GetInt("port"), base, underlying, i.Home())
	return
}

func listInstanceCSV(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	running := "N"
	disabled := "N"
	protected := "N"
	autostart := "N"
	tls := "N"

	if instance.IsRunning(i) {
		running = "Y"
	}
	if instance.IsDisabled(i) {
		disabled = "Y"
	}
	if instance.IsProtected(i) {
		protected = "Y"
	}
	if instance.IsAutoStart(i) {
		autostart = "Y"
	}
	secureArgs, _, _, _ := instance.SecureArgs(i)
	if len(secureArgs) > 0 {
		tls = "Y"
	}
	base, underlying, _ := instance.Version(i)
	var row []string

	if listCmdToolkit {
		row = append(row, instance.IDString(i))
	}
	row = append(row,
		i.Type().String(),
		i.Name(),
		i.Host().String(),
		autostart,
		disabled,
		protected,
		running,
		tls,
		fmt.Sprint(i.Config().GetInt("port")),
		fmt.Sprintf("%s:%s", base, underlying), i.Home(),
	)
	resp.Rows = append(resp.Rows, row)
	return
}

func listInstanceJSON(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	base, underlying, _ := instance.Version(i)
	secureArgs, _, _, _ := instance.SecureArgs(i)
	resp.Value = listCmdType{
		Type:      i.Type().String(),
		Name:      i.Name(),
		Host:      i.Host().String(),
		Running:   instance.IsRunning(i),
		Disabled:  instance.IsDisabled(i),
		Protected: instance.IsProtected(i),
		AutoStart: instance.IsAutoStart(i),
		TLS:       len(secureArgs) > 0,
		Port:      i.Config().GetInt64("port"),
		Version:   fmt.Sprintf("%s:%s", base, underlying),
		Home:      i.Home(),
	}
	return
}
