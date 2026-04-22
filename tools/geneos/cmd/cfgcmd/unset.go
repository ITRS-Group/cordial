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
	"strings"

	"github.com/itrs-group/cordial"
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
	Short:        "Unset a global program parameter",
	Long:         unsetCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, origargs []string) (err error) {
		if len(origargs) == 0 && command.Flags().NFlag() == 0 {
			return command.Usage()
		}

		_, args, params, err := cmd.FetchArgs(command)
		if err != nil {
			return err
		}
		args = append(args, params...)

		if len(args) == 0 {
			return command.Usage()
		}

		cf, err := config.Read(cordial.ExecutableName(),
			config.SkipWorkingDir(),
			config.SkipSystemDir(),
		)
		if err != nil {
			return err
		}

		for _, a := range args {
			b, _, _ := strings.Cut(a, "=")
			config.Delete(cf, b)
		}

		return cf.Write(cordial.ExecutableName())
	},
}
