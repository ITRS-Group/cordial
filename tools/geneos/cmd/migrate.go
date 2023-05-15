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
	"os"
	"os/exec"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var migrateCmdExecutables bool

func init() {
	GeneosCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().BoolVarP(&migrateCmdExecutables, "executables", "X", false, "Migrate executables by symlinking to this binary")
	migrateCmd.Flags().SortFlags = false
}

var migrateCmd = &cobra.Command{
	Use:   "migrate [TYPE] [NAME...]",
	Short: "Migrate legacy .rc configuration to new formats",
	Long: strings.ReplaceAll(`
Migrate any legacy .rc configuration files to JSON format and
rename the .rc file to .rc.orig. The entries in the new configuration
take on the new labels and are not a direct conversion.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		if migrateCmdExecutables {
			if err := migrateCommands(); err != nil {
				log.Error().Err(err).Msg("migrating old executables failed")
			}
		}
		return instance.ForAll(ct, migrateInstance, args, params)
	},
}

func migrateInstance(c geneos.Instance, params []string) (err error) {
	if err = instance.Migrate(c); err != nil {
		log.Error().Err(err).Msgf("%s cannot migrate configuration", c)
	}
	return
}

// search PATH for *ctl commands, and if they are not links to 'geneos'
// then update then, permissions allowing
func migrateCommands() (err error) {
	geneosExec, err := os.Executable()
	if err != nil {
		return
	}

	for _, ct := range geneos.RealComponents() {
		path, err := exec.LookPath(ct.String() + "ctl")
		if err != nil {
			continue
		}
		if err = os.Rename(path, path+".orig"); err != nil {
			fmt.Printf("cannot rename %q to .orig (skipping): %s\n", path, err)
			continue
		}
		if err = os.Symlink(geneosExec, path); err != nil {
			if err = os.Rename(path+".orig", path); err != nil {
				log.Fatal().Err(err).Msgf("cannot restore %s after backup (to .orig), please fix manually", path)
			}
			fmt.Printf("cannot link %s to %s (skipping): %s", path, geneosExec, err)
			continue
		}
		fmt.Printf("%s migrated\n", path)
	}
	return
}
