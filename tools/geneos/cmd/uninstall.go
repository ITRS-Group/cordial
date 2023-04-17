/*
Copyright © 2023 ITRS Group

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
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var uninstallCmdHost, uninstallCmdVersion string
var uninstallCmdAll, uninstallCmdForce, uninstallCmdDryRun, uninstallCmdRollback bool

func init() {
	rootCmd.AddCommand(uninstallCmd)

	uninstallCmd.Flags().BoolVarP(&uninstallCmdAll, "all", "A", false, "Uninstall all releases, stopping and disabling running instances")
	uninstallCmd.Flags().BoolVarP(&uninstallCmdForce, "force", "f", false, "Force uninstall, stopping instances using matching releases")
	uninstallCmd.Flags().BoolVarP(&uninstallCmdDryRun, "dryrun", "n", false, "Dry run. Show actions but do nothing")
	uninstallCmd.Flags().BoolVarP(&uninstallCmdRollback, "rollback", "R", false, "Rollback version, stopping instances using matching releases")
	uninstallCmd.Flags().StringVarP(&uninstallCmdHost, "host", "H", string(host.ALLHOSTS), "Perform on a remote host. \"all\" means all hosts and locally")
	uninstallCmd.Flags().StringVarP(&uninstallCmdVersion, "version", "V", "", "Uninstall a specific version")

	uninstallCmd.Flags().SortFlags = false
}

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [flags] [TYPE]",
	Short: "Uninstall Geneos releases",
	Long: strings.ReplaceAll(`
Uninstall selected Geneos releases. By default all releases that are
not used by any instance, including disabled instances, are removed
with the exception of the "latest" release for each component type.

If |TYPE| is given then only releases for that component are
uninstalled. Similarly if |--version VERSION| is given then only that
version is removed unless it is in use by an instance (including
disabled instances). Version wildcards are not yet supported.

To remove releases that are in use by instances you must give the
|--force| flag and this will first shutdown any running instance
using that release and update base links and versions to "latest"
unless to |--rollback| flag is used.

If the flag |--rollback| is used then the command will first try to
rollback to the previous release and restart any stopped instances,
or if an earlier release is not installed then it will roll-forward
to the next available release (and restart). Finally, if no other
release is available then the instance will be disabled. Instances
that were not already running are not started.

Any release that is referenced by a symlink (e.g. |active_prod|) will
have the symlink updated as for instances above. This includes the
need to pass |--force| if there are running instances, but unlike
instances that reference releases directly |--force| is not required
if there is no running process using the symlinked release.

Additionally if the |-all| flag is passed then all releases for
matching components are removed and all running instances stopped and
disabled. This can be used to force a "clean install" of a component
or before removal of a Geneos installation on a specific host.

If a host is not selected with the |--host HOST| flags then the
uninstall applies to all configured hosts. 

Use |geneos update ls| to see what is installed.
`, "|", "`"),
	Example: strings.ReplaceAll(`
geneos uninstall netprobe
geneos uninstall --version 5.14.1
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, _ := cmdArgs(cmd)
		h := host.Get(uninstallCmdHost)

		for _, h := range h.Range(host.AllHosts()...) {
			for _, ct := range ct.Range(geneos.RealComponents()...) {
				removeReleases := map[string]geneos.ReleaseDetails{}
				if ct.RelatedTypes != nil {
					log.Debug().Msgf("skipping %s as has related types, remove those instead", ct)
					continue
				}

				v, err := geneos.GetReleases(h, ct)
				if err != nil {
					return err
				}

				// save potential candidates for removal
				for _, i := range v {
					if uninstallCmdAll ||
						(uninstallCmdVersion == "" && !i.Latest) ||
						uninstallCmdVersion == i.Version {
						removeReleases[i.Version] = i
					}
				}

				if uninstallCmdAll || uninstallCmdVersion != "" {
					// ...
				}

				// default behaviour is to remove all releases except
				// the latest.
				//
				// loop over all instances and remove versions from a
				// list as they are found so we end up with a map
				// containing only releases to be removed
				for _, c := range instance.GetAll(h, ct) {
					_, version, err := instance.Version(c)
					if err != nil {
						continue
					}
					delete(removeReleases, version)
				}

				// directory that contains releases for this component
				// on the selected host
				basedir := h.Filepath("packages", ct.String())

				for version, release := range removeReleases {
					releaseDir := filepath.Join(basedir, version)
					if len(release.Links) != 0 {
						previousVersion, nextVersion, _ := geneos.AdjacentVersions(h, ct, version)

						if uninstallCmdRollback && previousVersion != "" {
							if !uninstallCmdDryRun {
								updateLinks(h, basedir, release, version, previousVersion)
							}
							delete(removeReleases, previousVersion)
						} else if nextVersion != "" {
							if !uninstallCmdDryRun {
								updateLinks(h, basedir, release, version, nextVersion)
							}
							delete(removeReleases, nextVersion)
						} else {
							if !uninstallCmdDryRun {
								for _, link := range release.Links {
									linkpath := filepath.Join(basedir, link)
									if err = h.Remove(linkpath); err != nil && !errors.Is(err, fs.ErrNotExist) {
										log.Error().Err(err)
										continue
									}
									fmt.Printf("removed unused link %s\n", link)
								}
							}
						}
					}

					// remove the release
					if !uninstallCmdDryRun {
						if err = h.RemoveAll(releaseDir); err != nil {
							log.Error().Err(err)
							continue
						}
					}
					fmt.Printf("removed %s release %s\n", ct, version)
				}
			}
		}

		return
	},
}

// updateLinks removes the base symlink for oldVersion and recreates a
// new one pointing to target. It also updates all other links in the
// map to the same old target to the new one.
func updateLinks(h *host.Host, releaseDir string, release geneos.ReleaseDetails, oldVersion, newVersion string) (err error) {
	for _, l := range release.Links {
		link := filepath.Join(releaseDir, l)
		if err = h.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Error().Err(err)
			continue
		}
		if err = h.Symlink(newVersion, link); err != nil {
			log.Error().Err(err)
			continue
		}
		fmt.Printf("updated %s, now linked to %s\n", l, newVersion)
	}

	return
}
