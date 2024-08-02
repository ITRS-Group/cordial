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
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var deleteCmdForce, deleteCmdRecurse, deleteCmdStop bool

func init() {
	hostCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVarP(&deleteCmdForce, "force", "F", false, "Delete instances without checking if disabled")
	deleteCmd.Flags().BoolVarP(&deleteCmdRecurse, "all", "R", false, "Recursively delete all instances on the host before removing the host config")
	deleteCmd.Flags().BoolVarP(&deleteCmdStop, "stop", "S", false, "Stop all instances on the host before deleting the local entry")

	deleteCmd.Flags().SortFlags = false
}

//go:embed _docs/delete.md
var deleteCmdDescription string

var deleteCmd = &cobra.Command{
	Use:          "delete [flags] NAME...",
	Aliases:      []string{"rm", "remove"},
	Short:        "Delete a remote host configuration",
	Long:         deleteCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdNoneMeansAll: "false",
		cmd.CmdRequireHome:  "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		_, args := cmd.ParseTypeNames(command)
		if len(args) == 0 {
			return geneos.ErrInvalidArgs
		}

		// check args are hosts ('any' means all non-local ?)
		var hosts []*geneos.Host
		for _, hostname := range args {
			h := geneos.GetHost(hostname)
			if !h.Exists() {
				log.Error().Msgf("%q is not a known host", hostname)
				return
			}
			hosts = append(hosts, h)
		}

		if deleteCmdRecurse {
			deleteCmdStop = true
		}

		for _, h := range hosts {
			// stop and/or delete instances on host
			if deleteCmdStop {
				instances, err := instance.Instances(h, nil)
				if err != nil {
					panic(err)
				}
				for _, c := range instances {
					if err = instance.Stop(c, deleteCmdForce, false); err != nil && !errors.Is(err, os.ErrProcessDone) {
						return err
					}
					if deleteCmdRecurse {
						if deleteCmdForce || instance.IsDisabled(c) {
							if err = c.Host().RemoveAll(c.Home()); err != nil {
								return err
							}
							fmt.Printf("%s deleted %s:%s\n", c, c.Host().String(), c.Home())
							c.Unload()
						} else {
							fmt.Printf("not deleting %q as it is not disabled and no --force flag given\n", c)
							return geneos.ErrInvalidArgs
						}
					}
				}
			}

			// remove host config
			h.Delete()
			fmt.Printf("%q deleted\n", h)
		}
		geneos.SaveHostConfig()

		return nil
	},
}
