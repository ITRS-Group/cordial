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
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var deleteHostCmdForce, deleteHostCmdRecurse, deleteHostCmdStop bool

func init() {
	deleteCmd.AddCommand(deleteHostCmd)

	deleteHostCmd.Flags().BoolVarP(&deleteHostCmdForce, "force", "F", false, "Delete instances without checking if disabled")
	deleteHostCmd.Flags().BoolVarP(&deleteHostCmdRecurse, "all", "R", false, "Recursively delete all instances on the host before removing the host config")
	deleteHostCmd.Flags().BoolVarP(&deleteHostCmdStop, "stop", "S", false, "Stop all instances on the host before deleting the local entry")

	deleteHostCmd.Flags().SortFlags = false
}

var deleteHostCmd = &cobra.Command{
	Use:     "host [flags] NAME...",
	Aliases: []string{"hosts", "remote", "remotes"},
	Short:   "Delete a remote host configuration",
	Long: strings.ReplaceAll(`
Delete the local configuration referring to a remote host.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		_, args, _ := cmdArgsParams(cmd)
		if len(args) == 0 {
			return geneos.ErrInvalidArgs
		}

		// check args are hosts ('all' means all non-local ?)
		var hosts []*host.Host
		for _, hostname := range args {
			h := host.Get(hostname)
			if !h.Exists() {
				log.Error().Msgf("%q is not a known host", hostname)
				return
			}
			hosts = append(hosts, h)
		}

		if deleteHostCmdRecurse {
			deleteHostCmdStop = true
		}

		for _, h := range hosts {
			// stop and/or delete instances on host
			if deleteHostCmdStop {
				for _, c := range instance.GetAll(h, nil) {
					if err = instance.Stop(c, deleteHostCmdForce, false); err != nil {
						return
					}
					if deleteHostCmdRecurse {
						if deleteHostCmdForce || instance.IsDisabled(c) {
							if err = c.Host().RemoveAll(c.Home()); err != nil {
								return
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
			// if err = host.LOCAL.RemoveAll(h.Home); err != nil {
			// 	return
			// }
			host.Delete(h)
			fmt.Printf("%q deleted\n", h)
		}
		host.WriteHostConfigFile()

		return nil
	},
}
