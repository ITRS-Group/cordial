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
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var installCmdLocal, installCmdNoSave, installCmdUpdate, installCmdNexus, installCmdSnapshot bool
var installCmdBase, installCmdHost, installCmdOverride, installCmdVersion, installCmdUsername, installCmdPwFile string
var installCmdPassword []byte

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().StringVarP(&installCmdBase, "base", "b", "active_prod", "Override the base active_prod link name")

	installCmd.Flags().BoolVarP(&installCmdLocal, "local", "L", false, "Install from local files only")
	installCmd.Flags().BoolVarP(&installCmdNoSave, "nosave", "n", false, "Do not save a local copy of any downloads")
	installCmd.Flags().StringVarP(&installCmdHost, "host", "H", string(host.ALLHOSTS), "Perform on a remote host. \"all\" means all hosts and locally")

	installCmd.Flags().BoolVarP(&installCmdNexus, "nexus", "N", false, "Download from nexus.itrsgroup.com. Requires auth.")
	installCmd.Flags().BoolVarP(&installCmdSnapshot, "snapshots", "p", false, "Download from nexus snapshots (pre-releases), not releases. Requires -N")
	installCmd.Flags().StringVarP(&installCmdVersion, "version", "V", "latest", "Download this version, defaults to latest. Doesn't work for EL8 archives.")
	installCmd.Flags().StringVarP(&installCmdUsername, "username", "u", "", "Username for downloads, defaults to configuration value in download.username")
	installCmd.Flags().StringVarP(&installCmdPwFile, "pwfile", "P", "", "Password file to read for downloads, defaults to configuration value in download.password or otherwise prompts")

	installCmd.Flags().BoolVarP(&installCmdUpdate, "update", "U", false, "Update the base directory symlink")
	installCmd.Flags().StringVarP(&installCmdOverride, "override", "T", "", "Override (set) the TYPE:VERSION for archive files with non-standard names")

	installCmd.Flags().SortFlags = false
}

var installCmd = &cobra.Command{
	Use:   "install [flags] [TYPE] | FILE|URL... | [VERSION | FILTER]",
	Short: "Install (remote or local) Geneos packages",
	Long: strings.ReplaceAll(`
Installs Geneos software packages in the Geneos directory structure under
directory |packages|.  The Geneos software packages will be sourced from
the [ITRS Dowload portal](https://resources.itrsgroup.com/downloads) or,
if specified as FILE or URL, a filename formatted as
|geneos-TYPE-VERSION*.tar.gz|.

Installation will ...
- Select the latest available version, or the version specified with option
  |-V <version>|.
- Store the packages downloaded from the ITRS Download portal into 
  |packages/downloads|, unless the |-n| option is selected.
  In case a FILE or URL is specified, the FILE or URL will be used as the
  packages source and nothing will be written to |packages/downloads|.
- Plase binaries for TYPE into |packages/<TYPE>/<version>|, where
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
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, args, _ := cmdArgsParams(cmd)
		if ct == nil && len(args) == 0 && installCmdLocal {
			log.Error().Msg("install -L (local) flag with no component or file/url")
			return nil
		}

		if installCmdUsername == "" {
			installCmdUsername = config.GetString("download.username")
		}

		if installCmdPwFile != "" {
			installCmdPassword = utils.ReadPasswordFile(installCmdPwFile)
		} else {
			installCmdPassword = config.GetByteSlice("download.password")
		}

		if installCmdUsername != "" && len(installCmdPassword) == 0 {
			installCmdPassword = utils.ReadPasswordPrompt()
		}

		// if we have a component on the command line then use an archive in packages/downloads
		// or download from official web site unless -L is given. version numbers checked.
		// default to 'latest'
		//
		// overrides do not work in this case as the version and type have to be part of the
		// archive file name
		if ct != nil || len(args) == 0 {
			log.Debug().Msgf("installing %q version of %s to %s host(s)", installCmdVersion, ct, installCmdHost)

			options := []geneos.GeneosOptions{
				geneos.Version(installCmdVersion),
				geneos.Basename(installCmdBase),
				geneos.NoSave(installCmdNoSave),
				geneos.LocalOnly(installCmdLocal),
				geneos.Force(installCmdUpdate),
				geneos.OverrideVersion(installCmdOverride),
				geneos.Username(installCmdUsername),
				geneos.Password(installCmdPassword),
			}
			if installCmdNexus {
				options = append(options, geneos.UseNexus())
				if installCmdSnapshot {
					options = append(options, geneos.UseSnapshots())
				}
			}
			return install(ct, installCmdHost, options...)
		}

		// work through command line args and try to install them using the naming format
		// of standard downloads - fix versioning
		for _, source := range args {
			options := []geneos.GeneosOptions{
				geneos.Source(source),
				geneos.Basename(installCmdBase),
				geneos.NoSave(installCmdNoSave),
				geneos.LocalOnly(installCmdLocal),
				geneos.Force(installCmdUpdate),
				geneos.OverrideVersion(installCmdOverride),
				geneos.Username(installCmdUsername),
				geneos.Password(installCmdPassword),
			}
			if err = install(ct, installCmdHost, options...); err != nil {
				return err
			}
		}
		return nil
	},
}

func install(ct *geneos.Component, target string, options ...geneos.GeneosOptions) (err error) {
	for _, h := range host.Match(target) {
		if err = geneos.MakeComponentDirs(h, ct); err != nil {
			return err
		}
		if err = geneos.Install(h, ct, options...); err != nil {
			log.Error().Err(err).Msg("")
			continue
		}
	}
	return
}
