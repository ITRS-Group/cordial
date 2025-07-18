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
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var listCmdJSON, listCmdIndent, listCmdCSV, listCmdToolkit bool

func init() {
	packageCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listCmdJSON, "json", "j", false, "Output JSON")
	listCmd.Flags().BoolVarP(&listCmdIndent, "pretty", "i", false, "Output indented JSON")
	listCmd.Flags().BoolVarP(&listCmdCSV, "csv", "c", false, "Output CSV")
	listCmd.Flags().BoolVarP(&listCmdToolkit, "toolkit", "t", false, "Output Toolkit formatted CSV")

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
		versions := []*geneos.ReleaseDetails{}

		for h := range h.OrList() {
			for ct := range ct.OrList() {
				v, err := geneos.GetReleases(h, ct)
				if err != nil {
					return err
				}
				// append in reverse order
				slices.Reverse(v)
				versions = append(versions, v...)
			}
		}

		// sort versions, by host ct
		slices.SortFunc(versions, func(a, b *geneos.ReleaseDetails) int {
			s := strings.Compare(a.Host, b.Host)
			if s != 0 {
				return s
			}
			return strings.Compare(a.Component, b.Component)
		})

		switch {
		case listCmdJSON, listCmdIndent:
			var b []byte
			for _, v := range versions {
				for _, l := range v.Links {
					v.Instances += len(instance.Instances(geneos.GetHost(v.Host), geneos.ParseComponent(v.Component), instance.FilterParameters("version="+l)))
				}
				v.Instances += len(instance.Instances(geneos.GetHost(v.Host), geneos.ParseComponent(v.Component), instance.FilterParameters("version="+v.Version)))
			}
			if listCmdIndent {
				b, err = json.MarshalIndent(versions, "", "    ")
			} else {
				b, err = json.Marshal(versions)
			}
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		case listCmdToolkit:
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{
				"ID",
				"component",
				"host",
				"version",
				"latestInstalled",
				"instances",
				"links",
				"lastModified",
				"path"})
			for _, d := range versions {
				id := d.Component + "-" + d.Version
				if d.Host != geneos.LOCALHOST {
					id += "@" + d.Host
				}
				var instances int
				for _, v := range d.Links {
					instances += len(instance.Instances(geneos.GetHost(d.Host), geneos.ParseComponent(d.Component), instance.FilterParameters("version="+v)))
				}
				instances += len(instance.Instances(geneos.GetHost(d.Host), geneos.ParseComponent(d.Component), instance.FilterParameters("version="+d.Version)))
				w.Write([]string{
					id,
					d.Component,
					d.Host,
					d.Version,
					fmt.Sprintf("%v", d.Latest),
					fmt.Sprintf("%d", instances),
					strings.Join(d.Links, " "),
					d.ModTime.Format(time.RFC3339),
					d.Path,
				})
			}
			w.Flush()
		case listCmdCSV:
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{
				"Component",
				"Host",
				"Version",
				"LatestInstalled",
				"Instances",
				"Links",
				"LastModified",
				"Path",
			})
			for _, d := range versions {
				var instances int
				for _, v := range d.Links {
					instances += len(instance.Instances(geneos.GetHost(d.Host), geneos.ParseComponent(d.Component), instance.FilterParameters("version="+v)))
				}
				instances += len(instance.Instances(geneos.GetHost(d.Host), geneos.ParseComponent(d.Component), instance.FilterParameters("version="+d.Version)))
				w.Write([]string{
					d.Component,
					d.Host,
					d.Version,
					fmt.Sprintf("%v", d.Latest),
					fmt.Sprintf("%d", instances),
					strings.Join(d.Links, ", "),
					d.ModTime.Format(time.RFC3339),
					d.Path,
				})
			}
			w.Flush()
		default:
			w := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(w, "Component\tHost\tVersion\tLinks\tInstances\tLastModified\tPath\n")
			for _, d := range versions {
				var instances int
				for _, v := range d.Links {
					instances += len(instance.Instances(geneos.GetHost(d.Host), geneos.ParseComponent(d.Component), instance.FilterParameters("version="+v)))
				}
				instances += len(instance.Instances(geneos.GetHost(d.Host), geneos.ParseComponent(d.Component), instance.FilterParameters("version="+d.Version)))
				name := d.Version
				if d.Latest {
					name = fmt.Sprintf("%s (latest)", d.Version)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
					d.Component,
					d.Host,
					name,
					strings.Join(d.Links, ", "),
					instances,
					d.ModTime.Format(time.RFC3339),
					d.Path)

			}
			w.Flush()
		}

		if err == os.ErrNotExist {
			err = nil
		}

		return
	},
}
