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
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(homeCmd)
}

// homeCmd represents the home command
var homeCmd = &cobra.Command{
	Use:     "home [flags] [TYPE] [NAME]",
	GroupID: GROUP_VIEW,
	Short:   "Output a directory path for given options",
	Long: strings.ReplaceAll(`
Output a directory path for use in shell expansion like |cd $(geneos
home mygateway)|.

Without arguments, the output will be the root of the Geneos
installation, if defined, or an empty string if not. In the latter
case a shell running |cd| would interpret this as go to your home
directory.

With only a TYPE and no instance NAME the output is the directory
root directory of that TYPE, e.g. |${GENEOS_HOME}/gateway|

Otherwise, if the first NAME argument results in a match to an instance
then the output is it's working directory. If no instance matches the
first NAME argument then the Geneos root directory is output as if no
other options were given.

For obvious reasons this only applies to the local host and the
|--host|/|-H| option is ignored. If NAME is given with a host
qualifier and this is not |localhost| then this is treated as a
failure and the Geneos home directory is returned.

If the resulting path contains whitespace your shell will see this as
multiple arguments and a typical |cd| will fail. To avoid this wrap
the expansion in double quotes, e.g. |cd "$(geneos home 'Demo
Gateway')"|. The best solution is to not use white space in any
instance name or directory path above it. (Note: We tried outputting
a quoted path but the bash shell ignores these quotes inside
|$(...)|)
`, "|", "`"),
	Example: strings.ReplaceAll(`
cd $(geneos home)
cd $(geneos home gateway example1)
cat $(geneos home gateway example2)/gateway.txt
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, _ := CmdArgsParams(cmd)

		if len(args) == 0 {
			if ct == nil {
				fmt.Println(geneos.Root())
				return nil
			}
			fmt.Println(geneos.LOCAL.Filepath(ct))
			return nil
		}

		i, err := instance.Match(ct, args[0])

		if err != nil || i.Host() != geneos.LOCAL {
			fmt.Println(geneos.Root())
			return nil
		}

		fmt.Println(i.Home())
		return nil
	},
}
