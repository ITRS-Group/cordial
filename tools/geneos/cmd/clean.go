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
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"

	"github.com/spf13/cobra"
)

var cleanCmdFull bool

func init() {
	Cmd.AddCommand(cleanCmd)

	cleanCmd.Flags().BoolVarP(&cleanCmdFull, "full", "F", false, "Perform a full clean. Removes more files than basic clean and restarts instances")
	cleanCmd.Flags().MarkDeprecated("full", "Please use the `reset` command instead `clean --full`")

	cleanCmd.Flags().SortFlags = false
}

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
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *responses.General) {
			resp = responses.NewResponse(i)
			resp.Completed = append(resp.Completed, "configured files and directories removed")
			resp.Err = instance.Clean(i, instance.FullClean(cleanCmdFull))
			return
		}).Report(os.Stdout)
		return nil
	},
}

var resetCmdForce bool

func init() {
	Cmd.AddCommand(resetCmd)

	resetCmd.Flags().BoolVarP(&resetCmdForce, "force", "F", false, "Force reset (require for proteced instances) even if instance is running")

	resetCmd.Flags().SortFlags = false
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
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *responses.General) {
			resp = responses.NewResponse(i)
			resp.Completed = append(resp.Completed, "transient files and directories removed")
			resp.Err = instance.Clean(i, instance.FullClean(true), instance.ForceClean(resetCmdForce))
			return
		}).Report(os.Stdout)
		return nil
	},
}
