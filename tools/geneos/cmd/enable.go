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
	"errors"
	"os"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(enableCmd)

	enableCmd.Flags().BoolVarP(&enableCmdStart, "start", "S", false, "Start enabled instances")

	enableCmd.Flags().SortFlags = false
}

var enableCmdStart bool

var enableCmd = &cobra.Command{
	Use:   "enable [flags] [TYPE] [NAME...]",
	Short: "Enable instances",
	Long: strings.ReplaceAll(`
Mark any matching instances as enabled.
Using option |-S| will also start the instance. When using option |-S|, 
only those instances that were disabled are started.

Enabling an instance will result in the file with extension |.disabled|
in the instance's working directory to be removed.

**Note**: Enabling an instance which is not disabled will do nothing and 
have no impact.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return instance.ForAll(ct, enableInstance, args, params)
	},
}

func enableInstance(c geneos.Instance, params []string) (err error) {
	err = c.Host().Remove(instance.ComponentFilepath(c, geneos.DisableExtension))
	if (err == nil || errors.Is(err, os.ErrNotExist)) && enableCmdStart {
		instance.Start(c)
	}
	return nil
}
