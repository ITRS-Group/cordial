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

package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

type lsCmdType struct {
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	Disabled  bool   `json:"disabled"`
	Protected bool   `json:"protected"`
	Host      string `json:"host,omitempty"`
	Port      int64  `json:"port,omitempty"`
	Version   string `json:"version,omitempty"`
	Home      string `json:"home,omitempty"`
}

var lsCmdEntries []lsCmdType

var lsCmdJSON, lsCmdCSV, lsCmdIndent bool

var lsTabWriter *tabwriter.Writer
var csvWriter *csv.Writer
var jsonEncoder *json.Encoder

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.PersistentFlags().BoolVarP(&lsCmdJSON, "json", "j", false, "Output JSON")
	lsCmd.PersistentFlags().BoolVarP(&lsCmdIndent, "pretty", "i", false, "Indent (pretty print) JSON")
	lsCmd.PersistentFlags().BoolVarP(&lsCmdCSV, "csv", "c", false, "Output CSV")

	lsCmd.Flags().SortFlags = false
}

var lsCmd = &cobra.Command{
	Use:   "ls [flags] [TYPE] [NAME...]",
	Short: "List instances, optionally in CSV or JSON format",
	Long: strings.ReplaceAll(`
List the matching instances and details.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, args, params := cmdArgsParams(cmd)
		switch {
		case lsCmdJSON:
			lsCmdEntries = []lsCmdType{}
			err = instance.ForAll(ct, lsInstanceJSON, args, params)
			var b []byte
			if lsCmdIndent {
				b, _ = json.MarshalIndent(lsCmdEntries, "", "    ")
			} else {
				b, _ = json.Marshal(lsCmdEntries)
			}
			fmt.Println(string(b))
		case lsCmdCSV:
			csvWriter = csv.NewWriter(os.Stdout)
			csvWriter.Write([]string{"Type", "Name", "Disabled", "Protected", "Host", "Port", "Version", "Home"})
			err = instance.ForAll(ct, lsInstanceCSV, args, params)
			csvWriter.Flush()
		default:
			lsTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(lsTabWriter, "Type\tName\tHost\tPort\tVersion\tHome\n")
			err = instance.ForAll(ct, lsInstancePlain, args, params)
			lsTabWriter.Flush()
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func lsInstancePlain(c geneos.Instance, params []string) (err error) {
	var suffix string
	if instance.IsDisabled(c) {
		suffix = "*"
	}
	if instance.IsProtected(c) {
		suffix += "+"
	}
	base, underlying, _ := instance.Version(c)
	fmt.Fprintf(lsTabWriter, "%s\t%s\t%s\t%d\t%s:%s\t%s\n", c.Type(), c.Name()+suffix, c.Host(), c.Config().GetInt("port"), base, underlying, c.Home())
	return
}

func lsInstanceCSV(c geneos.Instance, params []string) (err error) {
	var dis string = "N"
	var protected string = "N"
	if instance.IsDisabled(c) {
		dis = "Y"
	}
	if instance.IsProtected(c) {
		protected = "Y"
	}
	base, underlying, _ := instance.Version(c)
	csvWriter.Write([]string{c.Type().String(), c.Name(), dis, protected, c.Host().String(), fmt.Sprint(c.Config().GetInt("port")), fmt.Sprintf("%s:%s", base, underlying), c.Home()})
	return
}

func lsInstanceJSON(c geneos.Instance, params []string) (err error) {
	base, underlying, _ := instance.Version(c)
	lsCmdEntries = append(lsCmdEntries, lsCmdType{c.Type().String(), c.Name(), instance.IsDisabled(c), instance.IsProtected(c), c.Host().String(), c.Config().GetInt64("port"), fmt.Sprintf("%s:%s", base, underlying), c.Home()})
	return
}
