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
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var buffer []byte
		var cf *config.Config

		if showCmdAll {
			cf = config.GetConfig()
		} else {
			cf, _ = config.Load(cmd.Execname, config.IgnoreSystemDir(), config.IgnoreWorkingDir())
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
			if buffer, err = json.MarshalIndent(cf.ExpandAllSettings(config.NoDecode()), "", "    "); err != nil {
				return
			}
		}
		fmt.Println(string(buffer))

		return
	},
}
