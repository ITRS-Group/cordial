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
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var installCmdLocal, installCmdNoSave, installCmdUpdate, installCmdForce, installCmdNexus, installCmdSnapshot bool
var installCmdBase, installCmdOverride, installCmdVersion, installCmdUsername string
var installCmdDownloadOnly, installCmdAllTypes bool
var installCmdPassword *config.Plaintext

func init() {
	packageCmd.AddCommand(installCmd)

	installCmdPassword = &config.Plaintext{}

	installCmd.Flags().StringVarP(&installCmdUsername, "username", "u", "", "Username for downloads, defaults to configuration value in download.username")

	installCmd.Flags().BoolVarP(&installCmdLocal, "local", "L", false, "Install from local files only.\n* Implied if files or directories are given on command line.")
	installCmd.Flags().BoolVarP(&installCmdNoSave, "nosave", "n", false, "Do not save a local copy of any downloads.\n* Implied if files or directories are given on command line.")
	installCmd.Flags().BoolVarP(&installCmdDownloadOnly, "download", "D", false, "Download only, to the current directory.\nCannot be combined with --nosave")

	// note that "local" and "nosave" together are fine. illogical, but fine.
	installCmd.MarkFlagsMutuallyExclusive("nosave", "download")
	installCmd.MarkFlagsMutuallyExclusive("local", "download")

	installCmd.Flags().BoolVarP(&installCmdUpdate, "update", "U", false, "Update the base directory symlink, will restart unprotected instances")
	installCmd.Flags().BoolVarP(&installCmdForce, "force", "F", false, "Force restart of protected instances, implies --update")

	installCmd.Flags().StringVarP(&installCmdBase, "base", "b", "active_prod", "Override the base active_prod link name")

	installCmd.Flags().StringVarP(&installCmdVersion, "version", "V", "latest", "Download this version, defaults to latest. Doesn't work for EL8 archives.\nIgnored if local file(s) - not directories - are given to install on command-line")
	installCmd.Flags().StringVarP(&installCmdOverride, "override", "O", "", "Override the TYPE:VERSION for archive files with non-standard names")
	installCmd.MarkFlagsMutuallyExclusive("version", "override")

	installCmd.Flags().BoolVarP(&installCmdNexus, "nexus", "N", false, "Download from nexus.itrsgroup.com. Requires auth.")
	installCmd.Flags().BoolVarP(&installCmdSnapshot, "snapshots", "S", false, "Download from nexus snapshots (pre-releases), not releases. Requires -N")

	installCmd.Flags().BoolVarP(&installCmdAllTypes, "all", "A", false, "Install all types available, not just those types already installed")

	installCmd.Flags().SortFlags = false
}

//go:embed _docs/install.md
var installCmdDescription string

var installCmd = &cobra.Command{
	Use:   "install [flags] [TYPE] [FILE|URL...]",
	Short: "Install Geneos releases",
	Long:  installCmdDescription,
	Example: strings.ReplaceAll(`
geneos install gateway
geneos install fa2 -V 6.5 -U
geneos install netprobe -b active_dev -U
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		if installCmdDownloadOnly {
			if installCmdLocal || installCmdBase != "active_prod" || installCmdUpdate || installCmdNoSave || installCmdOverride != "" {
				return errors.New("flag --download/-D set with other incompatible options")
			}
			// force localhost
			cmd.Hostname = geneos.LOCALHOST
		} else {
			if geneos.LocalRoot() == "" {
				command.SetUsageTemplate(" ")
				return cmd.GeneosUnsetError
			}
		}

		ct, args, params := cmd.ParseTypeNamesParams(command)

		log.Debug().Msgf("args %v params: %v", args, params)
		params = append(args, params...)
		for _, p := range params {
			if strings.HasPrefix(p, "@") {
				return fmt.Errorf("the @HOST format is not valid here, perhaps you meant `-H HOST`?")
			}
		}

		h := geneos.GetHost(cmd.Hostname)

		// if params contains files and not directories then those
		// override/filter component types, so resolve those first
		//
		// if params contains directories then filter contents using
		// given ct, if version if given then filter on that
		if len(params) > 0 {
			log.Debug().Msg("parameters found, local install only")
			if installCmdOverride != "" {
				if len(params) > 1 {
					return fmt.Errorf("`--override` is only valid with a single file parameter")
				}

				st, err := h.Stat(params[0])
				if err != nil {
					return fmt.Errorf("%w: `--override` only valid with a single file parameter", err)
				}

				if st.IsDir() {
					return fmt.Errorf("%s is a directory: `--override` only valid with a single file parameter", params[0])
				}
			}

			// use ct as a filter, if set

			// base options
			options := []geneos.PackageOptions{
				geneos.Basename(installCmdBase),
				geneos.DoUpdate(installCmdUpdate || installCmdForce),
				geneos.Force(installCmdForce),
				geneos.LocalOnly(true),
				geneos.NoSave(true),
				geneos.OverrideVersion(installCmdOverride),
			}

			var installed bool
			// work through command line params and try to install each
			// argument using the naming format of standard downloads
			for _, source := range params {
				nct := ct
				version := installCmdVersion

				if installCmdOverride != "" {
					nct, version, err = geneos.OverrideToComponentVersion(installCmdOverride)
					if err != nil {
						fmt.Printf("cannot user override: %s\n", err)
						return nil
					}
				}

				switch {
				case geneos.IsURL(source):
					u, err := url.Parse(source)
					if err != nil {
						log.Debug().Err(err).Msg("skipping")
						continue
					}
					// just take the last part of the path
					p := path.Base(u.Path)
					if installCmdOverride == "" {
						var platform string
						nct, version, platform, _, err = geneos.FilenameToComponentVersion(ct, path.Base(p))
						if err != nil {
							log.Debug().Err(err).Msg("skipping")
							continue
						}
						if platform != "" {
							version += "+" + platform
						}
					}
				case geneos.IsDir(source):
					// use as is
					break
				case geneos.IsFile(source):
					p, err := filepath.EvalSymlinks(source)
					if err != nil {
						log.Debug().Err(err).Msg("skipping")
						continue
					}
					if installCmdOverride == "" {
						var platform string
						nct, version, platform, _, err = geneos.FilenameToComponentVersion(ct, path.Base(p))
						if err != nil {
							log.Debug().Err(err).Msg("skipping")
							continue
						}
						if platform != "" {
							version += "+" + platform
						}
					}

					if ct != nil && ct != nct {
						log.Debug().Msgf("ct %s and file ct %s do not match, skipping", ct, nct)
						continue
					}
				default:
					// if none of the above, skip
					log.Debug().Msg("skipping")
				}

				options = append(options, geneos.Version(version))

				log.Debug().Msgf("installing from %s as %q version of %s to %s host(s)", source, version, nct, cmd.Hostname)
				if err = Install(h, nct, append(options, geneos.Source(source))...); err != nil {
					return err
				}
				installed = true
			}
			if !installed {
				return fmt.Errorf("no matching archives found")
			}
			return nil
		}

		// don't look for credentials if we are only installing fom local sources
		if !installCmdLocal {
			if installCmdUsername == "" {
				installCmdUsername = config.GetString(config.Join("download", "username"))
			}

			installCmdPassword = config.GetPassword(config.Join("download", "password"))

			if installCmdUsername != "" && (installCmdPassword.IsNil() || installCmdPassword.Size() == 0) {
				installCmdPassword, err = config.ReadPasswordInput(false, 0)
				if err == config.ErrNotInteractive {
					err = fmt.Errorf("%w and password required", err)
					return
				}
			}
		}

		if installCmdForce {
			installCmdUpdate = true
		}

		// base options
		options := []geneos.PackageOptions{
			geneos.Basename(installCmdBase),
			geneos.DoUpdate(installCmdUpdate || installCmdForce),
			geneos.Force(installCmdForce),
			geneos.LocalOnly(installCmdLocal),
			geneos.NoSave(installCmdNoSave || installCmdLocal),
			geneos.Version(installCmdVersion),
			geneos.OverrideVersion(installCmdOverride),
			geneos.Password(installCmdPassword),
			geneos.Username(installCmdUsername),
			geneos.DownloadOnly(installCmdDownloadOnly),
		}

		if installCmdDownloadOnly {
			archive := "."
			if len(params) > 0 {
				archive = params[0]
			}
			log.Debug().Msgf("downloading %q version of %s to %s", installCmdVersion, ct, archive)
			options = append(options,
				geneos.Source(archive),
			)
			if installCmdSnapshot {
				installCmdNexus = true
				log.Debug().Msg("setting nexus snapshots")
				options = append(options, geneos.UseNexusSnapshots())
			}
			if installCmdNexus {
				log.Debug().Msg("setting nexus")
				options = append(options, geneos.UseNexus())
			}
			return Install(h, ct, options...)
		}

		if len(instance.Instances(h, ct, instance.FilterParameters("protected=true", "version="+installCmdBase))) > 0 && installCmdUpdate && !installCmdForce {
			fmt.Println("There are one or more protected instances using the current version. Use `--force` to override")
			return
		}

		// record which instances to stop early. once we get to components, we don't know about instances
		if installCmdUpdate {
			instances := []geneos.Instance{}
			for ct := range ct.OrList() {
				for _, i := range instance.Instances(h, ct) {
					if i.Config().GetString("version") != installCmdBase {
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
			options = append(options,
				geneos.Restart(instances...),
				geneos.StartFunc(instance.Start),
				geneos.StopFunc(instance.Stop))
		}

		log.Debug().Msgf("installing %q version of %s to %s host(s)", installCmdVersion, ct, cmd.Hostname)

		if installCmdSnapshot {
			installCmdNexus = true
			options = append(options, geneos.UseNexusSnapshots())
		}
		if installCmdNexus {
			options = append(options, geneos.UseNexus())
		}

		return Install(h, ct, options...)
	},
}

// Install installs the given component type on the given host, using the
// given options. If ct is nil, then all component types are installed.
func Install(h *geneos.Host, ct *geneos.Component, options ...geneos.PackageOptions) (err error) {
	installed := 0
	ctSet := ct != nil

	for h := range h.OrList() {
		for ct := range ct.OrList() {
			if !ctSet && !installCmdAllTypes {
				v, err := geneos.GetReleases(h, ct)
				if err != nil {
					return err
				}
				if len(v) == 0 {
					continue
				}
			}
			if err = ct.MakeDirs(h); err != nil {
				return err
			}
			if err = geneos.Install(h, ct, options...); err != nil {
				if errors.Is(err, fs.ErrExist) {
					fmt.Printf("%s installation already exists, skipping", ct)
					err = nil
					installed++
					continue
				}
				if errors.Is(err, fs.ErrNotExist) && installCmdVersion != "latest" {
					err = nil
					installed++
					continue
				}
				return err
			}
			installed++
		}
	}
	if installed == 0 {
		return fmt.Errorf("no matching release installed")
	}
	return
}
