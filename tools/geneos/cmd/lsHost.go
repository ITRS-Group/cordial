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
	lsCmd.AddCommand(lsHostCmd)
}

var lsHostCmd = &cobra.Command{
	Use:     "host [flags] [TYPE] [NAME...]",
	Aliases: []string{"hosts", "remote", "remotes"},
	Short:   "Alias for `host ls`",
	Long: strings.ReplaceAll(`
Alias for |host ls|. Please use |geneos host ls| in the future as
this alias will be removed in an upcoming release.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	DisableFlagParsing: true,
	Deprecated:         "Use `geneos host ls` instead.",
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"host", "ls"}, args)
	},
}
