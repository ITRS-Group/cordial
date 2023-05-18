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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/floating"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
)

func init() {
	InitCmd.AddCommand(initTemplatesCmd)
}

var initTemplatesCmd = &cobra.Command{
	Use:   "template",
	Short: "Initialise or overwrite templates",
	Long: strings.ReplaceAll(`
The |geneos| commands contains a number of default template files
that are normally written out during initialization of a new
installation. In the case of adopting a legacy installation or
upgrading the program it might be desirable to extract these template
files.

This command will overwrite any files with the same name but will not
delete other template files that may already exist.

Use this command if you get missing template errors using the |add|
command.
`, "|", "`"),
	Aliases:      []string{"templates"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.CmdArgsParams(command)
		log.Debug().Msgf("%s %v %v", ct, args, params)
		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error().Err(geneos.ErrInvalidArgs).Msg(ct.String())
			return geneos.ErrInvalidArgs
		}

		return initTemplates(geneos.LOCAL)
	},
}

func initTemplates(h *geneos.Host, options ...geneos.Options) (err error) {
	gatewayTemplates := h.Filepath(gateway.Gateway, "templates")
	h.MkdirAll(gatewayTemplates, 0775)
	tmpl := gateway.GatewayTemplate
	if initCmdGatewayTemplate != "" {
		if tmpl, err = geneos.ReadFrom(initCmdGatewayTemplate); err != nil {
			return
		}
	}
	if err := h.WriteFile(filepath.Join(gatewayTemplates, gateway.GatewayDefaultTemplate), tmpl, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	fmt.Printf("gateway template written to %s\n", filepath.Join(gatewayTemplates, gateway.GatewayDefaultTemplate))

	tmpl = gateway.InstanceTemplate
	if err := h.WriteFile(filepath.Join(gatewayTemplates, gateway.GatewayInstanceTemplate), tmpl, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	fmt.Printf("gateway instance template written to %s\n", filepath.Join(gatewayTemplates, gateway.GatewayInstanceTemplate))

	sanTemplates := h.Filepath(san.San, "templates")
	h.MkdirAll(sanTemplates, 0775)
	tmpl = san.SanTemplate
	if initCmdSANTemplate != "" {
		if tmpl, err = geneos.ReadFrom(initCmdSANTemplate); err != nil {
			return
		}
	}
	if err := h.WriteFile(filepath.Join(sanTemplates, san.SanDefaultTemplate), tmpl, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	fmt.Printf("san template written to %s\n", filepath.Join(sanTemplates, san.SanDefaultTemplate))

	floatingTemplates := h.Filepath(floating.Floating, "templates")
	h.MkdirAll(floatingTemplates, 0775)
	tmpl = floating.FloatingTemplate
	if initCmdFloatingTemplate != "" {
		if tmpl, err = geneos.ReadFrom(initCmdFloatingTemplate); err != nil {
			return
		}
	}
	if err := h.WriteFile(filepath.Join(floatingTemplates, floating.FloatingDefaultTemplate), tmpl, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	fmt.Printf("floating template written to %s\n", filepath.Join(floatingTemplates, floating.FloatingDefaultTemplate))

	return
}
