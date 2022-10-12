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
	"fmt"
	"net/url"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// addHostCmd represents the addHost command
var addHostCmd = &cobra.Command{
	Use:                   "host [-I] [NAME] [SSHURL]",
	Aliases:               []string{"remote"},
	Short:                 "Add a remote host",
	Long:                  `Add a remote host for integration with other commands.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Args:                  cobra.RangeArgs(1, 2),
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		_, args := cmdArgs(cmd)

		var h *host.Host
		sshurl, err := url.Parse(args[0])
		if err == nil && sshurl.Scheme != "" {
			h = host.Get(sshurl.Hostname())
		} else {
			h = host.Get(args[0])
			if len(args) > 1 {
				if sshurl, err = url.Parse(args[1]); err != nil {
					log.Error().Msgf("invalid ssh url %q", args[1])
					return geneos.ErrInvalidArgs
				}
			} else {
				sshurl = &url.URL{
					Scheme: "ssh",
					Host:   args[0],
				}
			}
		}

		return addHost(h, sshurl)
	},
}

func init() {
	addCmd.AddCommand(addHostCmd)

	addHostCmd.Flags().BoolVarP(&addHostCmdInit, "init", "I", false, "Initialise the remote host directories and component files")
	addHostCmd.Flags().SortFlags = false
}

var addHostCmdInit bool

func addHost(h *host.Host, sshurl *url.URL) (err error) {
	if h.Exists() {
		return fmt.Errorf("host %q already exists", h)
	}

	if sshurl == nil {
		return geneos.ErrInvalidArgs
	}

	if sshurl.Scheme != "ssh" {
		return fmt.Errorf("unsupported scheme (ssh only at the moment): %q", sshurl.Scheme)
	}

	h.SetDefault("hostname", sshurl.Hostname())
	h.SetDefault("port", 22)
	h.SetDefault("username", config.GetString("defaultuser"))
	// XXX default to remote user's home dir, not local
	h.SetDefault("geneos", host.Geneos())

	// now disassemble URL
	if sshurl.Hostname() == "" {
		h.Set("hostname", h.GetString("name"))
	}

	if sshurl.Port() != "" {
		h.Set("port", sshurl.Port())
	}

	if sshurl.User.Username() != "" {
		h.Set("username", sshurl.User.Username())
	}

	if sshurl.Path != "" {
		// XXX check and adopt local setting for remote user and/or remote global settings
		// - only if ssh URL does not contain explicit path
		h.Set("geneos", sshurl.Path)
	}

	// once we are bootstrapped, read os-release info and re-write config
	if err = h.GetOSReleaseEnv(); err != nil {
		return
	}

	host.Add(h)
	if err = host.WriteHostConfigFile(); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if addHostCmdInit {
		// initialise the remote directory structure, but perhaps ignore errors
		// as we may simply be adding an existing installation

		if err = geneos.Init(h, geneos.Force(true), geneos.LocalUsername(h.GetString("username")), geneos.Homedir(h.GetString("geneos"))); err != nil {
			return
		}
	}

	return
}
