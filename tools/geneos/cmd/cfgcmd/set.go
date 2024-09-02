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

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	configCmd.AddCommand(setCmd)

	// setCmd.Flags().VarP()
}

//go:embed _docs/set.md
var setCmdDescription string

var setCmd = &cobra.Command{
	Use:   "set [KEY=VALUE...]",
	Short: "Set program configuration",
	Long:  setCmdDescription,
	Example: `
geneos config set geneos="/opt/geneos"
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, origargs []string) (err error) {
		_, _, params := cmd.ParseTypeNamesParams(command)
		if len(origargs) == 0 && command.Flags().NFlag() == 0 {
			return command.Usage()
		}
		cf, err := config.Load(cmd.Execname,
			config.IgnoreSystemDir(),
			config.IgnoreWorkingDir(),
		)
		if err != nil {
			return
		}
		log.Debug().Msgf("setting params: %v", params)
		cf.SetKeyValues(params...)

		// fix breaking change
		if cf.IsSet("itrshome") {
			if !cf.IsSet("geneos") {
				cf.Set("geneos", cf.GetString("itrshome"))
			}
			cf.Set("itrshome", nil)
		}

		log.Debug().Msgf("save config %q", cmd.Execname)
		return cf.Save(cmd.Execname)
	},
}
