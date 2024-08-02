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
	"os"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(cleanCmd)

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
		CmdNoneMeansAll: "true",
		CmdRequireHome:  "true",
		CmdGlobNames:    "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, names := ParseTypeNames(command)
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)
			resp.Err = instance.Clean(i, cleanCmdFull)
			return
		}).Write(os.Stdout)
	},
}
