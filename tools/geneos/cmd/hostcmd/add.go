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

package hostcmd

import (
	"fmt"
	"net/url"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var hostAddCmdInit, hostAddCmdPrompt bool
var hostAddCmdPassword string
var hostAddCmdKeyfile config.KeyFile

func init() {
	HostCmd.AddCommand(hostAddCmd)

	hostAddCmdKeyfile = cmd.DefaultUserKeyfile
	hostAddCmd.Flags().BoolVarP(&hostAddCmdInit, "init", "I", false, "Initialise the remote host directories and component files")
	hostAddCmd.Flags().BoolVarP(&hostAddCmdPrompt, "prompt", "p", false, "Prompt for password")
	hostAddCmd.Flags().StringVarP(&hostAddCmdPassword, "password", "P", "", "Password")
	hostAddCmd.Flags().VarP(&hostAddCmdKeyfile, "keyfile", "k", "Keyfile")

	hostAddCmd.Flags().SortFlags = false
}

var hostAddCmd = &cobra.Command{
	Use:   "add [flags] [NAME] [SSHURL]",
	Short: "Add a remote host",
	Long: strings.ReplaceAll(`
Add a remote host |NAME| for seamless control of your Geneos estate.

One or both of |NAME| or |SSHURL| must be given. |NAME| is used as
the default hostname if not |SSHURL| is given and, conversely, the
hostname portion of the |SSHURL| is used if no NAME is supplied.

The |SSHURL| extends the standard format by allowing a path to the
root directory for the remote Geneos installation in the format:

  ssh://[USER@]HOST[:PORT][/PATH]

Here:

|USER| is the username to be used to connect to the target host. If
is not defined, it will default to the current username.

|PORT| is the ssh port used to connect to the target host. If not
defined the default is 22.

|HOST| the hostname or IP address of the target host. Required.
  
|PATH| is the root Geneos directory used on the target host. If not
defined, it is set to the same as the local Geneos root directory.
`, "|", "`"),
	Example: strings.ReplaceAll(`
geneos host add server1
geneos host add ssh://server2:50122
geneos host add remote1 ssh://server.example.com/opt/geneos
`, "|", "`"),

	SilenceUsage: true,
	Args:         cobra.RangeArgs(1, 2),
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		_, args := cmd.CmdArgs(command)

		var h *geneos.Host
		sshurl, err := url.Parse(args[0])
		if err == nil && sshurl.Scheme != "" {
			h = geneos.GetHost(sshurl.Hostname())
		} else {
			h = geneos.GetHost(args[0])
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

		return hostAdd(h, sshurl)
	},
}

func hostAdd(h *geneos.Host, sshurl *url.URL) (err error) {
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
	u, _ := user.Current()
	h.SetDefault("username", u.Username)
	// XXX default to remote user's home dir, not local
	h.SetDefault("geneos", geneos.Root())

	password := ""

	if hostAddCmdPrompt {
		if password, err = hostAddCmdKeyfile.EncodePasswordInput(true); err != nil {
			return
		}
	} else if hostAddCmdPassword != "" {
		if password, err = hostAddCmdKeyfile.EncodeString(hostAddCmdPassword, true); err != nil {
			return
		}
	}
	if password != "" {
		h.Set("password", password)
	}

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

	if !h.IsAvailable() {
		log.Debug().Err(err).Msg("cannot connect to remote host, not adding.")
		return err
	}

	// once we are bootstrapped, read os-release info and re-write config
	if err = h.SetOSReleaseEnv(); err != nil {
		return
	}

	if sshurl.Path != "" {
		// XXX check and adopt local setting for remote user and/or remote global settings
		// - only if ssh URL does not contain explicit path
		h.Set("geneos", sshurl.Path)
	} else if runtime.GOOS != h.GetString("os") {
		homedir := h.GetString("homedir")
		if filepath.Base(homedir) != "geneos" {
			homedir = filepath.Join(homedir, "geneos")
		}
		switch h.GetString("os") {
		case "windows":
			homedir = filepath.FromSlash(homedir)
		case "linux":
			homedir = filepath.ToSlash(homedir)
		}
		h.Set("geneos", homedir)
	}

	h.Add()
	if err = geneos.SaveHostConfig(); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if hostAddCmdInit {
		// initialise the remote directory structure, but perhaps ignore errors
		// as we may simply be adding an existing installation

		if err = geneos.Init(h,
			geneos.Force(true),
			geneos.Homedir(h.GetString("geneos"))); err != nil {
			return
		}
	}

	return
}
