/*
Copyright © 2023 ITRS Group

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

	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/spf13/cobra"
)

var uninstallCmdHost, uninstallCmdVersion string
var uninstallCmdForce bool

func init() {
	rootCmd.AddCommand(uninstallCmd)

	uninstallCmd.Flags().BoolVarP(&uninstallCmdForce, "force", "f", false, "Force uninstall, stopping instances using matching releases")
	uninstallCmd.Flags().StringVarP(&uninstallCmdHost, "host", "H", string(host.ALLHOSTS), "Perform on a remote host. \"all\" means all hosts and locally")
	uninstallCmd.Flags().StringVarP(&uninstallCmdVersion, "version", "V", "", "Uninstall a specific version")

	uninstallCmd.Flags().SortFlags = false
}

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [flags] [TYPE]",
	Short: "Uninstall Geneos releases",
	Long: strings.ReplaceAll(`
Uninstall selected Geneos releases. By default all releases that are
not used by an instance, including disabled instances, are removed
with the exception of the "latest" release for each component type.

If |TYPE| is given then only releases for that component are
uninstalled. Similarly if |--version VERSION| is given then only that
version is installed unless it is in use by an instance (including
disabled instances). Version wildcards are not yet supported.

To remote releases that are in use by instances you must give the
|--force| flag and this will first shutdown any running instance
using that releases, and the it will first try to rollback to the
previous release and restart any stopped instances, or if an earlier
release is not installed then it will roll-forward to the next
available release (and restart). Finally, if no other release is
available then the instance will be disabled. Instances that were not
already running are not started.

Any release that is referenced by a symlink (e.g. |active_prod|) will
have the symlink updated as for instances above. This includes the
need to pass |--force| if there are running instances, but unlike
instances that reference releases directly |--force| is not required
if there is no running process using the symlinked release.

If a host is not selected with the |--host HOST| flags then the
uninstall applies to all configured hosts. 

Use |geneos update ls| to see what is installed.
`, "|", "`"),
	Example: strings.ReplaceAll(`
geneos uninstall netprobe
geneos uninstall --version 5.14.1
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		fmt.Println("uninstall called - not yet implemented")
		return
	},
}
