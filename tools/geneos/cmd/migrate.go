/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"

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

//go:embed _docs/migrate.md
var migrateCmdDescription string

var migrateCmd = &cobra.Command{
	Use:          "migrate [--executables|-X] | [TYPE] [NAME...]",
	GroupID:      CommandGroupConfig,
	Short:        "Migrate Instance Configurations",
	Long:         migrateCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names := ParseTypeNames(cmd)
		if migrateCmdExecutables {
			if err := migrateCommands(); err != nil {
				log.Error().Err(err).Msg("migrating old executables failed")
			}
		}
		instance.Do(geneos.GetHost(Hostname), ct, names, migrateInstance).Write(os.Stdout)
		return
	},
}

func migrateInstance(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if resp.Err = instance.Migrate(i); resp.Err != nil {
		resp.Err = fmt.Errorf("cannot migrate configuration: %w", resp.Err)
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
