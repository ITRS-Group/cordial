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

package initcmd

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func init() {
	initCmd.AddCommand(templatesCmd)
}

//go:embed _docs/templates.md
var templatesCmdDescription string

var templatesCmd = &cobra.Command{
	Use:          "templates [TYPE]",
	Short:        "Initialise or overwrite templates",
	Long:         templatesCmdDescription,
	Aliases:      []string{"template"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		// log := log.With("command", "init templates")
		ct, _, _, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}

		return geneos.WriteTemplates(geneos.LOCAL, ct)
	},
}
