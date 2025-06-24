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
	"os"
	dbg "runtime/debug"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/spf13/cobra"
)

var versionCmdToolkit bool

func init() {
	GeneosCmd.AddCommand(versionCmd)

	versionCmd.Flags().BoolVarP(&versionCmdToolkit, "toolkit", "t", false, "toolkit formatted CSV output")

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
		if versionCmdToolkit {
			fmt.Printf("Name,Value\n")
			fmt.Printf("Version,%s\n", cmd.Version)
			fmt.Printf("Source,https://github.com/ITRS-Group/cordial\n")
			if execpath, err := os.Executable(); err == nil {
				fmt.Printf("Executable,%s\n", execpath)
			}
			fmt.Printf("Config Path,%s\n", configPath)
			fmt.Printf("Geneos Root,%s\n", geneos.LocalRoot())

			return
		}
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
