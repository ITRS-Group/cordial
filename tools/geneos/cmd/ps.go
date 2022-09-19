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
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// psCmd represents the ps command
var psCmd = &cobra.Command{
	Use:                   "ps [-c|-j [-i]] [TYPE] [NAMES...]",
	Short:                 "List process information for instances, optionally in CSV or JSON format",
	Long:                  `Show the status of the matching instances.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandPS(ct, args, params)
	},
}

func init() {
	rootCmd.AddCommand(psCmd)

	psCmd.PersistentFlags().BoolVarP(&psCmdJSON, "json", "j", false, "Output JSON")
	psCmd.PersistentFlags().BoolVarP(&psCmdIndent, "pretty", "i", false, "Indent / pretty print JSON")
	psCmd.PersistentFlags().BoolVarP(&psCmdCSV, "csv", "c", false, "Output CSV")
	psCmd.Flags().SortFlags = false
}

var psCmdJSON, psCmdIndent, psCmdCSV bool

var psTabWriter *tabwriter.Writer

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

func commandPS(ct *geneos.Component, args []string, params []string) (err error) {
	switch {
	case psCmdJSON:
		jsonEncoder = json.NewEncoder(os.Stdout)
		//jsonEncoder.SetIndent("", "    ")
		err = instance.ForAll(ct, psInstanceJSON, args, params)
	case psCmdCSV:
		csvWriter = csv.NewWriter(os.Stdout)
		csvWriter.Write([]string{"Type", "Name", "Host", "PID", "User", "Group", "Starttime", "Version", "Home"})
		err = instance.ForAll(ct, psInstanceCSV, args, params)
		csvWriter.Flush()
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
	ports := instance.Ports(c)

	fmt.Fprintf(psTabWriter, "%s\t%s\t%s\t%d\t%v\t%s\t%s\t%s\t%s:%s\t%s\n", c.Type(), c.Name(), c.Host(), pid, ports, username, groupname, time.Unix(mtime, 0).Local().Format(time.RFC3339), base, underlying, c.Home())

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
	csvWriter.Write([]string{c.Type().String(), c.Name(), c.Host().String(), fmt.Sprint(pid), username, groupname, time.Unix(mtime, 0).Local().Format(time.RFC3339), fmt.Sprintf("%s:%s", base, underlying), c.Home()})

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
	jsonEncoder.Encode(psType{c.Type().String(), c.Name(), c.Host().String(), fmt.Sprint(pid), username, groupname, time.Unix(mtime, 0).Local().Format(time.RFC3339), fmt.Sprintf("%s:%s", base, underlying), c.Home()})

	return
}
