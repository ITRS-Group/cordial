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

var sanCmdArchive, sanCmdVersion, sanCmdOverride string

func init() {
	initCmd.AddCommand(sanCmd)

	sanCmd.Flags().StringVarP(&sanCmdVersion, "version", "V", "latest", "Download this `VERSION`, defaults to latest. Doesn't work for EL8 archives.")
	sanCmd.Flags().StringVarP(&sanCmdArchive, "archive", "A", "", archiveOptionsText)
	sanCmd.Flags().StringVarP(&sanCmdOverride, "override", "O", "", "Override the `[TYPE:]VERSION` for archive files with non-standard names")

	sanCmd.Flags().VarP(&initCmdExtras.Gateways, "gateway", "g", instance.GatewaysOptionstext)
	sanCmd.Flags().VarP(&initCmdExtras.Attributes, "attribute", "a", instance.AttributesOptionsText)
	sanCmd.Flags().VarP(&initCmdExtras.Types, "type", "t", instance.TypesOptionsText)
	sanCmd.Flags().VarP(&initCmdExtras.Variables, "variable", "v", instance.VarsOptionsText)

	sanCmd.Flags().SortFlags = false
}

//go:embed _docs/san.md
var sanCmdDescription string

var sanCmd = &cobra.Command{
	Use:          "san [flags] [USERNAME] [DIRECTORY]",
	Short:        "Initialise a Geneos SAN (Self-Announcing Netprobe) environment",
	Long:         sanCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "false",
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

		if err = initCommon(command); err != nil {
			return
		}

		// prefix with netprobe
		if sanCmdOverride != "" && !strings.Contains(sanCmdOverride, ":") {
			sanCmdOverride = "netprobe:" + sanCmdOverride
		}

		options = append(options,
			geneos.LocalArchive(sanCmdArchive),
			geneos.Version(sanCmdVersion),
			geneos.OverrideVersion(sanCmdOverride),
		)
		return initSan(geneos.LOCAL, options...)
	},
}

func initSan(h *geneos.Host, options ...geneos.PackageOptions) (err error) {
	var sanname string

	e := []string{}

	if initCmdName != "" {
		sanname = initCmdName
	} else {
		sanname = h.Hostname()
	}
	if !h.IsLocal() {
		sanname = sanname + "@" + geneos.LOCALHOST
	}
	if err = install("san", geneos.LOCALHOST, options...); err != nil {
		return
	}
	if err = cmd.AddInstance(geneos.ParseComponent("san"), initCmdExtras, []string{}, sanname); err != nil {
		return
	}
	if err = cmd.Start(nil, initCmdLogs, true, e, e); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	cmd.CommandPS(nil, e, e)
	return
}
