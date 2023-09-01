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
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var updateCmdBase, updateCmdVersion string
var updateCmdForce, updateCmdRestart, updateCmdInstall bool

func init() {
	packageCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVarP(&updateCmdVersion, "version", "V", "latest", "Update to this version, defaults to latest")

	updateCmd.Flags().BoolVarP(&updateCmdInstall, "install", "I", false, "Install package updates if necessary")
	updateCmd.Flags().StringVarP(&updateCmdBase, "base", "b", "active_prod", "Base name for the symlink, defaults to active_prod")
	updateCmd.Flags().BoolVarP(&updateCmdRestart, "restart", "R", true, "Restart all instances that may have an update applied")

	updateCmd.Flags().BoolVarP(&updateCmdForce, "force", "F", false, "Will also update and restart protected instances")

	updateCmd.Flags().SortFlags = false
}

//go:embed _docs/update.md
var updateCmdDescription string

var updateCmd = &cobra.Command{
	Use:   "update [flags] [TYPE] [VERSION]",
	Short: "Update the active version of installed Geneos package",
	Long:  updateCmdDescription,
	Example: strings.ReplaceAll(`
geneos package update gateway -b active_prod
geneos package update gateway -b active_dev -V 5.11
geneos package update
geneos package update netprobe 5.13.2
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "true",
	},
	Args: cobra.RangeArgs(0, 2),
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.TypeNamesParams(command)

		for _, p := range params {
			if strings.HasPrefix(p, "@") {
				return fmt.Errorf("@HOST not valid here, please use the `--host`/`-H` option")
			}
		}

		h := geneos.GetHost(cmd.Hostname)

		// try to install the wanted version, latest (the default) means check
		if updateCmdInstall {
			types := make(map[string]bool)
			cs := instance.GetAll(h, ct)
			for _, c := range cs {
				if pt := c.Config().GetString("pkgtype"); pt != "" {
					types[pt] = true
				} else {
					types[c.Type().String()] = true
				}
			}
			for _, h := range h.OrList() {
				for t := range types {
					ct := geneos.ParseComponent(t)
					if err = geneos.Install(h, ct, geneos.Version(updateCmdVersion)); err != nil {
						if errors.Is(err, fs.ErrNotExist) {
							continue
						}
						log.Error().Err(err).Msg("")
					}
				}
			}
		}

		version := updateCmdVersion
		cs := instance.ByKeyValue(h, ct, "protected", "true")
		if len(cs) > 0 && !updateCmdForce {
			fmt.Println("There are one or more protected instances using the current version. Use `--force` to override")
			return
		}
		if len(args) > 0 {
			version = args[0]
		}
		var instances []geneos.Instance
		if updateCmdRestart {
			instances = instance.ByKeyValue(h, ct, "version", updateCmdBase)
		}
		if err = geneos.Update(h, ct,
			geneos.Version(version),
			geneos.Basename(updateCmdBase),
			geneos.Force(true),
			geneos.Restart(instances...),
			geneos.StartFunc(instance.Start),
			geneos.StopFunc(instance.Stop)); err != nil && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return
	},
}
