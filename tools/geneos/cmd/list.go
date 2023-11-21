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
		AnnotationExpand:    "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names := ParseTypeNames(cmd)
		switch {
		case listCmdJSON, listCmdIndent:
			instance.Do(geneos.GetHost(Hostname), ct, names, listInstanceJSON).Write(os.Stdout, instance.WriterIndent(listCmdIndent))
		case listCmdCSV:
			listCSVWriter := csv.NewWriter(os.Stdout)
			listCSVWriter.Write([]string{"Type", "Name", "Host", "Disabled", "Protected", "AutoStart", "Port", "Version", "Home"})
			instance.Do(geneos.GetHost(Hostname), ct, names, listInstanceCSV).Write(listCSVWriter)
		default:
			listTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(listTabWriter, "Type\tName\tHost\tFlags\tPort\tVersion\tHome\n")
			instance.Do(geneos.GetHost(Hostname), ct, names, listInstancePlain).Write(listTabWriter)
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func listInstancePlain(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	var flags string
	if instance.IsDisabled(i) {
		flags += "D"
	}
	if instance.IsProtected(i) {
		flags += "P"
	}
	if instance.IsAutoStart(i) {
		flags += "A"
	}
	if flags == "" {
		flags = "-"
	}
	base, underlying, _ := instance.Version(i)
	if pkgtype := i.Config().GetString("pkgtype"); pkgtype != "" {
		base = path.Join(pkgtype, base)
	}

	resp.Line = fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s:%s\t%s", i.Type(), i.Name(), i.Host(), flags, i.Config().GetInt("port"), base, underlying, i.Home())
	return
}

func listInstanceCSV(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	disabled := "N"
	protected := "N"
	autostart := "N"

	if instance.IsDisabled(i) {
		disabled = "Y"
	}
	if instance.IsProtected(i) {
		protected = "Y"
	}
	if instance.IsAutoStart(i) {
		autostart = "Y"
	}
	base, underlying, _ := instance.Version(i)
	resp.Rows = append(resp.Rows, []string{i.Type().String(), i.Name(), i.Host().String(), disabled, protected, autostart, fmt.Sprint(i.Config().GetInt("port")), fmt.Sprintf("%s:%s", base, underlying), i.Home()})
	return
}

func listInstanceJSON(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	base, underlying, _ := instance.Version(i)
	resp.Value = listCmdType{
		Type:      i.Type().String(),
		Name:      i.Name(),
		Host:      i.Host().String(),
		Disabled:  instance.IsDisabled(i),
		Protected: instance.IsProtected(i),
		AutoStart: instance.IsAutoStart(i),
		Port:      i.Config().GetInt64("port"),
		Version:   fmt.Sprintf("%s:%s", base, underlying),
		Home:      i.Home(),
	}
	return
}
