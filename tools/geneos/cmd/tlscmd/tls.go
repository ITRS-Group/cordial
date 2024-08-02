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

// Package tlscmd contains all the TLS subsystem commands
package tlscmd

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	cmd.GeneosCmd.AddCommand(tlsCmd)
}

//go:embed README.md
var tlsCmdDescription string

var tlsCmd = &cobra.Command{
	Use:          "tls",
	GroupID:      cmd.CommandGroupSubsystems,
	Short:        "TLS Certificate Operations",
	Long:         tlsCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdRequireHome: "true",
	},
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
}
