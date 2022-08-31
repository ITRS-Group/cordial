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
	"text/tabwriter"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:                   "ls [-c|-j [-i]] [TYPE] [NAME...]",
	Short:                 "List instances, optionally in CSV or JSON format",
	Long:                  `List the matching instances and their component type.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandLS(ct, args, params)
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.PersistentFlags().BoolVarP(&lsCmdJSON, "json", "j", false, "Output JSON")
	lsCmd.PersistentFlags().BoolVarP(&lsCmdIndent, "pretty", "i", false, "Indent / pretty print JSON")
	lsCmd.PersistentFlags().BoolVarP(&lsCmdCSV, "csv", "c", false, "Output CSV")
	lsCmd.Flags().SortFlags = false
}

var lsCmdJSON, lsCmdCSV, lsCmdIndent bool

var lsTabWriter *tabwriter.Writer
var csvWriter *csv.Writer
var jsonEncoder *json.Encoder

func commandLS(ct *geneos.Component, args []string, params []string) (err error) {
	switch {
	case lsCmdJSON:
		jsonEncoder = json.NewEncoder(log.Writer())
		if lsCmdIndent {
			jsonEncoder.SetIndent("", "    ")
		}
		err = instance.ForAll(ct, lsInstanceJSON, args, params)
	case lsCmdCSV:
		csvWriter = csv.NewWriter(log.Writer())
		csvWriter.Write([]string{"Type", "Name", "Disabled", "Host", "Port", "Version", "Home"})
		err = instance.ForAll(ct, lsInstanceCSV, args, params)
		csvWriter.Flush()
	default:
		lsTabWriter = tabwriter.NewWriter(log.Writer(), 3, 8, 2, ' ', 0)
		fmt.Fprintf(lsTabWriter, "Type\tName\tHost\tPort\tVersion\tHome\n")
		err = instance.ForAll(ct, lsInstancePlain, args, params)
		lsTabWriter.Flush()
	}
	if err == os.ErrNotExist {
		err = nil
	}
	return
}

func lsInstancePlain(c geneos.Instance, params []string) (err error) {
	var suffix string
	if instance.IsDisabled(c) {
		suffix = "*"
	}
	base, underlying, _ := instance.Version(c)
	fmt.Fprintf(lsTabWriter, "%s\t%s\t%s\t%d\t%s:%s\t%s\n", c.Type(), c.Name()+suffix, c.Host(), c.GetConfig().GetInt("port"), base, underlying, c.Home())
	return
}

func lsInstanceCSV(c geneos.Instance, params []string) (err error) {
	var dis string = "N"
	if instance.IsDisabled(c) {
		dis = "Y"
	}
	base, underlying, _ := instance.Version(c)
	csvWriter.Write([]string{c.Type().String(), c.Name(), dis, c.Host().String(), fmt.Sprint(c.GetConfig().GetInt("port")), fmt.Sprintf("%s:%s", base, underlying), c.Home()})
	return
}

type lsType struct {
	Type     string
	Name     string
	Disabled string
	Host     string
	Port     int64
	Version  string
	Home     string
}

func lsInstanceJSON(c geneos.Instance, params []string) (err error) {
	var dis string = "N"
	if instance.IsDisabled(c) {
		dis = "Y"
	}
	base, underlying, _ := instance.Version(c)
	jsonEncoder.Encode(lsType{c.Type().String(), c.Name(), dis, c.Host().String(), c.GetConfig().GetInt64("port"), fmt.Sprintf("%s:%s", base, underlying), c.Home()})
	return
}
