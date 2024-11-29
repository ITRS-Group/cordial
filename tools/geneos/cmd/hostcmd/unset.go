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

package hostcmd

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var unsetCmdWarned bool
var unsetCmdKeys instance.UnsetValues
var unsetCmdPrivateKeyfiles PrivateKeyFiles

func init() {
	hostCmd.AddCommand(unsetCmd)

	unsetCmd.Flags().VarP(&unsetCmdKeys, "key", "k", "Unset configuration parameter `KEY`\n(Repeat as required)")
	unsetCmd.Flags().VarP(&unsetCmdPrivateKeyfiles, "privatekey", "i", "Private key file")

	unsetCmd.Flags().SortFlags = false
}

//go:embed _docs/unset.md
var unsetCmdDescription string

var unsetCmd = &cobra.Command{
	Use:   "unset [flags] [TYPE] [NAME...]",
	Short: "Unset Host Parameters",
	Long:  unsetCmdDescription,
	Example: strings.ReplaceAll(`
geneos host unset rem2 -i /path/to/id_rsa
`, "|", "`"),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, origargs []string) (err error) {
		var hosts []*geneos.Host

		if len(origargs) == 0 && command.Flags().NFlag() == 0 {
			command.Usage()
			return
		}
		_, args := cmd.ParseTypeNames(command)

		hosts = geneos.RemoteHosts(false)
		for _, a := range args {
			h := geneos.GetHost(a)
			if h.Exists() {
				hosts = append(hosts, h)
			}
		}
		if len(hosts) == 0 {
			// nothing to do
			fmt.Println("nothing to do")
			return
		}

		log.Debug().Msgf("%d hosts: %v", len(hosts), hosts)

		for _, h := range hosts {
			if len(unsetCmdKeys) > 0 {
				settings := h.AllSettings()
				log.Debug().Msgf("host %s settings %#v", h, settings)
				for _, key := range unsetCmdKeys {
					log.Debug().Msgf("deleting %s from host %s", key, h)
					delete(settings, key)
				}
				// you can't delete keys in viper, so create an empty config, and merge in the settings left
				h.Config = config.New()
				h.MergeConfigMap(settings)
				log.Debug().Msgf("host %s settings %#v", h, settings)
			}

			if len(unsetCmdPrivateKeyfiles) > 0 {
				keys := h.GetStringSlice("privatekeys")
				if len(keys) == 0 {
					continue
				}
				keys = slices.DeleteFunc(keys, func(key string) bool {
					if slices.Contains(unsetCmdPrivateKeyfiles, key) {
						return true
					}
					return false
				})
				h.Set("privatekeys", keys)
			}
		}

		return geneos.SaveHostConfig()
	},
}
