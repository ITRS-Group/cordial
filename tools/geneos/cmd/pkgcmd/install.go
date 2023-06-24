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
)

var installCmdLocal, installCmdNoSave, installCmdUpdate, installCmdNexus, installCmdSnapshot bool
var installCmdBase, installCmdOverride, installCmdVersion, installCmdUsername, installCmdPwFile string
var installCmdPassword *config.Plaintext

func init() {
	packageCmd.AddCommand(installCmd)

	installCmdPassword = &config.Plaintext{}

	installCmd.Flags().StringVarP(&installCmdUsername, "username", "u", "", "Username for downloads, defaults to configuration value in download.username")
	installCmd.Flags().StringVarP(&installCmdPwFile, "pwfile", "P", "", "Password file to read for downloads, defaults to configuration value in download.password or otherwise prompts")

	installCmd.Flags().BoolVarP(&installCmdLocal, "local", "L", false, "Install from local files only")
	installCmd.Flags().BoolVarP(&installCmdNoSave, "nosave", "n", false, "Do not save a local copy of any downloads")

	installCmd.Flags().BoolVarP(&installCmdUpdate, "update", "U", false, "Update the base directory symlink")
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
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
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

		// base options
		options := []geneos.Options{
			geneos.Basename(installCmdBase),
			geneos.DoUpdate(installCmdUpdate),
			geneos.Force(installCmdUpdate),
			geneos.LocalOnly(installCmdLocal),
			geneos.NoSave(installCmdNoSave),
			geneos.OverrideVersion(installCmdOverride),
			geneos.Password(installCmdPassword),
			geneos.Username(installCmdUsername),
		}

		// if we have a component on the command line then use an archive in packages/downloads
		// or download from official web site unless -L is given. version numbers checked.
		// default to 'latest'
		//
		// overrides do not work in this case as the version and type have to be part of the
		// archive file name
		if ct != nil || len(args) == 0 {
			log.Debug().Msgf("installing %q version of %s to %s host(s)", installCmdVersion, ct, cmd.Hostname)

			options = append(options, geneos.Version(installCmdVersion))
			if installCmdSnapshot {
				installCmdNexus = true
				options = append(options, geneos.UseSnapshots())
			}
			if installCmdNexus {
				options = append(options, geneos.UseNexus())
			}
			err = install(h, ct, options...)
			return err
		}

		// work through command line args and try to install each
		// argument using the naming format of standard downloads
		for _, source := range args {
			o := append(options, geneos.Source(source))
			if err = install(h, ct, o...); err != nil {
				return err
			}
		}
		return nil
	},
}

func install(h *geneos.Host, ct *geneos.Component, options ...geneos.Options) (err error) {
	for _, h := range h.OrList() {
		if err = ct.MakeComponentDirs(h); err != nil {
			return err
		}
		for _, ct := range ct.OrList(geneos.RealComponents()...) {
			if err = geneos.Install(h, ct, options...); err != nil {
				if errors.Is(err, fs.ErrExist) {
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
