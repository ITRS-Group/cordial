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
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
)

var initSanCmdArchive, initSanCmdVersion, initSanCmdOverride string

func init() {
	initCmd.AddCommand(initSanCmd)

	initSanCmd.Flags().StringVarP(&initSanCmdVersion, "version", "V", "latest", "Download this `VERSION`, defaults to latest. Doesn't work for EL8 archives.")
	initSanCmd.Flags().StringVarP(&initSanCmdArchive, "archive", "A", "", "`PATH or URL` to software archive to install")
	initSanCmd.Flags().StringVarP(&initSanCmdOverride, "override", "T", "", "Override the `[TYPE:]VERSION` for archive files with non-standard names")
	initSanCmd.Flags().VarP(&initCmdExtras.Gateways, "gateway", "g", "Add gateway in the format NAME:PORT. Repeat flag for more gateways.")
	initSanCmd.Flags().VarP(&initCmdExtras.Attributes, "attribute", "a", "Add an attribute in the format NAME=VALUE. Repeat flag for more attributes.")
	initSanCmd.Flags().VarP(&initCmdExtras.Types, "type", "t", "Add a type NAME. Repeat flag for more types.")
	initSanCmd.Flags().VarP(&initCmdExtras.Variables, "variable", "v", "Add a variable in the format [TYPE:]NAME=VALUE. Repeat flag for more variables")

	initSanCmd.Flags().SortFlags = false
}

var initSanCmd = &cobra.Command{
	Use:   "san [flags] [USERNAME] [DIRECTORY]",
	Short: "Initialise a Geneos SAN (Self-Announcing Netprobe) environment",
	Long: strings.ReplaceAll(`
Install a Self-Announcing Netprobe (SAN) into a new Geneos install
directory.

Without any flags the command installs a SAN in a directory called
|geneos| under the user's home directory (unless the user's home
directory ends in |geneos| in which case it uses that directly),
downloads the latest netprobe release and create a SAN instance using
the |hostname| of the system.

In almost all cases authentication will be required to download the
Netprobe package and as this is a new Geneos installation it is
unlikely that the download credentials are saved in a local config
file, so use the |-u email@example.com| as appropriate.

If you have a netprobe software archive locally then use the |-A
PATH|. If the name of the file is not in the same format as
downloaded from the official site(s) then you have to also set the
type (netprobe) and version using the |-T [TYPE:]VERSION|. TYPE is
set to |netprobe| if not given. 

The initial configuration file is built from the default templates
installed and located in |.../templates| but this can be overridden
with the |-s| option. You can set |gateways|, |types|, |attributes|,
|variables| using the appropriate flags. These flags can be specified
multiple times.
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
			log.Error().Err(cmd.ErrInvalidArgs).Msg(ct.String())
			return cmd.ErrInvalidArgs
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

		// prefix with netprobe
		if initSanCmdOverride != "" && !strings.Contains(initSanCmdOverride, ":") {
			initSanCmdOverride = "netprobe:" + initSanCmdOverride
		}

		options = append(options,
			geneos.Source(initSanCmdArchive),
			geneos.Version(initSanCmdVersion),
			geneos.OverrideVersion(initSanCmdOverride),
		)
		return initSan(geneos.LOCAL, options...)
	},
}

func initSan(h *geneos.Host, options ...geneos.Options) (err error) {
	var sanname string

	e := []string{}

	if initCmdName != "" {
		sanname = initCmdName
	} else {
		sanname, _ = os.Hostname()
	}
	if h != geneos.LOCAL {
		sanname = sanname + "@" + geneos.LOCALHOST
	}
	install(&san.San, geneos.LOCALHOST, options...)
	cmd.AddInstance(&san.San, initCmdExtras, sanname)
	cmd.Start(nil, initCmdLogs, e, e)
	cmd.CommandPS(nil, e, e)
	return nil
}
