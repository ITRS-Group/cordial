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

package cfgcmd

import (
	_ "embed"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/spf13/cobra"
)

func init() {
	configCmd.AddCommand(unsetCmd)
}

//go:embed _docs/unset.md
var unsetCmdDescription string

var unsetCmd = &cobra.Command{
	Use:          "unset [KEY...]",
	Short:        "Unset a program parameter",
	Long:         unsetCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, origargs []string) error {
		var changed bool
		if len(origargs) == 0 && command.Flags().NFlag() == 0 {
			return command.Usage()
		}

		_, args := cmd.ParseTypeNames(command)
		orig, _ := config.Load(cmd.Execname,
			config.IgnoreWorkingDir(),
			config.IgnoreSystemDir(),
		)
		new := config.New()

	OUTER:
		for _, k := range orig.AllKeys() {
			for _, a := range args {
				if k == a {
					changed = true
					continue OUTER
				}
			}
			new.Set(k, orig.Get(k))
		}

		if changed {
			return new.Save(cmd.Execname)
		}
		return nil
	},
}
