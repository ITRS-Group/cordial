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

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	configCmd.AddCommand(setCmd)

	// setCmd.Flags().VarP()
}

//go:embed _docs/set.md
var setCmdDescription string

var setCmd = &cobra.Command{
	Use:   "set [KEY=VALUE...]",
	Short: "Set program configuration",
	Long:  setCmdDescription,
	Example: `
geneos config set geneos="/opt/geneos"
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, origargs []string) (err error) {
		_, _, params := cmd.CmdArgsParams(command)
		if len(origargs) == 0 && command.Flags().NFlag() == 0 {
			return command.Usage()
		}
		cf, _ := config.Load(cmd.Execname,
			config.IgnoreSystemDir(),
			config.IgnoreWorkingDir(),
		)
		cf.SetKeyValues(params...)

		// fix breaking change
		if cf.IsSet("itrshome") {
			if !cf.IsSet("geneos") {
				cf.Set("geneos", cf.GetString("itrshome"))
			}
			cf.Set("itrshome", nil)
		}

		return cf.Save(cmd.Execname)
	},
}
