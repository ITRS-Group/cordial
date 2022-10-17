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

package cmd

import (
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/webserver"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var initDemoCmdArchive string

func init() {
	initCmd.AddCommand(initDemoCmd)

	initDemoCmd.Flags().StringVarP(&initDemoCmdArchive, "archive", "A", "", "`PATH or URL` to software archive to install")

	initDemoCmd.Flags().VarP(&initCmdExtras.Envs, "env", "e", "Add environment variables in the format NAME=VALUE. Repeat flag for more values.")
	initDemoCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", "(gateways) Add an include file in the format PRIORITY:PATH")

	initDemoCmd.Flags().SortFlags = false
}

var initDemoCmd = &cobra.Command{
	Use:   "demo [flags] [USERNAME] [DIRECTORY]",
	Short: "Initialise a Geneos Demo environment",
	Long: strings.ReplaceAll(`
Install a Demo environment into a new Geneos install directory
layout.

Without any flags the command installs the components in a directory
called |geneos| under the user's home directory (unless the user's
home directory ends in |geneos| in which case it uses that directly),
downloads the latest release archives and creates a Gateway instance
using the name |Demo| as required for Demo licensing, as Netprobe and
a Webserver.

In almost all cases authentication will be required to download the
install packages and as this is a new Geneos installation it is
unlikely that the download credentials are saved in a local config
file, so use the |-u email@example.com| as appropriate.

The initial configuration file for the Gateway is built from the
default templates installed and located in |.../templates| but this
can be overridden with the |-s| option. For the Gateway you can add
include files using |-i PRIORITY:PATH| flag. This can be repeated
multiple times.

The |-e| flag adds environment variables to all instances created and
so should only be used for common values, such as |TZ|.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, args, params := cmdArgsParams(cmd)
		log.Debug().Msgf("%s %v %v", ct, args, params)
		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error().Err(ErrInvalidArgs).Msg(ct.String())
			return ErrInvalidArgs
		}
		options, err := initProcessArgs(args)
		if err != nil {
			return
		}

		if err = geneos.Init(host.LOCAL, options...); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if err = initMisc(); err != nil {
			return
		}

		options = append(options, geneos.Source(initDemoCmdArchive))
		return initDemo(host.LOCAL, options...)
	},
}

func initDemo(h *host.Host, options ...geneos.GeneosOptions) (err error) {
	e := []string{}
	g := []string{"Demo Gateway@" + h.String()}

	install(&gateway.Gateway, host.LOCALHOST, options...)
	install(&netprobe.Netprobe, host.LOCALHOST, options...)
	install(&webserver.Webserver, host.LOCALHOST, options...)

	addInstance(&gateway.Gateway, initCmdExtras, "Demo Gateway@"+h.String())
	set(&gateway.Gateway, g, []string{"options=-demo"})
	if len(initCmdExtras.Gateways) == 0 {
		initCmdExtras.Gateways.Set("localhost")
	}
	addInstance(&netprobe.Netprobe, initCmdExtras, "localhost@"+h.String())
	addInstance(&webserver.Webserver, initCmdExtras, "demo@"+h.String())

	start(nil, initCmdLogs, e, e)
	commandPS(nil, e, e)
	return
}
