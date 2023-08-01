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

package pkgcmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var installCmdLocal, installCmdNoSave, installCmdUpdate, installCmdForce, installCmdNexus, installCmdSnapshot bool
var installCmdBase, installCmdOverride, installCmdVersion, installCmdUsername, installCmdPwFile string
var installCmdDownloadOnly bool
var installCmdPassword *config.Plaintext

func init() {
	packageCmd.AddCommand(installCmd)

	installCmdPassword = &config.Plaintext{}

	installCmd.Flags().StringVarP(&installCmdUsername, "username", "u", "", "Username for downloads, defaults to configuration value in download.username")
	installCmd.Flags().StringVarP(&installCmdPwFile, "pwfile", "P", "", "Password file to read for downloads, defaults to configuration value in download::password or otherwise prompts")

	installCmd.Flags().BoolVarP(&installCmdLocal, "local", "L", false, "Install from local files only")
	installCmd.Flags().BoolVarP(&installCmdNoSave, "nosave", "n", false, "Do not save a local copy of any downloads")
	installCmd.Flags().BoolVarP(&installCmdDownloadOnly, "download", "D", false, "Download only")

	installCmd.Flags().BoolVarP(&installCmdUpdate, "update", "U", false, "Update the base directory symlink, will restart unprotected instances")
	installCmd.Flags().BoolVarP(&installCmdForce, "force", "F", false, "Will also restart protected instances, implies --update")

	installCmd.Flags().StringVarP(&installCmdBase, "base", "b", "active_prod", "Override the base active_prod link name")

	installCmd.Flags().StringVarP(&installCmdVersion, "version", "V", "latest", "Download this version, defaults to latest. Doesn't work for EL8 archives.")
	installCmd.Flags().StringVarP(&installCmdOverride, "override", "T", "", "Override (set) the TYPE:VERSION for archive files with non-standard names")

	installCmd.Flags().BoolVarP(&installCmdNexus, "nexus", "N", false, "Download from nexus.itrsgroup.com. Requires auth.")
	installCmd.Flags().BoolVarP(&installCmdSnapshot, "snapshots", "S", false, "Download from nexus snapshots (pre-releases), not releases. Requires -N")

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
geneos install fa2 5.5 -U
geneos install netprobe -b active_dev -U
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		if installCmdDownloadOnly {
			if installCmdLocal || installCmdBase != "active_prod" || installCmdUpdate || installCmdNoSave || installCmdOverride != "" {
				return errors.New("flag --download/-D set with other incompatible options")
			}
			// force localhost
			cmd.Hostname = geneos.LOCALHOST
		} else {
			if geneos.Root() == "" {
				command.SetUsageTemplate(" ")
				return cmd.GeneosUnsetError
			}
		}

		ct, args, params := cmd.CmdArgsParams(command)

		for _, p := range params {
			if strings.HasPrefix(p, "@") {
				return fmt.Errorf("@HOST not valid here, perhaps you meant `-H HOST`?")
			}
		}

		h := geneos.GetHost(cmd.Hostname)

		if installCmdUsername == "" {
			installCmdUsername = config.GetString(config.Join("download", "username"))
		}

		if installCmdPwFile != "" {
			var pp []byte
			if pp, err = os.ReadFile(installCmdPwFile); err != nil {
				return
			}
			installCmdPassword = config.NewPlaintext(pp)
		} else {
			installCmdPassword = config.GetPassword(config.Join("download", "password"))
		}

		if installCmdUsername != "" && (installCmdPassword.IsNil() || installCmdPassword.Size() == 0) {
			installCmdPassword, err = config.ReadPasswordInput(false, 0)
			if err == config.ErrNotInteractive {
				err = fmt.Errorf("%w and password required", err)
				return
			}
		}

		if installCmdForce {
			installCmdUpdate = true
		}

		// base options
		options := []geneos.Options{
			geneos.Basename(installCmdBase),
			geneos.DoUpdate(installCmdUpdate),
			geneos.Force(installCmdForce),
			geneos.LocalOnly(installCmdLocal),
			geneos.NoSave(installCmdNoSave),
			geneos.Version(installCmdVersion),
			geneos.OverrideVersion(installCmdOverride),
			geneos.Password(installCmdPassword),
			geneos.Username(installCmdUsername),
			geneos.DownloadOnly(installCmdDownloadOnly),
		}

		if installCmdDownloadOnly {
			archive := "."
			if len(args) > 0 {
				archive = args[0]
			} else if len(params) > 0 {
				archive = params[0]
			}
			log.Debug().Msgf("downloading %q version of %s to %s", installCmdVersion, ct, archive)
			options = append(options,
				geneos.Archive(archive),
			)
			if installCmdSnapshot {
				installCmdNexus = true
				options = append(options, geneos.UseSnapshots())
			}
			if installCmdNexus {
				options = append(options, geneos.UseNexus())
			}
			return install(h, ct, options...)
		}

		cs := instance.ByKeyValue(h, ct, "protected", "true")
		if len(cs) > 0 && installCmdUpdate && !installCmdForce {
			fmt.Println("There are one or more protected instances using the current version. Use `--force` to override")
			return
		}

		// stop instances early. once we get to components, we don't know about instances
		if installCmdUpdate {
			instances := instance.ByKeyValue(h, ct, "version", installCmdBase)
			for _, c := range instances {
				if err = instance.Stop(c, installCmdForce, false); err == nil {
					// only restart instances that we stopped, regardless of success of install/update
					defer instance.Start(c)
				}
			}
		}

		args = append(args, params...)

		// if we have a component on the command line then use an archive in packages/downloads
		// or download from official web site unless -L is given. version numbers checked.
		// default to 'latest'
		//
		// overrides do not work in this case as the version and type have to be part of the
		// archive file name
		if ct != nil || len(args) == 0 {
			log.Debug().Msgf("installing %q version of %s to %s host(s)", installCmdVersion, ct, cmd.Hostname)

			if installCmdSnapshot {
				installCmdNexus = true
				options = append(options, geneos.UseSnapshots())
			}
			if installCmdNexus {
				options = append(options, geneos.UseNexus())
			}
			return install(h, ct, options...)
		}

		// work through command line args and try to install each
		// argument using the naming format of standard downloads
		for _, source := range args {
			if err = install(h, ct, append(options, geneos.Archive(source))...); err != nil {
				return err
			}
		}
		return nil
	},
}

func install(h *geneos.Host, ct *geneos.Component, options ...geneos.Options) (err error) {
	for _, h := range h.OrList() {
		if err = ct.MakeDirs(h); err != nil {
			return err
		}
		for _, ct := range ct.OrList(geneos.RealComponents()...) {
			if err = geneos.Install(h, ct, options...); err != nil {
				if errors.Is(err, fs.ErrExist) {
					fmt.Printf("%s installation already exists, skipping", ct)
					err = nil
					continue
				}
				if errors.Is(err, fs.ErrNotExist) && installCmdVersion != "latest" {
					err = nil
					continue
				}
				return err
			}
		}
	}
	return
}
