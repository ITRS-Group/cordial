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
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(cleanCmd)

	cleanCmd.Flags().BoolVarP(&cleanCmdFull, "full", "F", false, "Perform a full clean. Removes more files than basic clean and restarts instances")
	cleanCmd.Flags().SortFlags = false
}

var cleanCmdFull bool

var cleanCmd = &cobra.Command{
	Use:     "clean [flags] [TYPE] [NAME...]",
	GroupID: GROUP_MANAGE,
	Short:   "Clean-up instance directories",
	Long: strings.ReplaceAll(`
Clean the working directories for all matching instances.

The default behavior is to leave the instance running and only inactive files
are removed.

With the |--full|/|-F| option, the command will stop the
instance, remove all non-essential files from the working
directory of the instance and restart the instance.

**Note**: Files removed by |geneos clean| are defined in the geneos main
configuration file |geneos.json| as |[TYPE]CleanList|. Files removed by
|geneos clean -F| or |geneos clean --full| are defined in the geneos main
configuration file |geneos.json| as |[TYPE]PurgeList|. Both these lists are
formatted as a PathListSeparator (typically a colon) separated list of file
globs.
`, "|", "`"),
	Example: strings.ReplaceAll(`
# Delete old logs and config file backups without affecting the running
# instance
geneos clean gateway Gateway1
# Stop all netprobes and remove all non-essential files from working 
# directories, then restart netprobes
geneos clean --full netprobe
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return instance.ForAll(ct, Hostname, cleanInstance, args, params)
	},
}

func cleanInstance(c geneos.Instance, params []string) (err error) {
	return instance.Clean(c, geneos.FullClean(cleanCmdFull))
}
