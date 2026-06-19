/*
Copyright © 2026 ITRS Group

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

package imscmd

import (
	_ "embed"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

//go:embed README.md
var incidentCmdDescription string

func init() {
	cmd.Cmd.AddCommand(incidentCmd)
}

var log = cordial.Logger

var incidentCmd = &cobra.Command{
	Use:          "incident",
	Short:        "Commands for working with incidents",
	Long:         incidentCmdDescription,
	GroupID:      cmd.CommandGroupSubsystems,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// imsLoadConfigFile reads in the IMS specific client config file.
//
// This configuration file is different to the global `geneos` config,
// and is specific to Incident Management Subsystem. It is typically
// named `${HOME}/.config/geneos/ims.yaml` and contain the relevant
// configuration for this subsystem, such as gateway types, URLs and
// profiles.
func imsLoadConfigFile(name string, configFile string) (cf *config.Config) {
	var err error

	if name == "" {
		name = "ims"
	}

	cf, err = config.Read(name,
		config.AppName(cordial.ExecutableName()),
		config.UseGlobal(),
		config.Format("yaml"),
		config.FilePath(configFile),
		config.MustExist(),
	)
	if err != nil {
		log.Error("failed to load a configuration file from any expected location", slog.Any("error", err))
	}
	log.Debug("loaded config file",
		slog.String("path",
			config.Path(name,
				config.AppName(cordial.ExecutableName()),
				config.UseGlobal(),
				config.Format("yaml"),
				config.FilePath(configFile),
			),
		),
	)

	return
}
