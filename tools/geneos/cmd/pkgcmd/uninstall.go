/*
Copyright © 2023 ITRS Group

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
	"path"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var uninstallCmdVersion string
var uninstallCmdAll, uninstallCmdForce, uninstallCmdKeep, uninstallCmdUpdate bool

func init() {
	packageCmd.AddCommand(uninstallCmd)

	uninstallCmd.Flags().StringVarP(&uninstallCmdVersion, "version", "V", "", "Uninstall `VERSION`")
	uninstallCmd.Flags().BoolVarP(&uninstallCmdAll, "all", "A", false, "Uninstall all releases, stopping and disabling running instances")
	uninstallCmd.Flags().BoolVarP(&uninstallCmdKeep, "keep", "k", false, "Keep cached downloads")
	uninstallCmd.Flags().BoolVarP(&uninstallCmdUpdate, "update", "U", false, "Update base links for instances to latest before restarting and removing")
	uninstallCmd.Flags().BoolVarP(&uninstallCmdForce, "force", "F", false, "Force uninstall, stopping protected instances first. Also requires --update")

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
		cmd.CmdNoneMeansAll: "false",
		cmd.CmdRequireHome:  "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, _ := cmd.ParseTypeNames(command)
		h := geneos.GetHost(cmd.Hostname)

		for _, h := range h.OrList() {
			for _, ct := range ct.OrList() {
				// remove cached packages, but only locally
				if h == geneos.LOCAL && !uninstallCmdKeep {
					pattern := ct.DownloadInfix
					if pattern == "" {
						pattern = ct.Name
					}
					files, err := filepath.Glob(h.PathTo("packages", "downloads", "*"+pattern+"*"))
					if err != nil {
						fmt.Printf("cannot find any cached downloads to remove in %q - %s\n", h.PathTo("packages", "downloads", "*"), err)
					}
					for _, f := range files {
						if err = h.Remove(f); err == nil {
							fmt.Printf("removed %q\n", f)
						} else {
							fmt.Printf("cannot remove %q - %s", f, err)
						}
					}
				}
				if len(ct.PackageTypes) > 0 {
					log.Debug().Msgf("skipping %s as has related types, remove those instead", ct)
					continue
				}

				releases, err := geneos.GetReleases(h, ct)
				if err != nil {
					return err
				}

				// save candidates for removal
				removeReleases := map[string]geneos.ReleaseDetails{}
				for _, i := range releases {
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
				instances, err := instance.Instances(h, nil)
				if err != nil {
					panic(err)
				}
				for _, i := range instances {
					if instance.IsDisabled(i) {
						fmt.Printf("%s is disabled, treating as an update\n", i)
						continue
					}

					_, version, err := instance.Version(i)
					if err != nil {
						log.Debug().Err(err).Msg("")
						continue
					}

					// if we are not updating then do not remove any
					// package referenced by an instance - except those
					// disabled as above
					if !uninstallCmdUpdate {
						delete(removeReleases, version)
						continue
					}

					// if we are updating and the instance is protected
					// then only update if forced
					if instance.IsProtected(i) {
						if !uninstallCmdForce {
							fmt.Printf("%s is marked protected and uses version %s, skipping\n", i, version)
							delete(removeReleases, version)
							continue
						}
					}

					if instance.IsRunning(i) {
						restart[version] = append(restart[version], i)
						continue
					}
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
					// remove the release
					if err = h.RemoveAll(path.Join(basedir, version)); err != nil {
						log.Error().Err(err).Msg("")
						continue
					}
					fmt.Printf("removed %s release %s from %s:%s\n", ct, version, h, basedir)

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
							updateLinks(h, ct, basedir, release, version, latest)
						}
					}

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
func updateLinks(h *geneos.Host, ct *geneos.Component, releaseDir string, release geneos.ReleaseDetails, oldVersion, newVersion string) (err error) {
	for _, l := range release.Links {
		link := path.Join(releaseDir, l)
		if err = h.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Error().Err(err).Msg("")
			continue
		}
		if err = h.Symlink(newVersion, link); err != nil {
			log.Error().Err(err).Msgf("cannot link %s to %s", link, newVersion)
			continue
		}
		fmt.Printf("updated %s %s on %s, now linked to %s\n", ct, l, h, newVersion)
	}

	return
}
