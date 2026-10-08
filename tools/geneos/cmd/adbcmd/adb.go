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

// package adbcmd implements a subsystem for manipulating exported dashboard (.adb) files
package adbcmd

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	cmd.Cmd.AddCommand(adbCmd)
}

//go:embed README.md
var adbCmdDescription string

var adbCmd = &cobra.Command{
	Use:          "adb",
	GroupID:      cmd.CommandGroupSubsystems,
	Short:        "ADB File Operations",
	Long:         adbCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
}
