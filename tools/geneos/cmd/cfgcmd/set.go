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
}

var setUserCmd = &cobra.Command{
	Use:   "set [KEY=VALUE...]",
	Short: "Set program configuration",
	Long: strings.ReplaceAll(`
Set configuration parameters for the |geneos| program.

Each value is in the form of KEY=VALUE where key is the configuration
item and value an arbitrary string value. Where a KEY is in a
hierarchy use a dot (|.|) as the delimiter.

While you can set arbitrary keys only some have any meaning. The most
important one is |geneos|, the path to the root directory of the
Geneos installation managed by the program. If you change or remove
this value you may break the functionality of the program, so please
be careful.

For an explanation of the various configuration parameters see the
main documentation.
`, "|", "`"),
	Example: `
geneos config set geneos="/opt/geneos"
geneos config set config.rebuild=always
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		_, _, params := cmd.CmdArgsParams(command)

		cf, _ := config.Load(cmd.Execname,
			config.IgnoreSystemDir(),
			config.IgnoreWorkingDir(),
			config.KeyDelimiter("."),
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
