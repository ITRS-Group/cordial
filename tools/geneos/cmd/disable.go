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
	"errors"
	"fmt"
	"os"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

var disableCmdForce, disableCmdStop bool

func init() {
	GeneosCmd.AddCommand(disableCmd)

	disableCmd.Flags().BoolVarP(&disableCmdStop, "stop", "S", false, "Stop instances")
	disableCmd.Flags().BoolVarP(&disableCmdForce, "force", "F", false, "Force disable instances")
	disableCmd.Flags().SortFlags = false
}

//go:embed _docs/disable.md
var disableCmdDescription string

var disableCmd = &cobra.Command{
	Use:          "disable [TYPE] [NAME...]",
	GroupID:      CommandGroupManage,
	Short:        "Disable instances",
	Long:         disableCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdNoneMeansAll: "explicit",
		CmdRequireHome:  "true",
		CmdGlobNames:    "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := ParseTypeNames(cmd)
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, a ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)

			if instance.IsDisabled(i) {
				return
			}

			if instance.IsProtected(i) && !disableCmdForce {
				resp.Err = geneos.ErrProtected
				return
			}

			if disableCmdStop || disableCmdForce {
				if i.Type() != &geneos.RootComponent {
					if err := instance.Stop(i, true, false); err != nil && !errors.Is(err, os.ErrProcessDone) {
						resp.Err = err
						return
					}
				}
			}

			if !disableCmdForce && instance.IsRunning(i) {
				fmt.Printf("%s is running, skipping. Use the `--stop` option to stop running instances\n", i)
				return
			}

			if !instance.IsProtected(i) || disableCmdForce {
				if resp.Err = instance.Disable(i); resp.Err == nil {
					resp.Completed = append(resp.Completed, "disabled")
					return
				}
			}

			resp.Err = fmt.Errorf("not disabled. Instances must not be running or use the '--force'/'-F' option")
			return
		}).Write(os.Stdout)
	},
}
