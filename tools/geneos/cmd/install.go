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
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [-b BASENAME] [-l] [-n] [-H HOST] [-U] [-T TYPE:VERSION] [TYPE] | FILE|URL [FILE|URL...] | [VERSION | FILTER]",
	Short: "Install files from downloaded Geneos packages. Intended for sites without Internet access",
	Long: `Installs files from FILE(s) in to the packages/ directory. The filename(s) must of of the form:

	geneos-TYPE-VERSION*.tar.gz

The directory for the package is created using the VERSION from the archive
filename unless overridden by the -T and -V flags.

If a TYPE is given then the latest version from the packages/downloads
directory for that TYPE is installed, otherwise it is treated as a
normal file path. This is primarily for installing to remote locations.

TODO:

Install only changes creates a base link if one does not exist.
To update an existing base link use the -U option. This stops any
instance, updates the link and starts the instance up again.

Use the update command to explicitly change the base link after installation.

Use the -b flag to change the base link name from the default 'active_prod'. This also
applies when using -U.

"geneos install gateway"
"geneos install fa2 5.5 -U"
"geneos install netprobe -b active_dev -U"
"geneos update gateway -b active_prod"

`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandInstall(ct, args, params)
	},
}

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

var installCmdLocal, installCmdNoSave, installCmdUpdate, installCmdNexus, installCmdSnapshot bool
var installCmdBase, installCmdHost, installCmdOverride, installCmdVersion, installCmdUsername, installCmdPassword, installCmdPwFile string

func commandInstall(ct *geneos.Component, args, params []string) (err error) {
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
		installCmdPassword = config.GetString("download.password")
	}

	if installCmdUsername != "" && installCmdPassword == "" {
		installCmdPassword = utils.ReadPasswordPrompt()
	}

	// if we have a component on the command line then use an archive in packages/downloads
	// or download from official web site unless -l is given. version numbers checked.
	// default to 'latest'
	//
	// overrides do not work in this case as the version and type have to be part of the
	// archive file name
	if ct != nil || len(args) == 0 {
		log.Debug().Msgf("installing %q version of %s to %s host(s)", installCmdVersion, ct, installCmdHost)

		options := []geneos.GeneosOptions{geneos.Version(installCmdVersion), geneos.Basename(installCmdBase), geneos.Force(installCmdUpdate), geneos.OverrideVersion(installCmdOverride), geneos.Username(installCmdUsername), geneos.Password(installCmdPassword)}
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
	for _, file := range args {
		options := []geneos.GeneosOptions{geneos.Filename(file), geneos.Basename(installCmdBase), geneos.Force(installCmdUpdate), geneos.OverrideVersion(installCmdOverride), geneos.Username(installCmdUsername), geneos.Password(installCmdPassword)}
		if err = install(ct, installCmdHost, options...); err != nil {
			return err
		}
	}
	return nil
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
