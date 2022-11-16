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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/spf13/cobra"
)

func init() {
	setCmd.AddCommand(setUserCmd)

	// setUserCmd.Flags().SortFlags = false
}

var setUserCmd = &cobra.Command{
	Use:   "user [KEY=VALUE...]",
	Short: "Set user configuration parameters",
	Long: strings.ReplaceAll(`
Set user configuration parameters.

Parameters set using the |geneos set user| command will be written or
updated in file |~/.config/geneos.json|.

**Note**: In case you set a parameter that is not supported, that parameter
will be written to the |json| configuration file, but will have any effect.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		_, _, params := cmdArgsParams(cmd)
		return writeConfigParams(geneos.UserConfigFilePaths()[0], params)
	},
}
