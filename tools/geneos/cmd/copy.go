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

	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(copyCmd)

	// copyCmd.Flags().SortFlags = false
}

var copyCmd = &cobra.Command{
	Use:     "copy [TYPE] SOURCE DESTINATION",
	Aliases: []string{"cp"},
	Short:   "Copy instances",
	Long: strings.ReplaceAll(`
Copy instance SOURCE to DESTINATION. If TYPE is not given than each
component type that has a named instance SOURCE will be copied to
DESTINATION. If DESTINATION is given as an @ followed by a remote
host then the instance is copied to the remote host but the name
retained. This can be used, for example, to create a standby Gateway
on another host.

Any instance using a legacy .rc file is migrated to a newer
configuration file format during the copy.

The instance is stopped before and started after the instance is
copied. It is an error to try to copy an instance to one that already
exists with the same name on the same host.

The configured port number, if there is one for that TYPE, is updated
if the existing one is already in use, otherwise it is left
unchanged.

If the component support Rebuild then this is run after the copy but
before the restart. This allows SANs to be updated as expected.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		if len(args) == 0 && len(params) == 2 && strings.HasPrefix(params[0], "@") && strings.HasPrefix(params[1], "@") {
			args = params
		}
		if len(args) == 1 && len(params) == 1 && strings.HasPrefix(params[0], "@") {
			args = append(args, params[0])
		}
		if len(args) != 2 {
			return ErrInvalidArgs
		}

		return instance.CopyInstance(ct, args[0], args[1], false)
	},
}
