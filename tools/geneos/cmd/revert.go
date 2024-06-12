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
	"os"
	"os/exec"
	"path/filepath"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var revertCmdExecutables bool

func init() {
	GeneosCmd.AddCommand(revertCmd)

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
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := ParseTypeNames(cmd)
		if revertCmdExecutables {
			revertCommands()
			return
		}
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)

			if instance.IsProtected(i) {
				resp.Err = geneos.ErrProtected
				return
			}

			// if *.rc file exists, remove rc.orig+new, continue
			if _, err := i.Host().Stat(instance.ComponentFilepath(i, "rc")); err == nil { // found ?
				// ignore errors
				if i.Host().Remove(instance.ComponentFilepath(i, "rc", "orig")) == nil || i.Host().Remove(instance.ComponentFilepath(i)) == nil {
					log.Debug().Msgf("%s removed extra config file(s)", i)
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
		}).Write(os.Stdout)
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
		path, err := exec.LookPath(ct.String() + "ctl")
		if err != nil {
			continue
		}
		realpath, err := filepath.EvalSymlinks(path)
		if err != nil {
			continue
		}
		realpath = filepath.ToSlash(realpath)
		if realpath != geneosExec {
			log.Debug().Msgf("%s is not a link to %s, skipping", path, geneosExec)
			continue
		}

		if err = os.Remove(path); err != nil {
			log.Fatal().Err(err).Msgf("cannot remove symlink %s, please take action to resolve", path)
		}
		if err = os.Rename(path+".orig", path); err != nil {
			log.Fatal().Err(err).Msgf("cannot rename %s.orig, please take action to resolve", path)
		}
		fmt.Printf("%s reverted\n", path)
	}
	return
}
