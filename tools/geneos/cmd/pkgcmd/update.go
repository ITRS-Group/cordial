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

package pkgcmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var packageUpdateCmdBase, packageUpdateCmdVersion string
var packageUpdateCmdForce, packageUpdateCmdRestart bool

func init() {
	PackageCmd.AddCommand(packageUpdateCmd)

	packageUpdateCmd.Flags().StringVarP(&packageUpdateCmdVersion, "version", "V", "latest", "Update to this version, defaults to latest")

	packageUpdateCmd.Flags().StringVarP(&packageUpdateCmdBase, "base", "b", "active_prod", "Base name for the symlink, defaults to active_prod")
	packageUpdateCmd.Flags().BoolVarP(&packageUpdateCmdForce, "force", "F", false, "Update all protected instances")
	packageUpdateCmd.Flags().BoolVarP(&packageUpdateCmdRestart, "restart", "R", false, "Restart all instances that may have an update applied")

	packageUpdateCmd.Flags().SortFlags = false
}

var packageUpdateCmd = &cobra.Command{
	Use:   "update [flags] [TYPE] [VERSION]",
	Short: "Update the active version of installed Geneos package",
	Long: strings.ReplaceAll(`
Update the symlink from the default base name of the package to
the best match for VERSION. The default base directory is |active_prod|
and is normally linked to the latest version of a component type in the
packages directory. VERSION can either be a semantic version style name or
(the default if not given) |latest|.

If TYPE is not supplied, all supported component types are updated to VERSION.

Update will stop all matching instances of the each type before
updating the link and starting them up again, but only if the
instance uses the same basename.

The matching of VERSION is based on directory names of the form:

|[GA]X.Y.Z|

Where X, Y, Z are each ordered in ascending numerical order. If a
directory starts |GA| it will be selected over a directory with the
same numerical versions. All other directories name formats will
result in unexpected behaviour. If multiple installed versions
match then the lexically latest match will be used. The chosen
match may be much higher than that given on the command line as
only installed packages are used in the search.

If a basename for the symlink does not already exist it will be created,
so it important to check the spelling carefully.
`, "|", "`"),
	Example: strings.ReplaceAll(`
geneos package update gateway -b active_prod
geneos package update gateway -b active_dev 5.11
geneos package update
geneos package update netprobe 5.13.2
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	Args: cobra.RangeArgs(0, 2),
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.CmdArgsParams(command)

		for _, p := range params {
			if strings.HasPrefix(p, "@") {
				return fmt.Errorf("@HOST not valid here, perhaps you meant `-H HOST`?")
			}
		}

		r := geneos.GetHost(cmd.Hostname)

		version := packageUpdateCmdVersion
		cs := instance.MatchKeyValue(r, ct, "protected", "true")
		if len(cs) > 0 && !packageUpdateCmdForce {
			fmt.Println("There are one or more protected instances using the current version. Use `--force` to override")
		}
		if len(args) > 0 {
			version = args[0]
		}
		options := []geneos.Options{geneos.Version(version), geneos.Basename(packageUpdateCmdBase), geneos.Force(true), geneos.Restart(packageUpdateCmdRestart)}
		if packageUpdateCmdRestart {
			cs := instance.MatchKeyValue(r, ct, "version", packageUpdateCmdBase)
			for _, c := range cs {
				instance.Stop(c, packageUpdateCmdForce, false)
				defer instance.Start(c)
			}
		}
		if err = geneos.Update(r, ct, options...); err != nil && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return
	},
}
