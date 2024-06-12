/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "true",
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

func initTemplates(h *geneos.Host, options ...geneos.PackageOptions) (err error) {
	for _, ct := range geneos.RealComponents() {
		if len(ct.Templates) == 0 {
			continue
		}
		templateDir := h.PathTo(ct, "templates")
		h.MkdirAll(templateDir, 0775)

		for _, t := range ct.Templates {
			tmpl := t.Content
			if initCmdGatewayTemplate != "" {
				if tmpl, err = geneos.ReadFrom(initCmdGatewayTemplate); err != nil {
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
