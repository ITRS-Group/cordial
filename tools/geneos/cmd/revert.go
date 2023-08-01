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
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args := CmdArgs(cmd)
		if revertCmdExecutables {
			return revertCommands()
		}
		return instance.ForAll(ct, Hostname, revertInstance, args)
	},
}

func revertInstance(c geneos.Instance, _ ...any) (err error) {
	if instance.IsProtected(c) {
		return geneos.ErrProtected
	}

	// if *.rc file exists, remove rc.orig+new, continue
	if _, err := c.Host().Stat(instance.ComponentFilepath(c, "rc")); err == nil {
		// ignore errors
		if c.Host().Remove(instance.ComponentFilepath(c, "rc", "orig")) == nil || c.Host().Remove(instance.ComponentFilepath(c)) == nil {
			log.Debug().Msgf("%s removed extra config file(s)", c)
		}
		return err
	}

	if err = c.Host().Rename(instance.ComponentFilepath(c, "rc", "orig"), instance.ComponentFilepath(c, "rc")); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return
	}

	if err = c.Host().Remove(instance.ComponentFilepath(c)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return
	}

	log.Debug().Msgf("%s reverted to RC config", c)
	return nil
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
			log.Fatal().Err(err).Msgf("cannot remove symlink %p, please take action to resolve", path)
		}
		if err = os.Rename(path+".orig", path); err != nil {
			log.Fatal().Err(err).Msgf("cannot rename %s.orig, please take action to resolve", path)
		}
		fmt.Printf("%s reverted\n", path)
	}
	return
}
