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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"

	"github.com/spf13/cobra"
)

var restartCmdAll, restartCmdKill, restartCmdForce, restartCmdLogs bool
var restartCmdExtras string
var restartCmdEnvs instance.NameValues
var restartCmdPort uint16

func init() {
	GeneosCmd.AddCommand(restartCmd)

	restartCmd.Flags().BoolVarP(&restartCmdAll, "all", "a", false, "Start all matching instances, not just those already running")
	restartCmd.Flags().BoolVarP(&restartCmdForce, "force", "F", false, "Force restart of protected instances")
	restartCmd.Flags().BoolVarP(&restartCmdKill, "kill", "K", false, "Force stop by sending an immediate SIGKILL")

	restartCmd.Flags().Uint16VarP(&restartCmdPort, "port", "p", 0, "Restart instance matching port (overrides TYPE and NAME)")
	restartCmd.Flags().StringVarP(&restartCmdExtras, "extras", "x", "", "Extra args passed to process, split on spaces and quoting ignored")
	restartCmd.Flags().VarP(&restartCmdEnvs, "env", "e", "Extra environment variable (Repeat as required)")

	restartCmd.Flags().BoolVarP(&restartCmdLogs, "log", "l", false, "Run 'logs -f' after starting instance(s)")

	restartCmd.Flags().SortFlags = false
}

//go:embed _docs/restart.md
var restartCmdDescription string

var restartCmd = &cobra.Command{
	Use:          "restart [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupProcess,
	Short:        "Restart Instances",
	Long:         restartCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:                "true",
		CmdRequireHome:           "true",
		CmdWildcardNames:         "true",
		CmdAllInstancesMustMatch: "true",
		CmdNonInstanceArgsError:  "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		if restartCmdPort != 0 {
			instances := []geneos.Instance{}
			for h := range geneos.GetHost(Hostname).OrList(geneos.LOCAL) {
				i, err := instance.ByPort(h, restartCmdPort)
				if err != nil {
					continue
				}
				instances = append(instances, i)
			}
			if len(instances) == 0 {
				fmt.Printf("no instances using port %d found\n", restartCmdPort)
				return
			}
			instance.DoInstances(instances, func(i geneos.Instance, a ...any) (resp *responses.Response) {
				resp = responses.NewResponse(i)
				resp.Err = instance.Stop(i, restartCmdForce, false)
				if resp.Err == nil || restartCmdAll {
					resp.Err = instance.Start(i, instance.StartingExtras(restartCmdExtras), instance.StartingEnvs(restartCmdEnvs))
				}
				return
			}).Report(os.Stdout)
			return
		}

		ct, names, _, err := FetchArgs(cmd)
		if err != nil {
			return
		}
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, a ...any) (resp *responses.Response) {
			resp = responses.NewResponse(i)
			resp.Err = instance.Stop(i, restartCmdForce, false)
			if resp.Err == nil || restartCmdAll {
				resp.Err = instance.Start(i, instance.StartingExtras(restartCmdExtras), instance.StartingEnvs(restartCmdEnvs))
			}
			return
		}).Report(os.Stdout)

		if restartCmdLogs {
			// also watch STDERR on start-up
			// never returns
			followLogs(ct, names, true)
		}
		return
	},
}
