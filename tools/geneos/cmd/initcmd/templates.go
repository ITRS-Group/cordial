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
	"fmt"
	"path"

	"github.com/rs/zerolog/log"
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
	Use:          "template",
	Short:        "Initialise or overwrite templates",
	Long:         templatesCmdDescription,
	Aliases:      []string{"templates"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.ParseTypeNamesParams(command)
		log.Debug().Msgf("%s %v %v", ct, args, params)
		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error().Err(geneos.ErrInvalidArgs).Msg(ct.String())
			return geneos.ErrInvalidArgs
		}

		return initTemplates(geneos.LOCAL)
	},
}

func initTemplates(h *geneos.Host) (err error) {
	for _, ct := range geneos.RealComponents() {
		if len(ct.Templates) == 0 {
			continue
		}
		templateDir := h.PathTo(ct, "templates")
		h.MkdirAll(templateDir, 0775)

		for _, t := range ct.Templates {
			tmpl := t.Content
			if initCmdGatewayTemplate != "" {
				if tmpl, err = geneos.ReadAll(initCmdGatewayTemplate); err != nil {
					return
				}
			}

			if err = h.WriteFile(path.Join(templateDir, t.Filename), tmpl, 0664); err != nil {
				return
			}
			fmt.Printf("%s template %q written to %s\n", ct, t.Filename, templateDir)
		}
	}

	return
}
