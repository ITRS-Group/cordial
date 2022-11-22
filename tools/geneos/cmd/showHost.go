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

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/spf13/cobra"
)

type showHostCmdConfig struct {
	Name string `json:"name,omitempty"`
	// Disabled  bool        `json:"disabled"`
	// Protected bool        `json:"protected"`
	Config interface{} `json:"config,omitempty"`
}

var showHostCmdRaw bool

func init() {
	showCmd.AddCommand(showHostCmd)

	showHostCmd.Flags().BoolVarP(&showHostCmdRaw, "raw", "r", false, "Show raw (unexpanded) configuration values")

	showHostCmd.Flags().SortFlags = false
}

// showHostCmd represents the showHost command
var showHostCmd = &cobra.Command{
	Use:   "host [flags] [NAME...]",
	Short: "Show details of remote host configuration",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	Aliases:      []string{"hosts"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var hosts []*host.Host

		if len(args) == 0 {
			hosts = host.RemoteHosts()
		} else {
			for _, a := range args {
				h := host.Get(a)
				if h != nil && h.Exists() {
					hosts = append(hosts, h)
				}
			}
		}

		var confs []showHostCmdConfig

		for _, h := range hosts {
			confs = append(confs, showHostCmdConfig{
				Name:   h.GetString("name"),
				Config: h.AllSettings(),
			})
		}

		b, _ := json.MarshalIndent(confs, "", "    ")
		fmt.Println(string(b))
		return
	},
}
