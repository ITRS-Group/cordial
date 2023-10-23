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
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "false",
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
				for _, c := range instance.GetAll(h, nil) {
					if err = instance.Stop(c, deleteCmdForce, false); err != nil && !errors.Is(err, os.ErrProcessDone) {
						return
					}
					if deleteCmdRecurse {
						if deleteCmdForce || instance.IsDisabled(c) {
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
			h.Delete()
			fmt.Printf("%q deleted\n", h)
		}
		geneos.SaveHostConfig()

		return nil
	},
}
