/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package pkgcmd

import (
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

var packageLsCmdHost string
var packageLsCmdJSON, packageLsCmdIndent, packageLsCmdCSV bool

var packageLsTabWriter *tabwriter.Writer
var packageLsCSVWriter *csv.Writer

func init() {
	PackageCmd.AddCommand(packageLsCmd)

	packageLsCmd.Flags().StringVarP(&packageLsCmdHost, "host", "H", string(geneos.ALLHOSTS),
		`Apply only on remote host. "all" (the default) means all remote hosts and locally`)
	packageLsCmd.Flags().BoolVarP(&packageLsCmdJSON, "json", "j", false, "Output JSON")
	packageLsCmd.Flags().BoolVarP(&packageLsCmdIndent, "pretty", "i", false, "Output indented JSON")
	packageLsCmd.Flags().BoolVarP(&packageLsCmdCSV, "csv", "c", false, "Output CSV")

	packageLsCmd.Flags().SortFlags = false
}

var packageLsCmd = &cobra.Command{
	Use:   "ls [flags] [TYPE]",
	Short: "List packages available for update command",
	Long: strings.ReplaceAll(`
List the packages for the matching TYPE or all component types if no
TYPE is given. The |-H| flags restricts the check to a specific
remote host.

All timestamps are displayed in UTC to avoid filesystem confusion
between local summer/winter times in some locales.

Versions are listed in descending order for each component type, i.e.
|latest| is always the first entry for each component.
`, "|", "`"),
	Aliases:      []string{"list"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		ct, _ := cmd.CmdArgs(command)

		h := geneos.GetHost(packageLsCmdHost)
		versions := []geneos.ReleaseDetails{}

		for _, h := range h.Range(geneos.AllHosts()...) {
			for _, ct := range ct.Range(geneos.RealComponents()...) {
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
		case packageLsCmdJSON, packageLsCmdIndent:
			var b []byte
			if packageLsCmdIndent {
				b, err = json.MarshalIndent(versions, "", "    ")
			} else {
				b, err = json.Marshal(versions)
			}
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		case packageLsCmdCSV:
			packageLsCSVWriter = csv.NewWriter(os.Stdout)
			packageLsCSVWriter.Write([]string{"Component", "Host", "Version", "Latest", "Links", "LastModified", "Path"})
			for _, d := range versions {
				packageLsCSVWriter.Write([]string{d.Component, d.Host, d.Version, fmt.Sprintf("%v", d.Latest), strings.Join(d.Links, ", "), d.ModTime.Format(time.RFC3339), d.Path})
			}
			packageLsCSVWriter.Flush()
		default:
			packageLsTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(packageLsTabWriter, "Component\tHost\tVersion\tLinks\tLastModified\tPath\n")
			for _, d := range versions {
				name := d.Version
				if d.Latest {
					name = fmt.Sprintf("%s (latest)", d.Version)
				}
				fmt.Fprintf(packageLsTabWriter, "%s\t%s\t%s\t%s\t%s\t%s\n", d.Component, d.Host, name, strings.Join(d.Links, ", "), d.ModTime.Format(time.RFC3339), d.Path)
			}
			packageLsTabWriter.Flush()
		}

		if err == os.ErrNotExist {
			err = nil
		}

		return
	},
}
