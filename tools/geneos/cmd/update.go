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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [FLAGS] [TYPE] [VERSION]",
	Short: "Update the active version of Geneos software",
	Long: `Update the symlink from the default base name of the package to
the best match for VERSION. The default base directory is 'active_prod'
and is normally linked to the latest version of a component type in the
packages directory. VERSION can either be a semantic version style name or
(the default if not given) 'latest'.

If TYPE is not supplied, all supported component types are updated to VERSION.

Update will stop all matching instances of the each type before
updating the link and starting them up again, but only if the
instance uses the same basename.

The matching of VERSION is based on directory names of the form:

[GA]X.Y.Z

Where X, Y, Z are each ordered in ascending numerical order. If a
directory starts 'GA' it will be selected over a directory with the
same numerical versions. All other directories name formats will
result in unexpected behaviour. If multiple installed versions
match then the lexically latest match will be used. The chosen
match may be much higher than that given on the command line as
only installed packages are used in the search.

If a basename for the synlink does not already exist it will be created,
so it important to check the spelling carefully.
`,
	Example: `
geneos update gateway -b active_dev 5.11
geneos update
geneos update netprobe 5.13.2
`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	Args: cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandUpdate(ct, args, params)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVarP(&cmdUpdateBase, "base", "b", "active_prod", "Base name for the symlink, defaults to active_prod")
	updateCmd.Flags().StringVarP(&cmdUpdateHost, "host", "H", string(host.ALLHOSTS), "Apply only on remote host. \"all\" (the default) means all remote hosts and locally")
	updateCmd.Flags().BoolVarP(&cmdUpdateRestart, "restart", "R", false, "Restart all instances that may have an update applied")
	updateCmd.Flags().SortFlags = false
}

var cmdUpdateBase, cmdUpdateHost string
var cmdUpdateRestart bool

func commandUpdate(ct *geneos.Component, args []string, params []string) (err error) {
	version := "latest"
	if len(args) > 0 {
		version = args[0]
	}
	r := host.Get(cmdUpdateHost)
	options := []geneos.GeneosOptions{geneos.Version(version), geneos.Basename(cmdUpdateBase), geneos.Force(true), geneos.Restart(cmdUpdateRestart)}
	if cmdUpdateRestart {
		cs := instance.MatchKeyValue(host.ALL, ct, "version", cmdUpdateBase)
		for _, c := range cs {
			instance.Stop(c, false)
			defer instance.Start(c)
		}
	}
	if err = geneos.Update(r, ct, options...); err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return
}
