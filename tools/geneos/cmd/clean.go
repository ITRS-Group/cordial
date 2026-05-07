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

func init() {
	Cmd.AddCommand(cleanCmd)
	Cmd.AddCommand(resetCmd)

	cleanCmd.Flags().BoolVarP(&cleanCmdFull, "full", "F", false, "Perform a full clean. Removes more files than basic clean and restarts instances")
	cleanCmd.Flags().SortFlags = false
}

var cleanCmdFull bool

//go:embed _docs/clean.md
var cleanCmdDescription string

var cleanCmd = &cobra.Command{
	Use:     "clean [flags] [TYPE] [NAME...]",
	GroupID: CommandGroupManage,
	Short:   "Clean-up Instance Directories",
	Long:    cleanCmdDescription,
	Example: strings.ReplaceAll(`
# Delete old logs and config file backups without affecting the running
# instance
geneos clean gateway Gateway1
# Stop all netprobes and remove all non-essential files from working 
# directories, then restart netprobes
geneos clean --full netprobe
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:                "true",
		CmdRequireHome:           "true",
		CmdWildcardNames:         "true",
		CmdAllInstancesMustMatch: "true",
		CmdNonInstanceArgsError:  "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		ct, names, _, err := FetchArgs(command)
		if err != nil {
			return err
		}
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *responses.Response) {
			resp = responses.NewResponse(i)
			resp.Completed = append(resp.Completed, "configured files and directories removed")
			resp.Err = instance.Clean(i, cleanCmdFull)
			return
		}).Report(os.Stdout)
		return nil
	},
}

//go:embed _docs/reset.md
var resetCmdDescription string

var resetCmd = &cobra.Command{
	Use:     "reset [flags] [TYPE] [NAME...]",
	GroupID: CommandGroupManage,
	Short:   "Reset Instance Directories",
	Long:    resetCmdDescription,
	Example: strings.ReplaceAll(`
# Stop all netprobes and remove all non-essential files from working 
# directories, then restart netprobes
geneos reset netprobe
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:                "false",
		CmdRequireHome:           "true",
		CmdWildcardNames:         "true",
		CmdAllInstancesMustMatch: "true",
		CmdNonInstanceArgsError:  "true",
	},
	RunE: func(command *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("you must provide at least the component type or one instance name or the 'all' keyword to match all instances")
		}
		ct, names, _, err := FetchArgs(command)
		if err != nil {
			return err
		}
		if ct == nil && len(names) == 0 {
			return fmt.Errorf("no matching instances")
		}
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *responses.Response) {
			resp = responses.NewResponse(i)
			resp.Completed = append(resp.Completed, "configured files and directories removed")
			resp.Err = instance.Clean(i, true)
			return
		}).Report(os.Stdout)
		return nil
	},
}
