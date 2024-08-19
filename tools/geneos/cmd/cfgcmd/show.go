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

package cfgcmd

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/spf13/cobra"
)

var showCmdAll bool

func init() {
	configCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVarP(&showCmdAll, "all", "a", false, "Show all the parameters including all defaults")
}

//go:embed _docs/show.md
var showCmdDescription string

var showCmd = &cobra.Command{
	Use:          "show [KEY...]",
	Short:        "Show program configuration",
	Long:         showCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var buffer []byte
		var cf *config.Config

		if showCmdAll {
			cf = config.GetConfig()
		} else {
			cf, _ = config.Load(cmd.Execname,
				config.IgnoreSystemDir(),
				config.IgnoreWorkingDir())
		}

		if len(args) > 0 {
			values := make(map[string]interface{})
			for _, k := range args {
				v := cf.Get(k)
				if v != nil {
					values[k] = v
				}
			}
			if buffer, err = json.MarshalIndent(values, "", "    "); err != nil {
				return
			}
		} else {
			if buffer, err = json.MarshalIndent(cf.ExpandAllSettings(config.NoDecode(true)), "", "    "); err != nil {
				return
			}
		}
		fmt.Println(string(buffer))

		return
	},
}
