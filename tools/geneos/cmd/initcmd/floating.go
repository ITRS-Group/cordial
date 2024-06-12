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

var floatingCmdArchive, floatingCmdVersion, floatingCmdOverride string

func init() {
	initCmd.AddCommand(floatingCmd)

	floatingCmd.Flags().StringVarP(&floatingCmdVersion, "version", "V", "latest", "Download this `VERSION`, defaults to latest. Doesn't work for EL8 archives.")
	floatingCmd.Flags().StringVarP(&floatingCmdArchive, "archive", "A", "", archiveOptionsText)
	floatingCmd.Flags().StringVarP(&floatingCmdOverride, "override", "O", "", "Override the `[TYPE:]VERSION` for archive files with non-standard names")

	floatingCmd.Flags().VarP(&initCmdExtras.Gateways, "gateway", "g", instance.GatewaysOptionstext)
	floatingCmd.Flags().VarP(&initCmdExtras.Attributes, "attribute", "a", instance.AttributesOptionsText)
	floatingCmd.Flags().VarP(&initCmdExtras.Types, "type", "t", instance.TypesOptionsText)
	floatingCmd.Flags().VarP(&initCmdExtras.Variables, "variable", "v", instance.VarsOptionsText)

	floatingCmd.Flags().SortFlags = false
}

//go:embed _docs/floating.md
var floatingCmdDescription string

var floatingCmd = &cobra.Command{
	Use:          "floating [flags] [USERNAME] [DIRECTORY]",
	Short:        "Initialise a Geneos Floating Netprobe environment",
	Long:         floatingCmdDescription,
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
		if floatingCmdOverride != "" && !strings.Contains(floatingCmdOverride, ":") {
			floatingCmdOverride = "netprobe:" + floatingCmdOverride
		}

		options = append(options,
			geneos.LocalArchive(floatingCmdArchive),
			geneos.Version(floatingCmdVersion),
			geneos.OverrideVersion(floatingCmdOverride),
		)
		return initFloating(geneos.LOCAL, options...)
	},
}

func initFloating(h *geneos.Host, options ...geneos.PackageOptions) (err error) {
	var floatingname string

	e := []string{}

	if initCmdName != "" {
		floatingname = initCmdName
	} else {
		floatingname = h.Hostname()
	}
	if !h.IsLocal() {
		floatingname = floatingname + "@" + geneos.LOCALHOST
	}
	if err = install("floating", geneos.LOCALHOST, options...); err != nil {
		return
	}

	if err = cmd.AddInstance(geneos.ParseComponent("floating"), initCmdExtras, []string{}, floatingname); err != nil {
		return
	}
	if err = cmd.Start(nil, initCmdLogs, true, e, e); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	cmd.CommandPS(nil, e, e)
	return
}
