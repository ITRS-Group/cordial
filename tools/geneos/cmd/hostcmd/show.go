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

package hostcmd

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/spf13/cobra"
)

type showCmdConfig struct {
	Name   string      `json:"name,omitempty"`
	Hidden bool        `json:"hidden,omitempty"`
	Config interface{} `json:"config,omitempty"`
}

func init() {
	hostCmd.AddCommand(showCmd)

	showCmd.Flags().SortFlags = false
}

//go:embed _docs/show.md
var showCmdDescription string

var showCmd = &cobra.Command{
	Use:          "show [flags] [NAME...]",
	Short:        "Show details of remote host configuration",
	Long:         showCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var hosts []*geneos.Host

		if len(args) == 0 {
			hosts = geneos.RemoteHosts(false)
		} else {
			for _, a := range args {
				h := geneos.GetHost(a)
				if h.Exists() {
					hosts = append(hosts, h)
				}
			}
		}

		var confs []showCmdConfig

		for _, h := range hosts {
			confs = append(confs, showCmdConfig{
				Name:   h.GetString("name"),
				Hidden: h.Hidden(),
				Config: h.AllSettings(),
			})
		}

		b, _ := json.MarshalIndent(confs, "", "    ")
		fmt.Println(string(b))
		return
	},
}
