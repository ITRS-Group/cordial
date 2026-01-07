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
	"os"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"

	"github.com/spf13/cobra"
)

var commandCmdJSON bool
var commandCmdExtras string
var commandCmdEnvs instance.NameValues

func init() {
	GeneosCmd.AddCommand(commandCmd)

	commandCmd.Flags().StringVarP(&commandCmdExtras, "extras", "x", "", "Extra args passed to process, split on spaces and quoting ignored")
	commandCmd.Flags().VarP(&commandCmdEnvs, "env", "e", "Extra environment variable (Repeat as required)")
	commandCmd.Flags().BoolVarP(&commandCmdJSON, "json", "j", false, "JSON formatted output")
}

//go:embed _docs/command.md
var commandCmdDescription string

var commandCmd = &cobra.Command{
	Use:          "command [TYPE] [NAME...]",
	GroupID:      CommandGroupView,
	Short:        "Show Instance Start-up Details",
	Long:         commandCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names := ParseTypeNames(cmd)
		if commandCmdJSON {
			results := instance.Do(geneos.GetHost(Hostname), ct, names, commandInstanceJSON)
			results.Report(os.Stdout, responses.IndentJSON(true))
			return nil
		}
		results := instance.Do(geneos.GetHost(Hostname), ct, names, commandInstance)
		results.Report(os.Stdout)
		return
	},
}

func commandInstance(i geneos.Instance, params ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	lines := []string{fmt.Sprintf("=== %s ===", i)}

	cmd, err := instance.BuildCmd(i, true, instance.StartingExtras(commandCmdExtras), instance.StartingEnvs(commandCmdEnvs))
	if err != nil {
		resp.Err = err
		return
	}

	cmdLine := ""
	for _, a := range cmd.Args {
		if strings.Contains(a, " ") {
			cmdLine += ` "` + a + `"`
		} else {
			cmdLine += " " + a
		}
	}
	lines = append(lines,
		"command line:",
		fmt.Sprint("\t", cmdLine),
		"",
		"working directory:",
		fmt.Sprint("\t", cmd.Dir),
		"",
		"environment:",
	)

	for _, e := range cmd.Environ() {
		lines = append(lines, fmt.Sprint("\t", e))
	}
	lines = append(lines, "")
	resp.Details = lines
	return
}

type command struct {
	Instance string   `json:"instance"`
	Type     string   `json:"type"`
	Host     string   `json:"host"`
	Path     string   `json:"path"`
	Args     []string `json:"arguments,omitempty"`
	Env      []string `json:"environment,omitempty"`
	Home     string   `json:"directory"`
}

func commandInstanceJSON(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	cmd, err := instance.BuildCmd(i, true, instance.StartingExtras(commandCmdExtras), instance.StartingEnvs(commandCmdEnvs))
	if err != nil {
		resp.Err = err
		return
	}
	command := &command{
		Instance: i.Name(),
		Type:     i.Type().Name,
		Host:     i.Host().String(),
		Path:     cmd.Path,
		Env:      cmd.Environ(),
		Home:     cmd.Dir,
	}
	if len(cmd.Args) > 1 {
		command.Args = cmd.Args[1:]
	}
	resp.Value = command
	return
}
