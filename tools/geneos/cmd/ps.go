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
	"encoding/json"
	"fmt"
	"os"
	"os/user"
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
	User      string
	Group     string
	Starttime string
	Version   string
	Home      string
}

var psCmdShowFiles, psCmdJSON, psCmdIndent, psCmdCSV bool

var psTabWriter *tabwriter.Writer
var psCSVWriter *csv.Writer
var psJSONEncoder *json.Encoder

func init() {
	GeneosCmd.AddCommand(psCmd)

	psCmd.Flags().BoolVarP(&psCmdShowFiles, "files", "f", false, "Show open files")
	psCmd.Flags().MarkHidden("files")
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
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return CommandPS(ct, args, params)
	},
}

// CommandPS writes running instance information to STDOUT
//
// XXX relies on global flags
func CommandPS(ct *geneos.Component, args []string, params []string) (err error) {
	switch {
	case psCmdJSON, psCmdIndent:
		psJSONEncoder = json.NewEncoder(os.Stdout)
		if psCmdIndent {
			psJSONEncoder.SetIndent("", "    ")
		}
		err = instance.ForAll(ct, Hostname, psInstanceJSON, args, params)
	case psCmdCSV:
		psCSVWriter = csv.NewWriter(os.Stdout)
		psCSVWriter.Write([]string{"Type", "Name", "Host", "PID", "User", "Group", "Starttime", "Version", "Home"})
		err = instance.ForAll(ct, Hostname, psInstanceCSV, args, params)
		psCSVWriter.Flush()
	default:
		psTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintf(psTabWriter, "Type\tName\tHost\tPID\tPorts\tUser\tGroup\tStarttime\tVersion\tHome\n")
		err = instance.ForAll(ct, Hostname, psInstancePlain, args, params)
		psTabWriter.Flush()
	}
	if err == os.ErrNotExist {
		err = nil
	}
	return
}

func psInstancePlain(c geneos.Instance, params []string) (err error) {
	if instance.IsDisabled(c) {
		return nil
	}
	pid, uid, gid, mtime, err := instance.GetPIDInfo(c)
	if err != nil {
		return nil
	}

	var u *user.User
	var g *user.Group

	username := fmt.Sprint(uid)
	groupname := fmt.Sprint(gid)

	if u, err = user.LookupId(username); err == nil {
		username = u.Username
	}
	if g, err = user.LookupGroupId(groupname); err == nil {
		groupname = g.Name
	}
	base, underlying, _ := instance.Version(c)
	ports := instance.TCPListenPorts(c)

	fmt.Fprintf(psTabWriter, "%s\t%s\t%s\t%d\t%v\t%s\t%s\t%s\t%s:%s\t%s\n", c.Type(), c.Name(), c.Host(), pid, ports, username, groupname, mtime.Local().Format(time.RFC3339), base, underlying, c.Home())

	if psCmdShowFiles {
		listOpenFiles(c)
	}
	return
}

func psInstanceCSV(c geneos.Instance, params []string) (err error) {
	if instance.IsDisabled(c) {
		return nil
	}
	pid, uid, gid, mtime, err := instance.GetPIDInfo(c)
	if err != nil {
		return nil
	}

	var u *user.User
	var g *user.Group

	username := fmt.Sprint(uid)
	groupname := fmt.Sprint(gid)

	if u, err = user.LookupId(username); err == nil {
		username = u.Username
	}
	if g, err = user.LookupGroupId(groupname); err == nil {
		groupname = g.Name
	}
	base, underlying, _ := instance.Version(c)
	psCSVWriter.Write([]string{c.Type().String(), c.Name(), c.Host().String(), fmt.Sprint(pid), username, groupname, mtime.Local().Format(time.RFC3339), fmt.Sprintf("%s:%s", base, underlying), c.Home()})

	return
}

func psInstanceJSON(c geneos.Instance, params []string) (err error) {
	if instance.IsDisabled(c) {
		return nil
	}
	pid, uid, gid, mtime, err := instance.GetPIDInfo(c)
	if err != nil {
		return nil
	}

	var u *user.User
	var g *user.Group

	username := fmt.Sprint(uid)
	groupname := fmt.Sprint(gid)

	if u, err = user.LookupId(username); err == nil {
		username = u.Username
	}
	if g, err = user.LookupGroupId(groupname); err == nil {
		groupname = g.Name
	}
	base, underlying, _ := instance.Version(c)
	psJSONEncoder.Encode(psType{c.Type().String(), c.Name(), c.Host().String(), fmt.Sprint(pid), username, groupname, mtime.Local().Format(time.RFC3339), fmt.Sprintf("%s:%s", base, underlying), c.Home()})

	return
}
