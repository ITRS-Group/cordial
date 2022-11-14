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
	"os"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/webserver"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var initAllCmdLicenseFile, initAllCmdArchive string

func init() {
	initCmd.AddCommand(initAllCmd)

	initAllCmd.Flags().StringVarP(&initAllCmdLicenseFile, "licence", "L", "geneos.lic", "`Filepath or URL` to license file")
	initAllCmd.MarkFlagRequired("licence")

	initAllCmd.Flags().StringVarP(&initAllCmdArchive, "archive", "A", "", "`PATH or URL` to software archive to install")

	initAllCmd.Flags().VarP(&initCmdExtras.Envs, "env", "e", "Add environment variables in the format NAME=VALUE. Repeat flag for more values.")
	initAllCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", "(gateways) Add an include file in the format PRIORITY:PATH")

	initAllCmd.Flags().SortFlags = false
}

var initAllCmd = &cobra.Command{
	Use:   "all [flags] [USERNAME] [DIRECTORY]",
	Short: "Initialise a more complete Geneos environment",
	Long: strings.ReplaceAll(`
Initialise a typical Geneos installation.

This command initialises a Geneos installation by:
- Creating the directory structure & user configuration file,
- Installing software ackages for component types |gateway|, |licd|,
  |netprobe| & |webserver|,
- Creating an instance for each component type named after the hostname
  (except for |netprobe| whose instance is named |localhost|)
- Starting the created instances.

A license file is required and should be given using option |-L|.
If a license file is not available, then use |-L /dev/null| which will
create an empty |geneos.lc| file that can be overwritten later.

Authentication will most-likely be required to download the installation
software packages and, as this is a new Geneos installation, it is unlikely
that the download credentials are saved in a local config file.
Use option |-u email@example.com| to define the username for downloading
software packages.

If packages are already downloaded locally, use option |-A Path_To_Archive|
to refer to the directory containing the package archives.  Package files
must be named in the same format as those downloaded from the 
[ITRS download portal](https://resources.itrsgroup.com/downloads).
If no version is given using option |-V|, then the latest version of each
component is installed.
`, "|", "`"),
	Example: strings.ReplaceAll(`
geneos init all -L https://myserver/files/geneos.lic -u email@example.com
geneos init all -L ~/geneos.lic -A ~/downloads /opt/itrs
sudo geneos init all -L /tmp/geneos-1.lic -u email@example.com myuser /opt/geneos
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

		options = append(options, geneos.Source(initAllCmdArchive))
		return initAll(host.LOCAL, options...)
	},
}

func initAll(h *host.Host, options ...geneos.GeneosOptions) (err error) {
	e := []string{}

	if initCmdName == "" {
		initCmdName, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	install(&licd.Licd, h.String(), options...)
	install(&gateway.Gateway, h.String(), options...)
	install(&netprobe.Netprobe, h.String(), options...)
	install(&webserver.Webserver, h.String(), options...)

	addInstance(&licd.Licd, initCmdExtras, initCmdName)
	importFiles(&licd.Licd, []string{initCmdName}, []string{"geneos.lic=" + initAllCmdLicenseFile})
	addInstance(&gateway.Gateway, initCmdExtras, initCmdName)
	// if len(initCmdExtras.Gateways) == 0 {
	// 	initCmdExtras.Gateways.Set("localhost")
	// }
	addInstance(&netprobe.Netprobe, initCmdExtras, "localhost@"+h.String())
	addInstance(&webserver.Webserver, initCmdExtras, initCmdName)
	start(nil, initCmdLogs, e, e)
	commandPS(nil, e, e)
	return nil
}
