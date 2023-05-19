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
	_ "embed"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	cmd.GeneosCmd.AddCommand(configCmd)
}

//go:embed README.md
var configCmdDescription string

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:     "config",
	GroupID: cmd.CommandGroupSubsystems,
	Short:   "Configure the command environment",
	Long:    configCmdDescription,
	Example: `
geneos config
geneos config geneos=/opt/itrs
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
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
