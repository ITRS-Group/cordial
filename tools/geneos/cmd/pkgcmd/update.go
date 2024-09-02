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

package pkgcmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
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

	updateCmd.Flags().StringVarP(&updateCmdVersion, "version", "V", "", "Update to this version, defaults to latest")

	updateCmd.Flags().BoolVarP(&updateCmdInstall, "install", "I", false, "Install package updates if necessary")
	updateCmd.Flags().MarkDeprecated("install", "please use the `package install` command instead")

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
geneos package update netprobe --version 5.13.2
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	Args: cobra.RangeArgs(0, 2),
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.ParseTypeNamesParams(command)

		for _, p := range params {
			if strings.HasPrefix(p, "@") {
				return fmt.Errorf("@HOST not valid here, please use the `--host`/`-H` option")
			}
		}

		h := geneos.GetHost(cmd.Hostname)

		// try to install the wanted version, latest (the default) means check
		if updateCmdInstall {
			types := make(map[string]bool)
			cs, err := instance.Instances(h, ct)
			if err != nil {
				return err
			}
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
						return err
					}
				}
			}
		}

		version := updateCmdVersion
		cs, err := instance.Instances(h, ct, instance.FilterParameters("protected=true", "version="+updateCmdBase))
		if err != nil {
			return err
		}
		if len(cs) > 0 && !updateCmdForce {
			fmt.Println("There are one or more protected instances using the current version. Use `--force` to override")
			return
		}
		if updateCmdVersion == "" && len(args) > 0 {
			version = args[0]
		}

		instances := []geneos.Instance{}
		if updateCmdRestart {
			for _, ct := range ct.OrList() {
				allInstances, err := instance.Instances(h, ct)
				if err != nil {
					return err
				}
				for _, i := range allInstances {
					if i.Config().GetString("version") != updateCmdBase {
						log.Debug().Msgf("%s base different", i)
						continue
					}
					pkg := i.Config().GetString("pkgtype")
					if pkg != "" && pkg == ct.String() {
						instances = append(instances, i)
						continue
					}
					instances = append(instances, i)
				}
			}
			log.Debug().Msgf("instances to restart: %v", instances)
		}

		return geneos.Update(h, ct,
			geneos.Version(version),
			geneos.Basename(updateCmdBase),
			geneos.Force(true),
			geneos.Restart(instances...),
			geneos.StartFunc(instance.Start),
			geneos.StopFunc(instance.Stop))
	},
}
