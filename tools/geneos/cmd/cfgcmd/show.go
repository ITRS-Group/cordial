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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/spf13/cobra"
)

var configShowCmdAll bool

func init() {
	configCmd.AddCommand(configShowCmd)

	configShowCmd.Flags().BoolVarP(&configShowCmdAll, "all", "a", false, "Show all the parameters including all defaults")
}

var configShowCmd = &cobra.Command{
	Use:   "show [KEY...]",
	Short: "Show program configuration",
	Long: strings.ReplaceAll(`
The show command outputs the current configuration for the |geneos|
program in JSON format. It shows the processed values from the
on-disk copy of your program configuration and not the final
configuration that the running program uses, which includes many
built-in defaults.

If any arguments are given then they are treated as a list of keys to
limit the output to just those keys that match and have a non-nil value.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var buffer []byte
		var cf *config.Config

		if configShowCmdAll {
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
			if buffer, err = json.MarshalIndent(cf.AllSettings(), "", "    "); err != nil {
				return
			}
		}
		fmt.Println(string(buffer))

		return
	},
}
