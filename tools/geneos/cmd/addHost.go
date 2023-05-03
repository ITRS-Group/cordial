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
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	addCmd.AddCommand(addHostCmd)

	addHostCmd.Flags().SortFlags = false
}

var addHostCmd = &cobra.Command{
	Use:     "host [flags] [NAME] [SSHURL]",
	Aliases: []string{"remote"},
	Short:   "Alias for `host add`",
	Long: strings.ReplaceAll(`
Alias for |host add|. Please use |geneos host add| in the future as
this alias will be removed in an upcoming release.
`, "|", "`"),
	SilenceUsage: true,
	Args:         cobra.RangeArgs(1, 2),
	Annotations: map[string]string{
		"wildcard": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"host", "add"}, args)
	},
}
