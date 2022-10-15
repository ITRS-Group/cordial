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
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/webserver"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var initAllCmdLicenseFile string

func init() {
	initCmd.AddCommand(initAllCmd)

	initAllCmd.Flags().StringVarP(&initAllCmdLicenseFile, "licence", "L", "geneos.lic", "Path (or URL) to license file")
	initAllCmd.Flags().SetNormalizeFunc(initAllCmdNormalizeFunc)

	initAllCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", "(gateways) Add an include file in the format PRIORITY:PATH")

	initAllCmd.Flags().SortFlags = false
}

// catch misspelling of the license/licence flag
func initAllCmdNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	switch name {
	case "license":
		name = "licence"
	}
	return pflag.NormalizedName(name)
}

var initAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Initialise a more complete Geneos environment",
	Long: strings.ReplaceAll(`
Initialise a Geneos installation and download and install common components.
`, "|", "`"),
	Example: strings.ReplaceAll(`
geneos init all /opt/itrs
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
	name := []string{initCmdName}
	localhost := []string{"localhost@" + host.LOCALHOST}

	install(&licd.Licd, h.String(), options...)
	install(&gateway.Gateway, h.String(), options...)
	install(&san.San, h.String(), options...)
	install(&webserver.Webserver, h.String(), options...)

	addInstance(&licd.Licd, initCmdExtras, name)
	importFiles(&licd.Licd, name, []string{"geneos.lic=" + initAllCmdLicenseFile})
	addInstance(&gateway.Gateway, initCmdExtras, name)
	if len(initCmdExtras.Gateways) == 0 {
		initCmdExtras.Gateways.Set("localhost")
	}
	addInstance(&san.San, initCmdExtras, localhost)
	addInstance(&webserver.Webserver, initCmdExtras, name)
	start(nil, initCmdLogs, e, e)
	commandPS(nil, e, e)
	return nil
}
