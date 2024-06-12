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

func init() {
	GeneosCmd.AddCommand(enableCmd)

	enableCmd.Flags().BoolVarP(&enableCmdStart, "start", "S", false, "Start enabled instances")

	enableCmd.Flags().SortFlags = false
}

var enableCmdStart bool

//go:embed _docs/enable.md
var enableCmdDescription string

var enableCmd = &cobra.Command{
	Use:          "enable [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupManage,
	Short:        "Enable instance",
	Long:         enableCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "explicit",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := ParseTypeNames(cmd)
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)

			if !instance.IsDisabled(i) {
				return
			}
			if resp.Err = instance.Enable(i); resp.Err == nil {
				resp.Completed = append(resp.Completed, "enabled")
				if enableCmdStart {
					resp.Err = instance.Start(i)
					return
				}
			}
			return
		}).Write(os.Stdout)
	},
}
