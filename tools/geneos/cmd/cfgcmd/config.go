/*
Copyright © 2023 ITRS Group

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

// Package cfgcmd groups config commands in their own package
package cfgcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	cmd.GeneosCmd.AddCommand(ConfigCmd)
}

// ConfigCmd represents the config command
var ConfigCmd = &cobra.Command{
	Use:     "config",
	GroupID: cmd.GROUP_SUBSYSTEMS,
	Short:   "Configure geneos command environment",
	Long: strings.ReplaceAll(`
The commands in the |config| subsystem allow you to control the
environment of the |geneos| program itself. Please see the
descriptions of the commands below for more information.

If you run this command directly then you will either be shown the
output of |geneos config show| or |geneos config set [ARGS]| if you
supply any further arguments that contain an "=".
`, "|", "`"),
	Example: `
geneos config
geneos config geneos=/opt/itrs
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		var doSet bool
		for _, a := range args {
			if strings.Contains(a, "=") {
				doSet = true
			}
		}
		if doSet {
			return cmd.RunE(command.Root(), []string{"config", "set"}, args)
		}
		return cmd.RunE(command.Root(), []string{"config", "show"}, args)
	},
}
