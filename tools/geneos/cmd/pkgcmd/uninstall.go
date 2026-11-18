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
	"log/slog"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var uninstallCmdBase string
var uninstallCmdVersion string
var uninstallCmdAll, uninstallCmdForce, uninstallCmdKeep, uninstallCmdUpdate bool

func init() {
	packageCmd.AddCommand(uninstallCmd)

	uninstallCmd.Flags().StringVarP(&uninstallCmdBase, "base", "b", "", "Remove the base link `BASE` for matching TYPEs if no instances reference it.\nThis overrides other options below.")

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
	Use:     "uninstall [flags] [TYPE] [VERSION]",
	Short:   "Uninstall Geneos releases",
	Long:    uninstallCmdDescription,
	Aliases: []string{"delete", "remove", "rm"},
	Example: strings.ReplaceAll(`
geneos uninstall netprobe
geneos uninstall --version 7.4.2
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		log := log.With("command", "package uninstall")
		ct, args, _, err := cmd.FetchArgs(command)
		if err != nil {
			return err
		}
		h := geneos.GetHost(cmd.Hostname)

		if uninstallCmdBase != "" {
			for h := range h.OrList() {
				for ct := range ct.OrList() {
					instances := instance.Instances(h, ct, instance.MatchParameters("version="+uninstallCmdBase))
					if len(instances) > 0 {
						fmt.Printf("skipping %q on %s as it has %d instances referencing base link %q\n", ct, h, len(instances), uninstallCmdBase)
						continue
					}

					// otherwise remove the base link if it exists
					baseLink := h.PathTo("packages", ct.String(), uninstallCmdBase)

					rmErr := h.Remove(baseLink)
					if rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
						err = cmd.ReportFailure(baseLink, fmt.Errorf("cannot remove base link: %w", rmErr))
						continue
					}
					if rmErr == nil {
						fmt.Printf("removed %q on %q base link %q\n", ct, h, uninstallCmdBase)
					}
				}
			}

			return
		}

		// allow version to be the first arg unless the flag is given
		version := uninstallCmdVersion
		if version == "" && len(args) > 0 {
			version = args[0]
		}

		// all the failures are reported but the other releases, types and
		// hosts are still processed
		var failed error
		var versionMatched bool

		for h := range h.OrList() {
			for ct := range ct.OrList() {
				if len(ct.PackageTypes) > 0 {
					log.Debug("skipping as has related types, remove those instead", slog.String("ct", ct.String()))
					continue
				}

				releases, err := geneos.GetReleases(h, ct)
				if err != nil {
					failed = cmd.ReportFailure(ct.String()+"@"+h.String(), err)
					continue
				}

				// create a slice of releases to remove
				removeReleases := slices.DeleteFunc(releases, func(r *geneos.ReleaseDetails) bool {
					if uninstallCmdAll {
						return false
					}
					if version == "" && !r.Latest {
						return false
					}
					if version != "" && strings.HasPrefix(strings.TrimLeftFunc(r.Version, func(r rune) bool { return !unicode.IsNumber(r) }), version) {
						return false
					}
					return true
				})
				if len(removeReleases) > 0 {
					versionMatched = true
				}

				// loop over all instances and remove versions from a
				// list as they are found so we end up with a map
				// containing only releases to be removed
				//
				// also save a list of instances to restart
				//
				// get all instances on host h and check type and pkgtype
				restart := map[string][]geneos.Instance{}
				for _, i := range instance.Instances(h, nil) {
					pkgtype := config.Get[string](i.Config(), "pkgtype")
					if i.Type() != ct && pkgtype != ct.String() {
						i.Log().Debug("instance not same type or package type, skipping", slog.String("ct", ct.String()), slog.String("pkgtype", pkgtype))
						continue
					}
					if instance.IsDisabled(i) {
						fmt.Printf("%s is disabled, treating as an update\n", i)
						continue
					}

					_, version, err := instance.Version(i)
					if err != nil {
						i.Log().Debug("cannot get instance version", slog.Any("error", err))
						continue
					}

					// if we are not updating or told to remove "all"
					// (version filter notwithstanding) then do not
					// remove any package referenced by an instance -
					// except those disabled as above
					if !uninstallCmdUpdate && !uninstallCmdAll {
						removeReleases = slices.DeleteFunc(removeReleases, func(r *geneos.ReleaseDetails) bool { return version == r.Version })
						continue
					}

					// if we are updating and the instance is protected
					// then only update if forced
					if instance.IsProtected(i) {
						if !uninstallCmdForce {
							fmt.Printf("%s is marked protected and uses version %s, skipping\n", i, version)
							removeReleases = slices.DeleteFunc(removeReleases, func(r *geneos.ReleaseDetails) bool { return version == r.Version })
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
				removed := []*geneos.ReleaseDetails{}

				for _, release := range removeReleases {
					// stop the running instances using this release
					// before it is removed
					for _, c := range restart[release.Version] {
						c.Log().Debug("stopping")
						if err := instance.Stop(c, instance.WithStopForce(true)); err != nil && !errors.Is(err, os.ErrProcessDone) {
							failed = cmd.ReportFailure(c.String(), fmt.Errorf("cannot stop instance: %w", err))
							continue
						}
						stopped = append(stopped, c)
					}
					// remove the release
					if err := h.RemoveAll(h.Join(basedir, release.Version)); err != nil {
						failed = cmd.ReportFailure(ct.String()+" "+release.Version, fmt.Errorf("cannot remove release: %w", err))
						continue
					}
					fmt.Printf("removed %s release %s from %s:%s\n", ct, release.Version, h, basedir)
					removed = append(removed, release)

					if len(release.Links) != 0 {
						if uninstallCmdAll {
							// remove all links to this release if given --all flag
							for _, l := range release.Links {
								h.Remove(h.Join(basedir, l))
							}
						} else {
							// update to latest version, remove all others
							latest := ""
							versions, err := geneos.InstalledReleases(h, ct)
							if err != nil {
								if !errors.Is(err, fs.ErrNotExist) {
									failed = cmd.ReportFailure(ct.String(), fmt.Errorf("cannot get installed releases: %w", err))
								}
								continue
							}
							if len(versions) > 0 {
								latest = versions[len(versions)-1]
							}
							if err := updateLinks(h, ct, basedir, release, release.Version, latest); err != nil {
								failed = cmd.ReportFailure(ct.String(), err)
							}
						}
					}

				}

				// restart instances previously stopped, if possible
				for _, c := range stopped {
					if err := instance.Start(c); err != nil {
						// if start fails, disable the instance
						instance.Disable(c)
						failed = cmd.ReportFailure(c.String(), fmt.Errorf("restart failed, disabling instance: %w", err))
					}
				}

				// remove cached packages, but only locally, and only
				// those for the releases removed unless removing all
				if h.IsLocalhost() && !uninstallCmdKeep {
					if err := removeCachedDownloads(h, ct, removed, uninstallCmdAll); err != nil {
						failed = cmd.ReportFailure(ct.String(), err)
					}
				}
			}
		}

		if failed == nil && version != "" && !versionMatched {
			return fmt.Errorf("no installed release matching %q found", version)
		}
		return failed
	},
}

// removeCachedDownloads removes the cached downloads for the component
// ct on host h. If all is true then every download for the type is
// removed, otherwise only those for the given releases.
func removeCachedDownloads(h *geneos.Host, ct *geneos.Component, releases []*geneos.ReleaseDetails, all bool) (err error) {
	infix := ct.DownloadInfix
	if infix == "" {
		infix = ct.Name
	}

	var errs []error
	var patterns []string
	if all {
		patterns = append(patterns, "*"+infix+"*")
	} else {
		for _, r := range releases {
			version := strings.TrimLeftFunc(r.Version, func(r rune) bool { return !unicode.IsNumber(r) })
			patterns = append(patterns, "*"+infix+"*-"+version+"-*", "*"+infix+"*-"+version+".*")
		}
	}

	for _, p := range patterns {
		pattern := h.PathTo("packages", "downloads", p)
		files, err := h.Glob(pattern)
		if err != nil {
			errs = append(errs, fmt.Errorf("find cached packages for %s: %w", ct, err))
			continue
		}
		if len(files) == 0 && all {
			fmt.Printf("cannot find any cached downloads to remove in %q\n", pattern)
		}
		for _, f := range files {
			if rmErr := h.Remove(f); rmErr == nil {
				fmt.Printf("removed %q\n", f)
			} else {
				errs = append(errs, fmt.Errorf("cannot remove cached download %q: %w", f, rmErr))
			}
		}
	}
	return errors.Join(errs...)
}

// updateLinks removes the base symlink for oldVersion and recreates a
// new one pointing to target. It also updates all other links in the
// map to the same old target to the new one.
func updateLinks(h *geneos.Host, ct *geneos.Component, releaseDir string, release *geneos.ReleaseDetails, oldVersion, newVersion string) error {
	var errs []error
	for _, l := range release.Links {
		link := h.Join(releaseDir, l)
		if err := h.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("cannot remove link %q: %w", link, err))
			continue
		}
		if err := h.Symlink(newVersion, link); err != nil {
			errs = append(errs, fmt.Errorf("cannot create symlink %q to %q: %w", link, newVersion, err))
			continue
		}
		fmt.Printf("updated %s %s on %s, now linked to %s\n", ct, l, h, newVersion)
	}

	return errors.Join(errs...)
}
