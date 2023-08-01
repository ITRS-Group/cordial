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

package pkgcmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var uninstallCmdVersion string
var uninstallCmdAll, uninstallCmdForce bool

func init() {
	packageCmd.AddCommand(uninstallCmd)

	uninstallCmd.Flags().StringVarP(&uninstallCmdVersion, "version", "V", "", "Uninstall `VERSION`")
	uninstallCmd.Flags().BoolVarP(&uninstallCmdAll, "all", "A", false, "Uninstall all releases, stopping and disabling running instances")
	uninstallCmd.Flags().BoolVarP(&uninstallCmdForce, "force", "f", false, "Force uninstall, stopping protected instances first")

	uninstallCmd.Flags().SortFlags = false
}

//go:embed _docs/uninstall.md
var uninstallCmdDescription string

var uninstallCmd = &cobra.Command{
	Use:     "uninstall [flags] [TYPE]",
	Short:   "Uninstall Geneos releases",
	Long:    uninstallCmdDescription,
	Aliases: []string{"delete", "remove", "rm"},
	Example: strings.ReplaceAll(`
geneos uninstall netprobe
geneos uninstall --version 5.14.1
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, _ := cmd.TypeNames(command)
		h := geneos.GetHost(cmd.Hostname)

		for _, h := range h.OrList(geneos.AllHosts()...) {
			for _, ct := range ct.OrList(geneos.RealComponents()...) {
				if ct.RelatedTypes != nil {
					log.Debug().Msgf("skipping %s as has related types, remove those instead", ct)
					continue
				}

				r, err := geneos.GetReleases(h, ct)
				if err != nil {
					return err
				}

				// save candidates for removal
				removeReleases := map[string]geneos.ReleaseDetails{}
				for _, i := range r {
					if uninstallCmdAll || // --all
						(uninstallCmdVersion == "" && !i.Latest) || // default leave 'latest'
						uninstallCmdVersion == i.Version { // specific --version
						removeReleases[i.Version] = i
					}
				}

				// loop over all instances and remove versions from a
				// list as they are found so we end up with a map
				// containing only releases to be removed
				//
				// also save a list of instances to restart
				restart := map[string][]geneos.Instance{}
				for _, c := range instance.GetAll(h, ct) {
					if instance.IsDisabled(c) {
						fmt.Printf("%s is disabled, not skipping\n", c)
						continue
					}

					_, version, err := instance.Version(c)
					if err != nil {
						log.Debug().Err(err).Msg("")
						continue
					}

					if instance.IsProtected(c) && !uninstallCmdForce {
						fmt.Printf("%s is marked protected and uses version %s, skipping\n", c, version)
					} else if !instance.IsProtected(c) || uninstallCmdForce {
						if instance.IsRunning(c) {
							restart[version] = append(restart[version], c)
						}
						continue
					}

					// none of the above, remove from list
					delete(removeReleases, version)
				}

				// directory that contains releases for this component
				// on the selected host
				basedir := h.PathTo("packages", ct.String())
				stopped := []geneos.Instance{}

				for version, release := range removeReleases {
					for _, c := range restart[version] {
						log.Debug().Msgf("stopping %s", c)
						instance.Stop(c, true, false)
						stopped = append(stopped, c)
					}
					if len(release.Links) != 0 {
						if uninstallCmdAll {
							// remove all links to this release if given --all flag
							for _, l := range release.Links {
								h.Remove(path.Join(basedir, l))
							}
						} else {
							// update to latest version, remove all others
							latest, err := geneos.LatestVersion(h, ct, "")
							if err != nil {
								log.Error().Err(err).Msg("")
								continue
							}
							updateLinks(h, basedir, release, version, latest)
						}
					}

					// remove the release
					if err = h.RemoveAll(path.Join(basedir, version)); err != nil {
						log.Error().Err(err)
						continue
					}
					fmt.Printf("removed %s release %s in %s\n", ct, version, basedir)
				}

				// restart instances previously stopped, if possible
				for _, c := range stopped {
					if err := instance.Start(c); err != nil {
						// if start fails, disable the instance
						instance.Disable(c)
						fmt.Printf("restart %s failed, disabling instance\n", c)
					}
				}
			}
		}

		return
	},
}

// updateLinks removes the base symlink for oldVersion and recreates a
// new one pointing to target. It also updates all other links in the
// map to the same old target to the new one.
func updateLinks(h *geneos.Host, releaseDir string, release geneos.ReleaseDetails, oldVersion, newVersion string) (err error) {
	for _, l := range release.Links {
		link := path.Join(releaseDir, l)
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
