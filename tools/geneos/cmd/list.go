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
	_ "embed"
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"text/tabwriter"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

type listCmdType struct {
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	Host      string `json:"host,omitempty"`
	Disabled  bool   `json:"disabled"`
	Protected bool   `json:"protected"`
	AutoStart bool   `json:"autostart"`
	Port      int64  `json:"port,omitempty"`
	Version   string `json:"version,omitempty"`
	Home      string `json:"home,omitempty"`
}

var listCmdJSON, listCmdCSV, listCmdIndent bool

var listTabWriter *tabwriter.Writer
var listCSVWriter *csv.Writer

func init() {
	GeneosCmd.AddCommand(listCmd)

	listCmd.PersistentFlags().BoolVarP(&listCmdJSON, "json", "j", false, "Output JSON")
	listCmd.PersistentFlags().BoolVarP(&listCmdIndent, "pretty", "i", false, "Output indented JSON")
	listCmd.PersistentFlags().BoolVarP(&listCmdCSV, "csv", "c", false, "Output CSV")

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
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names := TypeNames(cmd)
		switch {
		case listCmdJSON, listCmdIndent:
			results := instance.Do(geneos.GetHost(Hostname), ct, names, listInstanceJSON)
			results.Write(os.Stdout, instance.WriterIndent(listCmdIndent))
		case listCmdCSV:
			listCSVWriter = csv.NewWriter(os.Stdout)
			listCSVWriter.Write([]string{"Type", "Name", "Host", "Disabled", "Protected", "AutoStart", "Port", "Version", "Home"})
			results := instance.Do(geneos.GetHost(Hostname), ct, names, listInstanceCSV)
			results.Write(listCSVWriter)
			listCSVWriter.Flush()
		default:
			results := instance.Do(geneos.GetHost(Hostname), ct, names, listInstancePlain)
			listTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(listTabWriter, "Type\tNames\tHost\tFlag\tPort\tVersion\tHome\n")
			results.Write(listTabWriter)
			listTabWriter.Flush()
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func listInstancePlain(c geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	var flags string
	if instance.IsDisabled(c) {
		flags += "D"
	}
	if instance.IsProtected(c) {
		flags += "P"
	}
	if instance.IsAutoStart(c) {
		flags += "A"
	}
	if flags == "" {
		flags = "-"
	}
	base, underlying, _ := instance.Version(c)
	if pkgtype := c.Config().GetString("pkgtype"); pkgtype != "" {
		base = path.Join(pkgtype, base)
	}

	resp.Line = fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s:%s\t%s", c.Type(), c.Name(), c.Host(), flags, c.Config().GetInt("port"), base, underlying, c.Home())
	return
}

func listInstanceCSV(c geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	disabled := "N"
	protected := "N"
	autostart := "N"

	if instance.IsDisabled(c) {
		disabled = "Y"
	}
	if instance.IsProtected(c) {
		protected = "Y"
	}
	if instance.IsAutoStart(c) {
		autostart = "Y"
	}
	base, underlying, _ := instance.Version(c)
	resp.Row = []string{c.Type().String(), c.Name(), c.Host().String(), disabled, protected, autostart, fmt.Sprint(c.Config().GetInt("port")), fmt.Sprintf("%s:%s", base, underlying), c.Home()}
	return
}

func listInstanceJSON(c geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	base, underlying, _ := instance.Version(c)
	resp.Value = listCmdType{
		Type:      c.Type().String(),
		Name:      c.Name(),
		Host:      c.Host().String(),
		Disabled:  instance.IsDisabled(c),
		Protected: instance.IsProtected(c),
		AutoStart: instance.IsAutoStart(c),
		Port:      c.Config().GetInt64("port"),
		Version:   fmt.Sprintf("%s:%s", base, underlying),
		Home:      c.Home(),
	}
	return
}
