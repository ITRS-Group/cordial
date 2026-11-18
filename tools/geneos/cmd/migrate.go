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
	"cmp"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

var migrateCmdExecutables bool

func init() {
	Cmd.AddCommand(migrateCmd)

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
		CmdGlobal:               "true",
		CmdRequireHome:          "true",
		CmdWildcardNames:        "true",
		CmdNonInstanceArgsError: "true",
	},
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		log := log.With("command", "migrate")

		ct, names, _, err := FetchArgs(cmd)
		if err != nil {
			return
		}
		var executablesErr error
		if migrateCmdExecutables {
			if executablesErr = migrateCommands(); executablesErr != nil && !errors.Is(executablesErr, ErrReported) {
				log.Error("migrating old executables failed", slog.Any("error", executablesErr))
				return executablesErr
			}
		}
		// the instances are migrated even if some executables were not
		return cmp.Or(instance.Do(cmd.CommandPath(), geneos.GetHost(Hostname), ct, names, migrateInstance).Report(os.Stdout), executablesErr)
	},
}

func migrateInstance(i geneos.Instance, _ ...any) (resp *responses.General) {
	if resp = instance.Migrate(i); resp.Err != nil {
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
		// do not migrate again, which would replace the saved original
		// with the link
		if realpath, err := filepath.EvalSymlinks(path); err == nil && filepath.ToSlash(realpath) == filepath.ToSlash(geneosExec) {
			fmt.Printf("%s already migrated (skipping)\n", path)
			continue
		}
		if _, err := os.Lstat(path + ".orig"); err == nil {
			fmt.Printf("%s.orig already exists (skipping)\n", path)
			continue
		}
		if renameErr := os.Rename(path, path+".orig"); renameErr != nil {
			err = ReportFailure(path, fmt.Errorf("cannot rename to .orig (skipping): %w", renameErr))
			continue
		}
		if linkErr := os.Symlink(geneosExec, path); linkErr != nil {
			if rerr := os.Rename(path+".orig", path); rerr != nil {
				log.Error("cannot restore after backup", slog.Any("error", rerr), slog.String("file", path))
			}
			err = ReportFailure(path, fmt.Errorf("cannot link to %s (skipping): %w", geneosExec, linkErr))
			continue
		}
		fmt.Printf("%s migrated\n", path)
	}
	return
}
