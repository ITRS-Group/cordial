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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// homeCmd represents the home command
var homeCmd = &cobra.Command{
	Use:   "home [TYPE] [NAME]",
	Short: "Print the home directory of the first instance or the Geneos home dir",
	Long: `Output the home directory of the first matching instance or local
installation or the remote on stdout. This is intended for scripting,
e.g.

	cd $(geneos home)
	cd $(geneos home gateway example1)
		
No errors are logged. An error, for example no matching instance found, result in the Geneos
root directory being printed.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandHome(ct, args, params)
	},
}

func init() {
	rootCmd.AddCommand(homeCmd)
	homeCmd.Flags().SortFlags = false
}

func commandHome(ct *geneos.Component, args []string, params []string) error {
	if ct == nil && len(args) == 0 {
		fmt.Println(host.Geneos())
		return nil
	}

	var i []geneos.Instance
	if len(args) == 0 {
		i = instance.GetAll(host.LOCAL, ct)
	} else {
		i = instance.MatchAll(ct, args[0])
	}

	if len(i) == 0 {
		fmt.Println(host.Geneos())
		return nil
	}

	fmt.Println(i[0].Home())
	return nil
}
