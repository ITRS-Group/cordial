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
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var packageInstallCmdLocal, packageInstallCmdNoSave, packageInstallCmdUpdate, packageInstallCmdNexus, packageInstallCmdSnapshot bool
var packageInstallCmdBase, packageInstallCmdHost, packageInstallCmdOverride, packageInstallCmdVersion, packageInstallCmdUsername, packageInstallCmdPwFile string
var packageInstallCmdPassword config.Plaintext

func init() {
	PackageCmd.AddCommand(packageInstallCmd)

	packageInstallCmd.Flags().StringVarP(&packageInstallCmdBase, "base", "b", "active_prod", "Override the base active_prod link name")

	packageInstallCmd.Flags().BoolVarP(&packageInstallCmdLocal, "local", "L", false, "Install from local files only")
	packageInstallCmd.Flags().BoolVarP(&packageInstallCmdNoSave, "nosave", "n", false, "Do not save a local copy of any downloads")
	packageInstallCmd.Flags().StringVarP(&packageInstallCmdHost, "host", "H", string(geneos.LOCALHOST), "Perform on a remote host. \"all\" means all hosts and locally")

	packageInstallCmd.Flags().BoolVarP(&packageInstallCmdNexus, "nexus", "N", false, "Download from nexus.itrsgroup.com. Requires auth.")
	packageInstallCmd.Flags().BoolVarP(&packageInstallCmdSnapshot, "snapshots", "p", false, "Download from nexus snapshots (pre-releases), not releases. Requires -N")
	packageInstallCmd.Flags().StringVarP(&packageInstallCmdVersion, "version", "V", "latest", "Download this version, defaults to latest. Doesn't work for EL8 archives.")
	packageInstallCmd.Flags().StringVarP(&packageInstallCmdUsername, "username", "u", "", "Username for downloads, defaults to configuration value in download.username")
	packageInstallCmd.Flags().StringVarP(&packageInstallCmdPwFile, "pwfile", "P", "", "Password file to read for downloads, defaults to configuration value in download.password or otherwise prompts")

	packageInstallCmd.Flags().BoolVarP(&packageInstallCmdUpdate, "update", "U", false, "Update the base directory symlink")
	packageInstallCmd.Flags().StringVarP(&packageInstallCmdOverride, "override", "T", "", "Override (set) the TYPE:VERSION for archive files with non-standard names")

	packageInstallCmd.Flags().SortFlags = false
}

var packageInstallCmd = &cobra.Command{
	Use:   "install [flags] [TYPE] [FILE|URL...]",
	Short: "Install Geneos releases",
	Long: strings.ReplaceAll(`
Installs Geneos software packages in the Geneos directory structure under
directory |packages|. The Geneos software packages will be sourced from
the [ITRS Download portal](https://resources.itrsgroup.com/downloads) or,
if specified as FILE or URL, a filename formatted as
|geneos-TYPE-VERSION*.tar.gz|.

Installation will ...
- Select the latest available version, or the version specified with option
  |-V <version>|.
- Store the packages downloaded from the ITRS Download portal into 
  |packages/downloads|, unless the |-n| option is selected.
  In case a FILE or URL is specified, the FILE or URL will be used as the
  packages source and nothing will be written to |packages/downloads|.
- Place binaries for TYPE into |packages/<TYPE>/<version>|, where
  <version> is the version number of the package and can be overridden by
  using option |-T|.
- in case no symlink pointing to the default version exists, one will be
  created as |active_prod| or using the name provided with option 
  |-b <symlink_name>|.
  **Note**: Option |-b <symlink_name>| may be used in conjunction with |-U|.
- If option |-U| is used, the symlink will be updated.
  If this is used, instances of the binary will be stopped and restarted
  after the link has been updated.

  The |geneos install| command works for the following component types:
  |licd| (license daemon), |gateway|, |netprobe|, |webserver| (webserver for 
  web dashboards), |fa2| (fix analyser netprobe), |fileagent| (file agent for 
  fix analyser).
`, "|", "`"),
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
		if ct == nil && len(args) == 0 && packageInstallCmdLocal {
			log.Error().Msg("install -L (local) flag with no component or file/url")
			return nil
		}

		for _, p := range params {
			if strings.HasPrefix(p, "@") {
				return fmt.Errorf("@HOST not valid here, perhaps you meant `-H HOST`?")
			}
		}

		if packageInstallCmdUsername == "" {
			packageInstallCmdUsername = config.GetString(config.Join("download", "username"))
		}

		if packageInstallCmdPwFile != "" {
			var pp []byte
			if pp, err = os.ReadFile(packageInstallCmdPwFile); err != nil {
				return
			}
			packageInstallCmdPassword = config.NewPlaintext(pp)
		} else {
			packageInstallCmdPassword = config.GetPassword(config.Join("download", "password"))
		}

		if packageInstallCmdUsername != "" && (packageInstallCmdPassword.IsNil() || packageInstallCmdPassword.Size() == 0) {
			packageInstallCmdPassword, _ = config.ReadPasswordInput(false, 0)
		}

		// if we have a component on the command line then use an archive in packages/downloads
		// or download from official web site unless -L is given. version numbers checked.
		// default to 'latest'
		//
		// overrides do not work in this case as the version and type have to be part of the
		// archive file name
		if ct != nil || len(args) == 0 {
			log.Debug().Msgf("installing %q version of %s to %s host(s)", packageInstallCmdVersion, ct, packageInstallCmdHost)

			options := []geneos.Options{
				geneos.Version(packageInstallCmdVersion),
				geneos.Basename(packageInstallCmdBase),
				geneos.NoSave(packageInstallCmdNoSave),
				geneos.LocalOnly(packageInstallCmdLocal),
				geneos.DoUpdate(packageInstallCmdUpdate),
				geneos.Force(packageInstallCmdUpdate),
				geneos.OverrideVersion(packageInstallCmdOverride),
				geneos.Username(packageInstallCmdUsername),
				geneos.Password(packageInstallCmdPassword),
			}
			if packageInstallCmdNexus {
				options = append(options, geneos.UseNexus())
				if packageInstallCmdSnapshot {
					options = append(options, geneos.UseSnapshots())
				}
			}
			return install(ct, packageInstallCmdHost, options...)
		}

		// work through command line args and try to install them using the naming format
		// of standard downloads - fix versioning
		for _, source := range args {
			options := []geneos.Options{
				geneos.Source(source),
				geneos.Basename(packageInstallCmdBase),
				geneos.NoSave(packageInstallCmdNoSave),
				geneos.LocalOnly(packageInstallCmdLocal),
				geneos.DoUpdate(packageInstallCmdUpdate),
				geneos.Force(packageInstallCmdUpdate),
				geneos.OverrideVersion(packageInstallCmdOverride),
				geneos.Username(packageInstallCmdUsername),
				geneos.Password(packageInstallCmdPassword),
			}
			if err = install(ct, packageInstallCmdHost, options...); err != nil {
				return err
			}
		}
		return nil
	},
}

func install(ct *geneos.Component, target string, options ...geneos.Options) (err error) {
	for _, h := range geneos.Match(target) {
		if err = ct.MakeComponentDirs(h); err != nil {
			return err
		}
		if err = geneos.Install(h, ct, options...); err != nil {
			return
		}
	}
	return
}
