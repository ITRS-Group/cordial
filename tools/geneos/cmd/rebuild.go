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

// rebuildCmd represents the rebuild command
var rebuildCmd = &cobra.Command{
	Use:                   "rebuild [-F] [-r] [TYPE] [NAME...]",
	Short:                 "Rebuild instance configuration files",
	Long:                  `Rebuild instance configuration files based on current templates and instance configuration values.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return instance.ForAll(ct, rebuildInstance, args, params)
	},
}

func init() {
	rootCmd.AddCommand(rebuildCmd)

	rebuildCmd.Flags().BoolVarP(&rebuildCmdForce, "force", "F", false, "Force rebuild")
	rebuildCmd.Flags().BoolVarP(&rebuildCmdReload, "reload", "r", false, "Reload instances after rebuild")
	rebuildCmd.Flags().SortFlags = false
}

var rebuildCmdForce, rebuildCmdReload bool

func rebuildInstance(c geneos.Instance, params []string) (err error) {
	if err = c.Rebuild(rebuildCmdForce); err != nil {
		return
	}
	logDebug.Println(c, "configuration rebuilt (if supported)")
	if !rebuildCmdReload {
		return
	}
	return reloadInstance(c, params)
}
