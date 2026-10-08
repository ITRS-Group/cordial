/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos/commands"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

var snapshotCmdValue, snapshotCmdSeverity, snapshotCmdSnooze, snapshotCmdUserAssignment, snapshotCmdXpathsonly bool
var snapshotCmdMaxitems int
var snapshotCmdUsername string
var snapshotCmdPassword config.Secret
var snapshotCmdFormat string

func init() {
	Cmd.AddCommand(snapshotCmd)

	snapshotCmd.Flags().SortFlags = false

	snapshotCmd.Flags().BoolVarP(&snapshotCmdValue, "value", "V", true, "Request cell values")
	snapshotCmd.Flags().BoolVarP(&snapshotCmdSeverity, "severity", "S", false, "Request cell severities")
	snapshotCmd.Flags().BoolVarP(&snapshotCmdSnooze, "snooze", "Z", false, "Request cell snooze info")
	snapshotCmd.Flags().BoolVarP(&snapshotCmdUserAssignment, "userassignment", "U", false, "Request cell user assignment info")

	snapshotCmd.Flags().StringVarP(&snapshotCmdUsername, "username", "u", "", "Username")
	snapshotCmd.Flags().StringVarP(&snapshotCmdFormat, "format", "f", "json", "Output format (json, toolkit, toolkit-multi)")

	snapshotCmd.Flags().IntVarP(&snapshotCmdMaxitems, "limit", "l", 0, "limit matching items to display. default is unlimited. results unsorted.")
	snapshotCmd.Flags().BoolVarP(&snapshotCmdXpathsonly, "xpaths", "x", false, "just show matching xpaths")

	snapshotCmd.Flags().SortFlags = false
}

//go:embed _docs/snapshot.md
var snapshotCmdDescription string

var snapshotCmd = &cobra.Command{
	Use:          "snapshot [flags] [gateway] [NAME] XPATH...",
	GroupID:      CommandGroupOther,
	Short:        "Capture a snapshot of each matching dataview",
	Long:         snapshotCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		// CmdComponent:   "gateway",
		CmdGlobal:                "false",
		CmdRequireHome:           "true",
		CmdWildcardNames:         "true",
		CmdAllInstancesMustMatch: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		var err error
		ct, names, params, err := FetchArgs(cmd)
		if err != nil {
			return
		}
		if ct == nil {
			ct = geneos.ParseComponent("gateway")
		} else if !ct.IsA("gateway") {
			fmt.Println("snapshots are only valid for gateways")
			return
		}

		if len(names) == 0 {
			fmt.Println(`no gateway name(s) supplied. Use a NAME of "all" as an explicit wildcard`)
			return
		}
		if len(params) == 0 {
			fmt.Printf("no dataview xpath(s) supplied\n")
			return
		}

		cf := config.Global()
		if snapshotCmdUsername == "" {
			snapshotCmdUsername = config.Get[string](cf, cf.Join("snapshot", "username"))
		}

		snapshotCmdPassword = config.Get[config.Secret](cf, cf.Join("snapshot", "password"))

		if snapshotCmdUsername != "" && snapshotCmdPassword == nil {
			snapshotCmdPassword, err = config.ReadPasswordInput(false, 0)
			if err == config.ErrNotInteractive {
				fmt.Printf("not running interactive and password required")
				return
			}
			defer clear(snapshotCmdPassword)
		}

		var resp responses.GeneralResponses

		switch {
		case snapshotCmdFormat == "toolkit":
			resp = instance.Do(geneos.GetHost(Hostname), ct, names, snapshotInstanceToolkit, params)
		case snapshotCmdFormat == "toolkit-multi":
			resp = instance.Do(geneos.GetHost(Hostname), ct, names, snapshotInstanceToolkitMulti, params)
		default:
			resp = instance.Do(geneos.GetHost(Hostname), ct, names, snapshotInstanceJSON, params)
		}
		resp.Report(os.Stdout, responses.IndentJSON(true))
	},
}

func snapshotInstanceJSON(i geneos.Instance, params ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	paths, ok := params[0].([]string)
	if !ok {
		panic("wrong type")
	}

	// values := []any{}

	values, err := instance.SnapshotDataviews(i, paths,
		instance.SnapshotUsername(snapshotCmdUsername),
		instance.SnapshotPassword(snapshotCmdPassword),
		instance.SnapshotScopes(commands.Scope{
			Value:          snapshotCmdValue,
			Severity:       snapshotCmdSeverity,
			Snooze:         snapshotCmdSnooze,
			UserAssignment: snapshotCmdUserAssignment,
		}),
		instance.SnapshotMaxItems(snapshotCmdMaxitems),
		instance.SnapshotXPathOnly(snapshotCmdXpathsonly),
	)
	if err != nil {
		resp.Err = err
		return
	}

	if len(values) > 0 {
		resp.Value = values
	}
	return
}

// snapshotInstanceToolkit handles snapshot requests for the toolkit
// format. Only the first matching dataview is rendered, even though all
// paths matching the first params element (as a []string) are
// considered, which may impact performance.
func snapshotInstanceToolkit(i geneos.Instance, params ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	paths, ok := params[0].([]string)
	if !ok {
		panic("wrong type")
	}

	dataviews, err := instance.SnapshotDataviews(i, paths,
		instance.SnapshotUsername(snapshotCmdUsername),
		instance.SnapshotPassword(snapshotCmdPassword),
		instance.SnapshotMaxItems(snapshotCmdMaxitems),
	)
	if err != nil {
		resp.Err = err
		return
	}

	// render as Toolkit CSV format
	if len(dataviews) > 0 {
		dv := dataviews[0]
		columns := dv.ColumnOrder
		rows := dv.RowOrder
		headlines := dv.HeadlineOrder

		if len(rows) > 0 {
			fmt.Println(strings.Join(columns, ","))
		} else {
			fmt.Println("rowname")
		}
		for _, headline := range headlines {
			fmt.Printf("<!>%s,%s\n", headline, strings.ReplaceAll(dv.Headlines[headline].Value, ",", "\\,"))
		}
		if len(rows) == 0 {
			return
		}
		for _, row := range rows {
			values := []string{strings.ReplaceAll(row, ",", "\\,")}
			r := dv.Table[row]
			for _, column := range columns[1:] {
				values = append(values, strings.ReplaceAll(r[column].Value, ",", "\\,"))
			}
			fmt.Println(strings.Join(values, ","))
		}
	}
	return
}

func snapshotInstanceToolkitMulti(i geneos.Instance, params ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	paths, ok := params[0].([]string)
	if !ok {
		panic("wrong type")
	}

	dataviews, err := instance.SnapshotDataviews(i, paths,
		instance.SnapshotUsername(snapshotCmdUsername),
		instance.SnapshotPassword(snapshotCmdPassword),
		instance.SnapshotMaxItems(snapshotCmdMaxitems),
	)
	if err != nil {
		resp.Err = err
		return
	}

	// render as Toolkit CSV format
	for _, dv := range dataviews {
		columns := dv.ColumnOrder
		rows := dv.RowOrder
		headlines := dv.HeadlineOrder

		fmt.Println("<dataview>" + dv.Name)
		if len(rows) > 0 {
			fmt.Println("<h>" + strings.Join(columns, ","))
		} else {
			fmt.Println("<h>rowname")
		}
		for _, headline := range headlines {
			fmt.Printf("<!>%s,%s\n", headline, strings.ReplaceAll(dv.Headlines[headline].Value, ",", "\\,"))
		}
		if len(rows) == 0 {
			i.Log().Debug("no rows available", slog.String("dataview", dv.Name))
			continue
		}
		for _, row := range rows {
			values := []string{strings.ReplaceAll(row, ",", "\\,")}
			r := dv.Table[row]
			for _, column := range columns[1:] {
				values = append(values, strings.ReplaceAll(r[column].Value, ",", "\\,"))
			}
			fmt.Println(strings.Join(values, ","))
		}
	}
	return
}
