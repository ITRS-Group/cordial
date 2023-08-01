/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "false",
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
