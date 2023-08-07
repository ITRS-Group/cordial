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
	Use:          "revert [TYPE] [NAME...]",
	GroupID:      CommandGroupConfig,
	Short:        "Revert Migrated Instance Configuration",
	Long:         revertCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := TypeNames(cmd)
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
