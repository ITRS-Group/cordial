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
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var initAllCmdLicenseFile, initAllCmdArchive string

func init() {
	initCmd.AddCommand(initAllCmd)

	initAllCmd.Flags().StringVarP(&initAllCmdLicenseFile, "licence", "L", "geneos.lic", "Licence file location")
	initAllCmd.MarkFlagRequired("licence")

	initAllCmd.Flags().StringVarP(&initAllCmdArchive, "archive", "A", "", archiveOptionsText)
	initAllCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", instance.GatewayValuesOptionstext)

	initAllCmd.Flags().SortFlags = false
}

//go:embed _docs/all.md
var initAllCmdDescription string

var initAllCmd = &cobra.Command{
	Use:   "all [flags] [USERNAME] [DIRECTORY]",
	Short: "Initialise a more complete Geneos environment",
	Long:  initAllCmdDescription,
	Example: strings.ReplaceAll(`
geneos init all -L https://myserver/files/geneos.lic -u email@example.com
geneos init all -L ~/geneos.lic -A ~/downloads /opt/itrs
sudo geneos init all -L /tmp/geneos-1.lic -u email@example.com myuser /opt/geneos
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

		options = append(options, geneos.Source(initAllCmdArchive))
		return initAll(geneos.LOCAL, options...)
	},
}

func initAll(h *geneos.Host, options ...geneos.Options) (err error) {
	e := []string{}

	if initCmdName == "" {
		initCmdName, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	if err = install("licd", h.String(), options...); err != nil {
		return
	}
	if err = install("gateway", h.String(), options...); err != nil {
		return
	}
	if err = install("netprobe", h.String(), options...); err != nil {
		return
	}
	if err = install("webserver", h.String(), options...); err != nil {
		return
	}

	if err = cmd.AddInstance(geneos.FindComponent("licd"), initCmdExtras, []string{}, initCmdName); err != nil {
		return
	}
	if err = cmd.ImportFiles(geneos.FindComponent("licd"), []string{initCmdName}, []string{"geneos.lic=" + initAllCmdLicenseFile}); err != nil {
		return
	}
	if err = cmd.AddInstance(geneos.FindComponent("gateway"), initCmdExtras, []string{}, initCmdName); err != nil {
		return
	}
	// if len(initCmdExtras.Gateways) == 0 {
	// 	initCmdExtras.Gateways.Set("localhost")
	// }
	if err = cmd.AddInstance(geneos.FindComponent("netprobe"), initCmdExtras, []string{}, "localhost@"+h.String()); err != nil {
		return
	}
	if err = cmd.AddInstance(geneos.FindComponent("webserver"), initCmdExtras, []string{}, initCmdName); err != nil {
		return
	}
	if err = cmd.Start(nil, initCmdLogs, e, e); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	return cmd.CommandPS(nil, e, e)
}
