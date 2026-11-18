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
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

var revertCmdExecutables bool

func init() {
	Cmd.AddCommand(revertCmd)

	revertCmd.Flags().BoolVarP(&revertCmdExecutables, "executables", "X", false, "Revert 'ctl' executables")
	revertCmd.Flags().SortFlags = false
}

//go:embed _docs/revert.md
var revertCmdDescription string

var revertCmd = &cobra.Command{
	Use:          "revert [--executables|-X] | [TYPE] [NAME...]",
	GroupID:      CommandGroupConfig,
	Short:        "Revert Migrated Instance Configuration",
	Long:         revertCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:                "true",
		CmdRequireHome:           "true",
		CmdWildcardNames:         "true",
		CmdAllInstancesMustMatch: "true",
		CmdNonInstanceArgsError:  "true",
	},
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names, _, err := FetchArgs(cmd)
		if err != nil {
			return
		}
		if revertCmdExecutables {
			return revertCommands()
		}
		return instance.Do(cmd.CommandPath(), geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *responses.General) {
			resp = responses.New[responses.General](i)

			if instance.IsProtected(i) {
				resp.Err = geneos.ErrProtected
				return
			}

			// if *.rc file exists, remove rc.orig+new, continue
			if _, err := i.Host().Stat(instance.ComponentFilepath(i, "rc")); err == nil { // found ?
				// try to remove both files, only a file that is already
				// gone is not an error
				origErr := i.Host().Remove(instance.ComponentFilepath(i, "rc", "orig"))
				jsonErr := i.Host().Remove(instance.ComponentFilepath(i))
				if resp.Err = errors.Join(ignoreNotExist(origErr), ignoreNotExist(jsonErr)); resp.Err == nil {
					i.Log().Debug("removed extra config file(s)")
				}
				return
			}

			if err := i.Host().Rename(instance.ComponentFilepath(i, "rc", "orig"), instance.ComponentFilepath(i, "rc")); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return
				}
				resp.Err = err
				return
			}

			if err := i.Host().Remove(instance.ComponentFilepath(i)); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return
				}
				resp.Err = err
				return
			}

			resp.Completed = append(resp.Completed, "reverted to RC config")
			return
		}).Report(os.Stdout)
	},
}

// search PATH for *ctl commands, and if they are links to 'geneos'
// then try to revert them from .orig, permissions allowing
func revertCommands() (err error) {
	geneosExec, err := os.Executable()
	if err != nil {
		return
	}

	for _, ct := range geneos.RealComponents() {
		path, lookErr := exec.LookPath(ct.String() + "ctl")
		if lookErr != nil {
			continue
		}
		realpath, linkErr := filepath.EvalSymlinks(path)
		if linkErr != nil {
			continue
		}
		realpath = filepath.ToSlash(realpath)
		if realpath != geneosExec {
			log.Debug("not a link to geneos executable", slog.String("path", path), slog.String("geneosExec", geneosExec))
			continue
		}

		if rmErr := os.Remove(path); rmErr != nil {
			err = ReportFailure(path, fmt.Errorf("cannot remove symlink: %w", rmErr))
			continue
		}
		if mvErr := os.Rename(path+".orig", path); mvErr != nil {
			err = ReportFailure(path, fmt.Errorf("cannot rename symlink: %w", mvErr))
			continue
		}
		fmt.Printf("%s reverted\n", path)
	}
	return
}

// ignoreNotExist returns err unless it means that a file did not exist.
func ignoreNotExist(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
