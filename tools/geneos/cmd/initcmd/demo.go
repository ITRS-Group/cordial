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
	"strings"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var initDemoCmdArchive string

func init() {
	initCmd.AddCommand(initDemoCmd)

	initDemoCmd.Flags().StringVarP(&initDemoCmdArchive, "archive", "A", "", ArchiveOptionsText)
	initDemoCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", instance.GatewayValuesOptionstext)
	initDemoCmd.Flags().SortFlags = false
}

var initDemoCmd = &cobra.Command{
	Use:   "demo [flags] [USERNAME] [DIRECTORY]",
	Short: "Initialise a Geneos Demo environment",
	Long: strings.ReplaceAll(`
Initialise a Geneos Demo environment, creating a new directory
structure as required.

Without any flags the command installs the components in a directory
called |geneos| under the user's home directory (unless the user's
home directory ends in |geneos| in which case it uses that directly),
downloads the latest release archives and creates a Gateway instance
using the name |Demo Gateway| (with embedded space) as required for
Demo licensing, as Netprobe and a Webserver.

If the release archive files required have already been downloaded then
use the |-A directory| flag to indicate their location. For each
component type this directory is checked for the latest release.

Otherwise, to fetch the releases from the ITRS download server
authentication will be required use the |-u email@example.com| to
specify the user account and you will be prompted for a password.

The initial configuration file for the Gateway is built from the
default templates installed and located in |.../templates| but this
can be overridden with the |-s| option. For the Gateway you can add
include files using |-i PRIORITY:PATH| flag. This can be repeated
multiple times.

Other flags inherited from the |geneos init| command can be used to
influence the installation.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.CmdArgsParams(command)
		log.Debug().Msgf("%s %v %v", ct, args, params)
		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error().Err(geneos.ErrInvalidArgs).Msg(ct.String())
			return geneos.ErrInvalidArgs
		}
		options, err := initProcessArgs(args)
		if err != nil {
			return
		}

		if err = geneos.Init(geneos.LOCAL, options...); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if err = initMisc(command); err != nil {
			return
		}

		options = append(options, geneos.Source(initDemoCmdArchive))
		return initDemo(geneos.LOCAL, options...)
	},
}

func initDemo(h *geneos.Host, options ...geneos.Options) (err error) {
	e := []string{}
	g := []string{"Demo Gateway@" + h.String()}

	if err = install("gateway", geneos.LOCALHOST, options...); err != nil {
		return
	}
	if err = install("netprobe", geneos.LOCALHOST, options...); err != nil {
		return
	}
	if err = install("webserver", geneos.LOCALHOST, options...); err != nil {
		return
	}

	if err = cmd.AddInstance(geneos.FindComponent("gateway"), initCmdExtras, []string{}, "Demo Gateway@"+h.String()); err != nil {
		return
	}
	if err = cmd.Set(geneos.FindComponent("gateway"), g, []string{"options=-demo"}); err != nil {
		return
	}
	// if len(initCmdExtras.Gateways) == 0 {
	// 	initCmdExtras.Gateways.Set("localhost")
	// }
	if err = cmd.AddInstance(geneos.FindComponent("netprobe"), initCmdExtras, []string{}, "localhost@"+h.String()); err != nil {
		return
	}
	if err = cmd.AddInstance(geneos.FindComponent("webserver"), initCmdExtras, []string{}, "demo@"+h.String()); err != nil {
		return
	}

	if err = cmd.Start(nil, initCmdLogs, e, e); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	return cmd.CommandPS(nil, e, e)
}
