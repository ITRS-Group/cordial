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

package cmd

import (
	_ "embed"
	"fmt"
	dbg "runtime/debug"

	"github.com/itrs-group/cordial"
	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(versionCmd)
}

//go:embed _docs/version.md
var versionCmdDescription string

var versionCmd = &cobra.Command{
	Use:          "version",
	GroupID:      CommandGroupOther,
	Short:        "Show program version",
	Long:         versionCmdDescription,
	SilenceUsage: true,
	Version:      cordial.VERSION,
	Annotations: map[string]string{
		CmdGlobal:      "false",
		CmdRequireHome: "false",
	},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s version %s\n", cordial.ExecutableName(), cmd.Version)
		if debug {
			info, ok := dbg.ReadBuildInfo()
			if ok {
				fmt.Println("additional info:")
				fmt.Println(info)
			}
		}
	},
}
