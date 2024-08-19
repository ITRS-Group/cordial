/*
Copyright © 2023 ITRS Group

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

package pkgcmd

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var listCmdJSON, listCmdIndent, listCmdCSV bool

func init() {
	packageCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listCmdJSON, "json", "j", false, "Output JSON")
	listCmd.Flags().BoolVarP(&listCmdIndent, "pretty", "i", false, "Output indented JSON")
	listCmd.Flags().BoolVarP(&listCmdCSV, "csv", "c", false, "Output CSV")

	listCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var listCmdDescription string

var listCmd = &cobra.Command{
	Use:          "list [flags] [TYPE]",
	Short:        "List packages available for update command",
	Long:         listCmdDescription,
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		ct, _ := cmd.ParseTypeNames(command)
		h := geneos.GetHost(cmd.Hostname)
		versions := []geneos.ReleaseDetails{}

		for _, h := range h.OrList(geneos.AllHosts()...) {
			for _, ct := range ct.OrList(geneos.RealComponents()...) {
				v, err := geneos.GetReleases(h, ct)
				if err != nil {
					return err
				}
				// append in reverse order
				for i := len(v) - 1; i >= 0; i-- {
					versions = append(versions, v[i])
				}
			}
		}

		switch {
		case listCmdJSON, listCmdIndent:
			var b []byte
			if listCmdIndent {
				b, err = json.MarshalIndent(versions, "", "    ")
			} else {
				b, err = json.Marshal(versions)
			}
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		case listCmdCSV:
			packageListCSVWriter := csv.NewWriter(os.Stdout)
			packageListCSVWriter.Write([]string{"Component", "Host", "Version", "Latest", "Links", "LastModified", "Path"})
			for _, d := range versions {
				packageListCSVWriter.Write([]string{d.Component, d.Host, d.Version, fmt.Sprintf("%v", d.Latest), strings.Join(d.Links, ", "), d.ModTime.Format(time.RFC3339), d.Path})
			}
			packageListCSVWriter.Flush()
		default:
			packageListTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(packageListTabWriter, "Component\tHost\tVersion\tLinks\tLastModified\tPath\n")
			for _, d := range versions {
				name := d.Version
				if d.Latest {
					name = fmt.Sprintf("%s (latest)", d.Version)
				}
				fmt.Fprintf(packageListTabWriter, "%s\t%s\t%s\t%s\t%s\t%s\n", d.Component, d.Host, name, strings.Join(d.Links, ", "), d.ModTime.Format(time.RFC3339), d.Path)
			}
			packageListTabWriter.Flush()
		}

		if err == os.ErrNotExist {
			err = nil
		}

		return
	},
}
