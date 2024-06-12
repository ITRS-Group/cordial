/*
Copyright © 2023 ITRS Group

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

var protectCmdUnprotect bool

func init() {
	GeneosCmd.AddCommand(protectCmd)

	protectCmd.Flags().BoolVarP(&protectCmdUnprotect, "unprotect", "U", false, "unprotect instances")
}

//go:embed _docs/protect.md
var protectCmdDescription string

var protectCmd = &cobra.Command{
	Use:          "protect [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupManage,
	Short:        "Mark instances as protected",
	Long:         protectCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	DisableFlagsInUseLine: true,
	Run: func(command *cobra.Command, _ []string) {
		ct, args := ParseTypeNames(command)
		instance.Do(geneos.GetHost(Hostname), ct, args, func(i geneos.Instance, params ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)
			cf := i.Config()

			if len(params) == 0 {
				resp.Err = geneos.ErrInvalidArgs
				return
			}
			protect, ok := params[0].(bool)
			if !ok {
				panic("wrong param")
			}

			cf.Set("protected", protect)

			if cf.Type == "rc" {
				resp.Err = instance.Migrate(i)
			} else {
				resp.Err = instance.SaveConfig(i)
			}
			if resp.Err != nil {
				return
			}

			if protect {
				resp.Completed = append(resp.Completed, "protected")
			} else {
				resp.Completed = append(resp.Completed, "unprotected")
			}

			return
		}, !protectCmdUnprotect).Write(os.Stdout)
	},
}
