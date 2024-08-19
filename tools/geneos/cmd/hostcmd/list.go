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

package hostcmd

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type listCmdType struct {
	Name      string
	Username  string
	Hostname  string
	Flags     string
	Port      int64
	Directory string
}

var listCmdShowHidden, listCmdJSON, listCmdIndent, listCmdCSV bool

var listCmdEntries []listCmdType

func init() {
	hostCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listCmdShowHidden, "all", "a", false, "Show all hosts")
	listCmd.Flags().BoolVarP(&listCmdJSON, "json", "j", false, "Output JSON")
	listCmd.Flags().BoolVarP(&listCmdIndent, "pretty", "i", false, "Output indented JSON")
	listCmd.Flags().BoolVarP(&listCmdCSV, "csv", "c", false, "Output CSV")

	listCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var listCmdDescription string

var listCmd = &cobra.Command{
	Use:          "list [flags] [NAME...]",
	Short:        "List hosts, optionally in CSV or JSON format",
	Long:         listCmdDescription,
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		switch {
		case listCmdJSON, listCmdIndent:
			listCmdEntries = []listCmdType{}
			err = loopHosts(hostListInstanceJSONHosts, os.Stdout, listCmdShowHidden)
			var b []byte
			if listCmdIndent {
				b, _ = json.MarshalIndent(listCmdEntries, "", "    ")
			} else {
				b, _ = json.Marshal(listCmdEntries)
			}
			fmt.Println(string(b))
		case listCmdCSV:
			hostListCSVWriter := csv.NewWriter(os.Stdout)
			hostListCSVWriter.Write([]string{"Name", "Username", "Hostname", "Flags", "Port", "Directory"})
			err = loopHosts(hostListInstanceCSVHosts, hostListCSVWriter, listCmdShowHidden)
			hostListCSVWriter.Flush()
		default:
			hostListTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(hostListTabWriter, "Name\tUsername\tHostname\tFlags\tPort\tDirectory\n")
			err = loopHosts(hostListInstancePlainHosts, hostListTabWriter, listCmdShowHidden)
			hostListTabWriter.Flush()
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func loopHosts(fn func(*geneos.Host, any) error, w any, showHidden bool) error {
	for _, h := range geneos.RemoteHosts(showHidden) {
		fn(h, w)
	}
	return nil
}

func hostListInstancePlainHosts(h *geneos.Host, w any) (err error) {
	flags := "-"
	if h.Hidden() {
		flags = "H"
	}
	username := h.GetString("username")
	if username == "" {
		username = "-"
	}
	fmt.Fprintf(w.(io.Writer), "%s\t%s\t%s\t%s\t%d\t%s\n", h.GetString("name"), username, h.GetString("hostname"), flags, h.GetInt("port", config.Default(22)), h.GetString(cmd.Execname))
	return
}

func hostListInstanceCSVHosts(h *geneos.Host, w any) (err error) {
	flags := "-"
	if h.Hidden() {
		flags = "H"
	}
	username := h.GetString("username")

	c := w.(*csv.Writer)
	c.Write([]string{h.String(), username, h.GetString("hostname"), flags, fmt.Sprint(h.GetInt("port", config.Default(22))), h.GetString(cmd.Execname)})
	return
}

func hostListInstanceJSONHosts(h *geneos.Host, w any) (err error) {
	flags := "-"
	if h.Hidden() {
		flags = "H"
	}
	username := h.GetString("username")

	listCmdEntries = append(listCmdEntries, listCmdType{h.String(), username, h.GetString("hostname"), flags, h.GetInt64("port", config.Default(22)), h.GetString(cmd.Execname)})
	return
}
