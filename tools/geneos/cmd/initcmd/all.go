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
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var allCmdLicenseFile, allCmdArchive string
var allCmdMinimal bool

func init() {
	initCmd.AddCommand(allCmd)

	allCmd.Flags().StringVarP(&allCmdLicenseFile, "licence", "L", "geneos.lic", "Licence file location")
	allCmd.MarkFlagRequired("licence")

	allCmd.Flags().BoolVarP(&allCmdMinimal, "minimal", "M", false, "use a minimal Netprobe release")

	allCmd.Flags().StringVarP(&allCmdArchive, "archive", "A", "", archiveOptionsText)
	allCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", instance.GatewaysOptionstext)

	allCmd.Flags().SortFlags = false
}

//go:embed _docs/all.md
var allCmdDescription string

var allCmd = &cobra.Command{
	Use:   "all [flags] [USERNAME] [DIRECTORY]",
	Short: "Initialise a more complete Geneos environment",
	Long:  allCmdDescription,
	Example: strings.ReplaceAll(`
geneos init all -L https://myserver/files/geneos.lic -u email@example.com
geneos init all -L ~/geneos.lic -A ~/downloads /opt/itrs
sudo geneos init all -L /tmp/geneos-1.lic -u email@example.com myuser /opt/geneos
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.ParseTypeNamesParams(command)
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

		if err = geneos.Initialise(geneos.LOCAL, options...); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if err = initCommon(); err != nil {
			return
		}

		options = append(options, geneos.LocalArchive(allCmdArchive))
		return initAll(geneos.LOCAL, options...)
	},
}

func initAll(h *geneos.Host, options ...geneos.PackageOptions) (err error) {
	e := []string{}

	if initCmdName == "" {
		initCmdName = h.Hostname()
	}

	licdCT := geneos.ParseComponent("licd")
	gatewayCT := geneos.ParseComponent("gateway")
	netprobeCT := geneos.ParseComponent("netprobe")
	minimalCT := geneos.ParseComponent("minimal")
	webserverCT := geneos.ParseComponent("webserver")

	if err = geneos.Install(h, licdCT, options...); err != nil {
		return
	}
	if err = geneos.Install(h, gatewayCT, options...); err != nil {
		return
	}
	if allCmdMinimal {
		if err = geneos.Install(h, minimalCT, options...); err != nil {
			return
		}
	} else {
		if err = geneos.Install(h, netprobeCT, options...); err != nil {
			return
		}
	}
	if err = geneos.Install(h, webserverCT, options...); err != nil {
		return
	}

	if err = cmd.AddInstance(licdCT, initCmdExtras, []string{}, initCmdName); err != nil {
		return
	}
	if err = cmd.ImportFiles(licdCT, []string{initCmdName}, []string{"geneos.lic=" + allCmdLicenseFile}); err != nil {
		return
	}
	if err = cmd.AddInstance(gatewayCT, initCmdExtras, []string{}, initCmdName); err != nil {
		return
	}

	probename := "localhost"
	if allCmdMinimal {
		probename = "minimal:" + probename
	}
	if err = cmd.AddInstance(netprobeCT, initCmdExtras, []string{}, probename+"@"+h.String()); err != nil {
		return
	}
	if err = cmd.AddInstance(webserverCT, initCmdExtras, []string{}, initCmdName); err != nil {
		return
	}
	if err = cmd.Start(nil, initCmdLogs, true, e, e); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	cmd.CommandPS(nil, e, e)
	return
}
