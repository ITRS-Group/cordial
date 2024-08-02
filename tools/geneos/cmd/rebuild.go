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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var rebuildCmdForce, rebuildCmdReload bool

func init() {
	GeneosCmd.AddCommand(rebuildCmd)

	rebuildCmd.Flags().BoolVarP(&rebuildCmdForce, "force", "F", false, "Force rebuild")
	rebuildCmd.Flags().BoolVarP(&rebuildCmdReload, "reload", "r", false, "Reload instances after rebuild")
	rebuildCmd.Flags().SortFlags = false
}

//go:embed _docs/rebuild.md
var rebuildCmdDescription string

var rebuildCmd = &cobra.Command{
	Use:          "rebuild [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupConfig,
	Short:        "Rebuild Instance Configurations From Templates",
	Long:         rebuildCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdNoneMeansAll: "true",
		CmdRequireHome:  "true",
		CmdGlobNames:    "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := ParseTypeNames(cmd)
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)

			if resp.Err = i.Rebuild(rebuildCmdForce); resp.Err != nil {
				return
			}
			resp.Completed = append(resp.Completed, "configuration rebuilt")
			log.Debug().Msgf("%s configuration rebuilt (if supported)", i)
			if !rebuildCmdReload {
				return
			}
			resp2 := ReloadInstance(i)
			return instance.MergeResponse(resp, resp2)
		}).Write(os.Stdout)
	},
}
