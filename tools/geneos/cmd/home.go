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
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(homeCmd)
}

//go:embed _docs/home.md
var homeCmdDescription string

var homeCmd = &cobra.Command{
	Use:     "home [TYPE] [NAME]",
	GroupID: CommandGroupView,
	Short:   "Display Instance and Component Home Directories",
	Long:    homeCmdDescription,
	Example: strings.ReplaceAll(`
cd $(geneos home)
cd $(geneos home gateway example1)
cat $(geneos home gateway example2)/gateway.txt
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:      "false",
		CmdRequireHome: "true",
	},
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, names, _ := ParseTypeNamesParams(cmd)

		if len(names) == 0 {
			if ct == nil {
				fmt.Println(geneos.LocalRoot())
				return nil
			}
			fmt.Println(geneos.LOCAL.PathTo(ct))
			return nil
		}

		h, _, n := instance.Decompose(names[0], geneos.GetHost(Hostname))
		if h == geneos.LOCAL || h == geneos.ALL {
			i, err := instance.Get(ct, n)

			if err != nil {
				fmt.Println(geneos.LocalRoot())
				return nil
			}
			fmt.Println(i.Home())
		} else {
			fmt.Println(geneos.LocalRoot())
		}
		return nil
	},
}
