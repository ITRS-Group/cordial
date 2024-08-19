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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

var stopCmdForce, stopCmdKill bool

func init() {
	GeneosCmd.AddCommand(stopCmd)

	stopCmd.Flags().BoolVarP(&stopCmdForce, "force", "F", false, "Stop protected instances")
	stopCmd.Flags().BoolVarP(&stopCmdKill, "kill", "K", false, "Force immediate stop by sending an immediate SIGKILL")

	stopCmd.Flags().SortFlags = false
}

//go:embed _docs/stop.md
var stopCmdDescription string

var stopCmd = &cobra.Command{
	Use:          "stop [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupProcess,
	Short:        "Stop Instances",
	Long:         stopCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := ParseTypeNames(cmd)
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, a ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)
			resp.Err = instance.Stop(i, stopCmdForce, stopCmdKill)
			return
		}).Write(os.Stdout,
			instance.WriterShowTimes(),
			instance.WriterTimingFormat("%s stopped in %.2fs\n"),
		)
	},
}
