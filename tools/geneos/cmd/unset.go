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

var unsetCmdWarned bool

var unsetCmdValues = instance.UnsetConfigValues{}

func init() {
	GeneosCmd.AddCommand(unsetCmd)

	unsetCmd.Flags().VarP(&unsetCmdValues.Keys, "key", "k", "Unset configuration parameter `KEY`\n(Repeat as required)")

	unsetCmd.Flags().VarP(&unsetCmdValues.Envs, "env", "e", "Remove an environment variable `NAME`\n(Repeat as required)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Includes, "include", "i", "Remove an include file with `PRIORITY`\n(Repeat as required, gateways only)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Gateways, "gateway", "g", "Remove the gateway `NAME`\n(Repeat as required, san and floating only)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Attributes, "attribute", "a", "Remove the attribute `NAME`\n(Repeat as required, san only)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Types, "type", "t", "Remove the type `NAME`\n(Repeat as required, san only)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Variables, "variable", "v", "Remove the variable `NAME`\n(Repeat as required, san only)")

	unsetCmd.Flags().SortFlags = false
}

//go:embed _docs/unset.md
var unsetCmdDescription string

var unsetCmd = &cobra.Command{
	Use:     "unset [flags] [TYPE] [NAME...]",
	GroupID: CommandGroupConfig,
	Short:   "Unset Instance Parameters",
	Long:    unsetCmdDescription,
	Example: strings.ReplaceAll(`
geneos unset gateway GW1 -k aesfile
geneos unset san -g Gateway1
`, "|", "`"),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && cmd.Flags().NFlag() == 0 {
			cmd.Usage()
			return
		}
		ct, names := ParseTypeNames(cmd)
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *responses.Response) {
			resp = responses.NewResponse(i)

			cf := i.Config()

			changed := instance.UnsetInstanceValues(i, unsetCmdValues)

			if len(unsetCmdValues.Keys) > 0 {
				for _, k := range unsetCmdValues.Keys {
					if cf.IsSet(k) {
						cf.Set(k, "")
						changed = true
					}
				}
			}

			if !changed && !unsetCmdWarned {
				resp.Err = fmt.Errorf("nothing unset. perhaps you forgot to use -k KEY or one of the other options?")
				unsetCmdWarned = true
				return
			}

			resp.Err = instance.SaveConfig(i)
			return
		}).Report(os.Stdout)
	},
}
