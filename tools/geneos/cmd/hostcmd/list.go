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
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "false",
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
	fmt.Fprintf(w.(io.Writer), "%s\t%s\t%s\t%s\t%d\t%s\n", h.GetString("name"), h.GetString("username"), h.GetString("hostname"), flags, h.GetInt("port", config.Default(22)), h.GetString(cmd.Execname))
	return
}

func hostListInstanceCSVHosts(h *geneos.Host, w any) (err error) {
	flags := "-"
	if h.Hidden() {
		flags = "H"
	}
	c := w.(*csv.Writer)
	c.Write([]string{h.String(), h.GetString("username"), h.GetString("hostname"), flags, fmt.Sprint(h.GetInt("port", config.Default(22))), h.GetString(cmd.Execname)})
	return
}

func hostListInstanceJSONHosts(h *geneos.Host, w any) (err error) {
	flags := "-"
	if h.Hidden() {
		flags = "H"
	}
	listCmdEntries = append(listCmdEntries, listCmdType{h.String(), h.GetString("username"), h.GetString("hostname"), flags, h.GetInt64("port", config.Default(22)), h.GetString(cmd.Execname)})
	return
}
