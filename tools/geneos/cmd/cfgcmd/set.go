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
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	ConfigCmd.AddCommand(setUserCmd)

	// setUserCmd.Flags().SortFlags = false
}

var setUserCmd = &cobra.Command{
	Use:   "set [KEY=VALUE...]",
	Short: "Set configuration parameters",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		_, _, params := cmd.CmdArgsParams(command)

		vp, _ := config.Load(cmd.Execname, config.IgnoreSystemDir(), config.IgnoreWorkingDir())
		vp.SetKeyValues(params...)

		// fix breaking change
		if vp.IsSet("itrshome") {
			if !vp.IsSet("geneos") {
				vp.Set("geneos", vp.GetString("itrshome"))
			}
			vp.Set("itrshome", nil)
		}

		return vp.Save(cmd.Execname)
	},
}
