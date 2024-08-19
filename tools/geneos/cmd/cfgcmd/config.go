/*
Copyright © 2023 ITRS Group

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

// Package cfgcmd groups config commands in their own package
package cfgcmd

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	cmd.GeneosCmd.AddCommand(configCmd)
}

//go:embed README.md
var configCmdDescription string

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:          "config",
	GroupID:      cmd.CommandGroupSubsystems,
	Short:        "Configure Command Behaviour",
	Long:         configCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
}
