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
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type hostListCmdType struct {
	Name      string
	Username  string
	Hostname  string
	Port      int64
	Directory string
}

var hostListCmdJSON, hostListCmdIndent, hostListCmdCSV bool

var hostListCmdEntries []hostListCmdType

var hostListTabWriter *tabwriter.Writer
var hostListCSVWriter *csv.Writer

func init() {
	hostCmd.AddCommand(hostListCmd)

	hostListCmd.Flags().BoolVarP(&hostListCmdJSON, "json", "j", false, "Output JSON")
	hostListCmd.Flags().BoolVarP(&hostListCmdIndent, "pretty", "i", false, "Output indented JSON")
	hostListCmd.Flags().BoolVarP(&hostListCmdCSV, "csv", "c", false, "Output CSV")

	hostListCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var hostListCmdDescription string

var hostListCmd = &cobra.Command{
	Use:          "list [flags] [TYPE] [NAME...]",
	Short:        "List hosts, optionally in CSV or JSON format",
	Long:         hostListCmdDescription,
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		switch {
		case hostListCmdJSON, hostListCmdIndent:
			hostListCmdEntries = []hostListCmdType{}
			err = loopHosts(hostListInstanceJSONHosts)
			var b []byte
			if hostListCmdIndent {
				b, _ = json.MarshalIndent(hostListCmdEntries, "", "    ")
			} else {
				b, _ = json.Marshal(hostListCmdEntries)
			}
			fmt.Println(string(b))
		case hostListCmdCSV:
			hostListCSVWriter = csv.NewWriter(os.Stdout)
			hostListCSVWriter.Write([]string{"Type", "Name", "Disabled", "Username", "Hostname", "Port", "Directory"})
			err = loopHosts(hostListInstanceCSVHosts)
			hostListCSVWriter.Flush()
		default:
			hostListTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(hostListTabWriter, "Name\tUsername\tHostname\tPort\tDirectory\n")
			err = loopHosts(hostListInstancePlainHosts)
			hostListTabWriter.Flush()
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

func hostListInstancePlainHosts(h *geneos.Host) (err error) {
	fmt.Fprintf(hostListTabWriter, "%s\t%s\t%s\t%d\t%s\n", h.GetString("name"), h.GetString("username"), h.GetString("hostname"), h.GetInt("port", config.Default(22)), h.GetString(cmd.Execname))
	return
}

func hostListInstanceCSVHosts(h *geneos.Host) (err error) {
	hostListCSVWriter.Write([]string{h.String(), h.GetString("username"), h.GetString("hostname"), fmt.Sprint(h.GetInt("port", config.Default(22))), h.GetString(cmd.Execname)})
	return
}

func hostListInstanceJSONHosts(h *geneos.Host) (err error) {
	hostListCmdEntries = append(hostListCmdEntries, hostListCmdType{h.String(), h.GetString("username"), h.GetString("hostname"), h.GetInt64("port", config.Default(22)), h.GetString(cmd.Execname)})
	return
}
