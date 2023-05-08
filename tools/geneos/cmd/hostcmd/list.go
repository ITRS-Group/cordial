/*
Copyright © 2022 ITRS Group

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

package hostcmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type hostLsCmdType struct {
	Name      string
	Username  string
	Hostname  string
	Port      int64
	Directory string
}

var hostLsCmdJSON, hostLsCmdIndent, hostLsCmdCSV bool

var hostLsCmdEntries []hostLsCmdType

var hostLsTabWriter *tabwriter.Writer
var hostLsCSVWriter *csv.Writer

func init() {
	HostCmd.AddCommand(hostLsCmd)

	hostLsCmd.Flags().BoolVarP(&hostLsCmdJSON, "json", "j", false, "Output JSON")
	hostLsCmd.Flags().BoolVarP(&hostLsCmdIndent, "pretty", "i", false, "Output indented JSON")
	hostLsCmd.Flags().BoolVarP(&hostLsCmdCSV, "csv", "c", false, "Output CSV")

	hostLsCmd.Flags().SortFlags = false
}

var hostLsCmd = &cobra.Command{
	Use:   "ls [flags] [TYPE] [NAME...]",
	Short: "List hosts, optionally in CSV or JSON format",
	Long: strings.ReplaceAll(`
List the matching remote hosts.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		// ct, args, params := CmdArgsParams(cmd)
		switch {
		case hostLsCmdJSON, hostLsCmdIndent:
			hostLsCmdEntries = []hostLsCmdType{}
			err = loopHosts(lsInstanceJSONHosts)
			var b []byte
			if hostLsCmdIndent {
				b, _ = json.MarshalIndent(hostLsCmdEntries, "", "    ")
			} else {
				b, _ = json.Marshal(hostLsCmdEntries)
			}
			fmt.Println(string(b))
		case hostLsCmdCSV:
			hostLsCSVWriter = csv.NewWriter(os.Stdout)
			hostLsCSVWriter.Write([]string{"Type", "Name", "Disabled", "Username", "Hostname", "Port", "Directory"})
			err = loopHosts(lsInstanceCSVHosts)
			hostLsCSVWriter.Flush()
		default:
			hostLsTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(hostLsTabWriter, "Name\tUsername\tHostname\tPort\tDirectory\n")
			err = loopHosts(lsInstancePlainHosts)
			hostLsTabWriter.Flush()
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func loopHosts(fn func(*geneos.Host) error) error {
	for _, h := range geneos.RemoteHosts() {
		fn(h)
	}
	return nil
}

func lsInstancePlainHosts(h *geneos.Host) (err error) {
	fmt.Fprintf(hostLsTabWriter, "%s\t%s\t%s\t%d\t%s\n", h.GetString("name"), h.GetString("username"), h.GetString("hostname"), h.GetInt("port", config.Default(22)), h.GetString("geneos"))
	return
}

func lsInstanceCSVHosts(h *geneos.Host) (err error) {
	hostLsCSVWriter.Write([]string{h.String(), h.GetString("username"), h.GetString("hostname"), fmt.Sprint(h.GetInt("port"), config.Default(22)), h.GetString("geneos")})
	return
}

func lsInstanceJSONHosts(h *geneos.Host) (err error) {
	hostLsCmdEntries = append(hostLsCmdEntries, hostLsCmdType{h.String(), h.GetString("username"), h.GetString("hostname"), h.GetInt64("port", config.Default(22)), h.GetString("geneos")})
	return
}
