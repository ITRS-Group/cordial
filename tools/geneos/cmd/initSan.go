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
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	initCmd.AddCommand(initSanCmd)

	initSanCmd.Flags().VarP(&initCmdExtras.Envs, "env", "e", "Add an environment variable in the format NAME=VALUE. Repeat flag for more variables.")
	initSanCmd.Flags().VarP(&initCmdExtras.Gateways, "gateway", "g", "Add gateway in the format NAME:PORT. Repeat flag for more gateways.")
	initSanCmd.Flags().VarP(&initCmdExtras.Attributes, "attribute", "a", "Add an attribute in the format NAME=VALUE. Repeat flag for more attributes.")
	initSanCmd.Flags().VarP(&initCmdExtras.Types, "type", "t", "Add a type NAME. Repeat flag for more types.")
	initSanCmd.Flags().VarP(&initCmdExtras.Variables, "variable", "v", "Add a variable in the format [TYPE:]NAME=VALUE. Repeat flag for more variables")

	initSanCmd.Flags().SortFlags = false
}

var initSanCmd = &cobra.Command{
	Use:   "san",
	Short: "Initialise a Geneos SAN (Self-Announcing Netprobe) environment",
	Long: strings.ReplaceAll(`
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
		options, err := initProcessArgs(args, params)
		if err != nil {
			return
		}

		if err = geneos.Init(host.LOCAL, options...); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if err = initMisc(); err != nil {
			return
		}

		return initSan(host.LOCAL, options...)
	},
}

func initSan(h *host.Host, options ...geneos.GeneosOptions) (err error) {
	var sanname string
	var s []string

	e := []string{}

	if initCmdName != "" {
		sanname = initCmdName
	} else {
		sanname, _ = os.Hostname()
	}
	if h != host.LOCAL {
		sanname = sanname + "@" + host.LOCALHOST
	}
	s = []string{sanname}
	install(&san.San, host.LOCALHOST, options...)
	addInstance(&san.San, initCmdExtras, s)
	start(nil, initCmdLogs, e, e)
	commandPS(nil, e, e)
	return nil
}
