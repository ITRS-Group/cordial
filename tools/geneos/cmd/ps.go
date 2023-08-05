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
	"os/user"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

type psType struct {
	Type      string
	Name      string
	Host      string
	PID       string
	Ports     []int
	User      string
	Group     string
	Starttime string
	Version   string
	Home      string
}

var psCmdLong, psCmdShowFiles, psCmdJSON, psCmdIndent, psCmdCSV, psCmdNoLookups bool

var psTabWriter *tabwriter.Writer
var psCSVWriter *csv.Writer

func init() {
	GeneosCmd.AddCommand(psCmd)

	psCmd.Flags().BoolVarP(&psCmdShowFiles, "files", "f", false, "Show open files")
	psCmd.Flags().MarkHidden("files")

	psCmd.Flags().BoolVarP(&psCmdLong, "long", "l", false, "Show more output (remote ports etc.)")
	psCmd.Flags().BoolVarP(&psCmdNoLookups, "nolookup", "n", false, "No lookups for user/groups")

	psCmd.Flags().BoolVarP(&psCmdJSON, "json", "j", false, "Output JSON")
	psCmd.Flags().BoolVarP(&psCmdIndent, "pretty", "i", false, "Output indented JSON")
	psCmd.Flags().BoolVarP(&psCmdCSV, "csv", "c", false, "Output CSV")

	psCmd.Flags().SortFlags = false
}

//go:embed _docs/ps.md
var psCmdDescription string

var psCmd = &cobra.Command{
	Use:          "ps [flags] [TYPE] [NAMES...]",
	GroupID:      CommandGroupView,
	Short:        "List Running Instance Details",
	Long:         psCmdDescription,
	Aliases:      []string{"status"},
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, names, params := TypeNamesParams(cmd)
		return CommandPS(ct, names, params)
	},
}

// CommandPS writes running instance information to STDOUT
//
// XXX relies on global flags
func CommandPS(ct *geneos.Component, names []string, params []string) (err error) {
	switch {
	case psCmdJSON, psCmdIndent:
		responses := instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceJSON)
		responses.Write(os.Stdout, instance.WriterIndent(psCmdIndent))
	case psCmdCSV:
		psCSVWriter = csv.NewWriter(os.Stdout)
		psCSVWriter.Write([]string{"Type", "Name", "Host", "PID", "Ports", "User", "Group", "Starttime", "Version", "Home"})
		results := instance.Do(geneos.GetHost(Hostname), ct, names, psInstanceCSV)
		results.Write(psCSVWriter)
		psCSVWriter.Flush()
	default:
		results := instance.Do(geneos.GetHost(Hostname), ct, names, psInstancePlain)
		psTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintf(psTabWriter, "Type\tName\tHost\tPID\tPorts\tUser\tGroup\tStarttime\tVersion\tHome\n")
		results.Write(psTabWriter)
		psTabWriter.Flush()
	}
	if err == os.ErrNotExist {
		err = nil
	}
	return
}

func psInstancePlain(c geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	if instance.IsDisabled(c) {
		return
	}
	pid, uid, gid, mtime, err := instance.GetPIDInfo(c)
	if err != nil {
		return
	}

	var u *user.User
	var g *user.Group

	username := fmt.Sprint(uid)
	groupname := fmt.Sprint(gid)

	if !psCmdNoLookups {
		if u, err = user.LookupId(username); err == nil {
			username = u.Username
		}
		if g, err = user.LookupGroupId(groupname); err == nil {
			groupname = g.Name
		}
	}
	base, underlying, actual, _ := instance.LiveVersion(c, pid)
	if pkgtype := c.Config().GetString("pkgtype"); pkgtype != "" {
		base = path.Join(pkgtype, base)
	}

	var portlist string
	if c.Host().IsLocal() || psCmdLong {
		portlist = strings.Join(instance.ListeningPortsStrings(c), " ")
	}
	if !c.Host().IsLocal() && portlist == "" {
		portlist = "..."
	}
	if underlying != actual {
		base += "*"
	}

	resp.Line = fmt.Sprintf("%s\t%s\t%s\t%d\t[%s]\t%s\t%s\t%s\t%s:%s\t%s", c.Type(), c.Name(), c.Host(), pid, portlist, username, groupname, mtime.Local().Format(time.RFC3339), base, actual, c.Home())

	// if psCmdShowFiles {
	// 	listOpenFiles(c)
	// }
	return
}

func psInstanceCSV(c geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	if instance.IsDisabled(c) {
		return
	}
	pid, uid, gid, mtime, err := instance.GetPIDInfo(c)
	if err != nil {
		err = nil // skip
		return
	}

	var u *user.User
	var g *user.Group

	username := fmt.Sprint(uid)
	groupname := fmt.Sprint(gid)

	if !psCmdNoLookups {
		if u, err = user.LookupId(username); err == nil {
			username = u.Username
		}
		if g, err = user.LookupGroupId(groupname); err == nil {
			groupname = g.Name
		}
	}
	ports := []string{}
	if c.Host().IsLocal() || psCmdLong {
		ports = instance.ListeningPortsStrings(c)
	}
	portlist := strings.Join(ports, ":")
	base, underlying, actual, _ := instance.LiveVersion(c, pid)
	if underlying != actual {
		base += "*"
	}
	resp.Row = []string{c.Type().String(), c.Name(), c.Host().String(), fmt.Sprint(pid), portlist, username, groupname, mtime.Local().Format(time.RFC3339), fmt.Sprintf("%s:%s", base, actual), c.Home()}

	return
}

func psInstanceJSON(c geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	if instance.IsDisabled(c) {
		return
	}
	pid, uid, gid, mtime, err := instance.GetPIDInfo(c)
	if err != nil {
		// skip errors for now
		return
	}

	var u *user.User
	var g *user.Group

	username := fmt.Sprint(uid)
	groupname := fmt.Sprint(gid)

	if !psCmdNoLookups {
		if u, err = user.LookupId(username); err == nil {
			username = u.Username
		}
		if g, err = user.LookupGroupId(groupname); err == nil {
			groupname = g.Name
		}
	}
	ports := []int{}
	if c.Host().IsLocal() || psCmdLong {
		ports = instance.ListeningPorts(c)
	}
	base, underlying, actual, _ := instance.LiveVersion(c, pid)
	if underlying != actual {
		base += "*"
	}

	resp.Value = psType{
		Type:      c.Type().String(),
		Name:      c.Name(),
		Host:      c.Host().String(),
		PID:       fmt.Sprint(pid),
		Ports:     ports,
		User:      username,
		Group:     groupname,
		Starttime: mtime.Local().Format(time.RFC3339),
		Version:   fmt.Sprintf("%s:%s", base, actual),
		Home:      c.Home(),
	}

	return
}
