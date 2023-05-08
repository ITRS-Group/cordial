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
	"os/user"
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
	RootCmd.AddCommand(psCmd)

	psCmd.Flags().BoolVarP(&psCmdShowFiles, "files", "f", false, "Show open files")
	psCmd.Flags().BoolVarP(&psCmdJSON, "json", "j", false, "Output JSON")
	psCmd.Flags().BoolVarP(&psCmdIndent, "pretty", "i", false, "Output indented JSON")
	psCmd.Flags().BoolVarP(&psCmdCSV, "csv", "c", false, "Output CSV")

	psCmd.Flags().SortFlags = false
}

var psCmd = &cobra.Command{
	Use:   "ps [flags] [TYPE] [NAMES...]",
	Short: "List process information for instances, optionally in CSV or JSON format",
	Long: strings.ReplaceAll(`
Show the status of the matching instances.
`, "|", "`"),
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

func CommandPS(ct *geneos.Component, args []string, params []string) (err error) {
	switch {
	case psCmdJSON, psCmdIndent:
		psJSONEncoder = json.NewEncoder(os.Stdout)
		if psCmdIndent {
			psJSONEncoder.SetIndent("", "    ")
		}
		err = instance.ForAll(ct, psInstanceJSON, args, params)
	case psCmdCSV:
		psCSVWriter = csv.NewWriter(os.Stdout)
		psCSVWriter.Write([]string{"Type", "Name", "Host", "PID", "User", "Group", "Starttime", "Version", "Home"})
		err = instance.ForAll(ct, psInstanceCSV, args, params)
		psCSVWriter.Flush()
	default:
		psTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintf(psTabWriter, "Type\tName\tHost\tPID\tPorts\tUser\tGroup\tStarttime\tVersion\tHome\n")
		err = instance.ForAll(ct, psInstancePlain, args, params)
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

	// if psCmdShowFiles {
	// 	// list open files (test code)

	// 	instdir := c.Home()
	// 	files := instance.Files(c)
	// 	fds := make([]int, len(files))
	// 	i := 0
	// 	for f := range files {
	// 		fds[i] = f
	// 		i++
	// 	}
	// 	sort.Ints(fds)
	// 	for _, n := range fds {
	// 		fdPath := files[n].FD
	// 		perms := ""
	// 		p := files[n].FDMode & 0700
	// 		log.Debug().Msgf("%s perms %o", fdPath, p)
	// 		if p&0400 == 0400 {
	// 			perms += "r"
	// 		}
	// 		if p&0200 == 0200 {
	// 			perms += "w"
	// 		}

	// 		path := files[n].Path
	// 		if strings.HasPrefix(path, instdir) {
	// 			path = strings.Replace(path, instdir, ".", 1)
	// 		}
	// 		fmt.Fprintf(psTabWriter, "\t%d:%s (%d bytes) %s\n", n, perms, files[n].Stat.Size(), path)
	// 	}
	// }
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
