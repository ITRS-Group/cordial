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
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// copyCmd represents the copy command
var copyCmd = &cobra.Command{
	Use:     "copy [TYPE] SOURCE DESTINATION",
	Aliases: []string{"cp"},
	Short:   "Copy instances",
	Long: `Copy instances. As any existing legacy .rc file is never changed,
this will migrate the instance from .rc to JSON. The instance is
stopped and restarted after the instance is moved. It is an error to
try to copy an instance to one that already exists with the same
name.

If the component support Rebuild then this is run after the move but
before the restart. This allows SANs to be updated as expected.

Moving across hosts is supported.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandCopy(ct, args, params)
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
	copyCmd.Flags().SortFlags = false

}

// use case:
// gateway standby instance copy
// distribute common config netprobe across multiple hosts
// also create hosts as required?
func commandCopy(ct *geneos.Component, args []string, params []string) (err error) {
	if len(args) != 2 {
		return ErrInvalidArgs
	}

	return instance.CopyInstance(ct, args[0], args[1], false)
}
