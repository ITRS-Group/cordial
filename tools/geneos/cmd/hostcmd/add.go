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
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var hostAddCmdInit, hostAddCmdPrompt bool
var hostAddCmdPassword *config.Plaintext
var hostAddCmdKeyfile config.KeyFile

func init() {
	hostCmd.AddCommand(hostAddCmd)

	hostAddCmdPassword = &config.Plaintext{}
	hostAddCmdKeyfile = cmd.DefaultUserKeyfile
	hostAddCmd.Flags().BoolVarP(&hostAddCmdInit, "init", "I", false, "Initialise the remote host directories and component files")
	hostAddCmd.Flags().BoolVarP(&hostAddCmdPrompt, "prompt", "p", false, "Prompt for password")
	hostAddCmd.Flags().VarP(hostAddCmdPassword, "password", "P", "Password")
	hostAddCmd.Flags().VarP(&hostAddCmdKeyfile, "keyfile", "k", "Keyfile")

	hostAddCmd.Flags().SortFlags = false
}

//go:embed _docs/add.md
var hostAddCmdDescription string

var hostAddCmd = &cobra.Command{
	Use:   "add [flags] [NAME] [SSHURL]",
	Short: "Add a remote host",
	Long:  hostAddCmdDescription,
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
	RunE: func(command *cobra.Command, _ []string) (err error) {
		_, args := cmd.CmdArgs(command)

		hostcf := config.New()

		var sshurl *url.URL
		var name string

		switch len(args) {
		case 1:
			if sshurl, err = url.Parse(args[0]); err != nil || sshurl.Scheme == "" {
				// build a URL from just the hostname arg
				sshurl = &url.URL{
					Scheme: "ssh",
					Host:   args[0],
				}
			}
			name = sshurl.Hostname()
		case 2:
			name = args[0]
			if sshurl, err = url.Parse(args[1]); err != nil {
				log.Error().Msgf("invalid ssh url %q", args[1])
				return
			}
		}

		// validate name - almost anything but no double colons
		if strings.Contains(name, "::") {
			log.Error().Msg("a remote hostname may not contain `::`")
			return geneos.ErrInvalidArgs
		}

		log.Debug().Msgf("hostname: %s", sshurl.Hostname())

		hostcf.SetDefault("hostname", sshurl.Hostname())
		var username string
		if u, err := user.Current(); err == nil {
			username = u.Username
		} else {
			username = os.Getenv("USER")
		}

		hostcf.SetDefault("username", username)
		hostcf.SetDefault("port", 22)
		// XXX default to remote user's home dir, not local
		hostcf.SetDefault(cmd.Execname, geneos.Root())

		var password string
		var pw = &config.Plaintext{}

		if hostAddCmdPrompt {
			pw, err = config.ReadPasswordInput(true, 3)
			if err != nil {
				return
			}
		} else if !hostAddCmdPassword.IsNil() {
			pw = hostAddCmdPassword
		}

		if !pw.IsNil() && pw.Size() > 0 {
			var crc uint32
			var created bool
			crc, created, err = hostAddCmdKeyfile.Check(true)
			if err != nil {
				return
			}
			if created {
				fmt.Printf("%s created, checksum %08X\n", hostAddCmdKeyfile, crc)
			}
			if password, err = hostAddCmdKeyfile.Encode(pw, true); err != nil {
				return
			}

			if len(password) > 0 {
				// this is the encoded password for the config file, not an enclave
				hostcf.Set("password", password)
			}
		}

		if sshurl == nil {
			return geneos.ErrInvalidArgs
		}

		if sshurl.Scheme != "ssh" {
			return fmt.Errorf("unsupported scheme %q (ssh only at the moment)", sshurl.Scheme)
		}

		// now disassemble URL
		if sshurl.Hostname() == "" {
			hostcf.Set("hostname", hostcf.GetString("name"))
		}

		if sshurl.Port() != "" {
			hostcf.Set("port", sshurl.Port())
		}

		if sshurl.User.Username() != "" {
			hostcf.Set("username", sshurl.User.Username())
		}

		var h *geneos.Host

		// creating the host initialises the remote homedir, so use it
		// as the default later...

		if sshurl.Scheme != "" {
			h = geneos.NewHost(name,
				host.Hostname(hostcf.GetString("hostname")),
				host.Username(hostcf.GetString("username")),
				host.Port(uint16(hostcf.GetInt("port"))),
				host.Password(pw.Enclave),
			)
		} else {
			h = geneos.NewHost(args[0],
				host.Hostname(args[0]),
			)
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

		h.MergeConfigMap(hostcf.AllSettings())

		if h.Exists() {
			return fmt.Errorf("host %q already exists", name)
		}

		if !h.IsAvailable() {
			log.Debug().Err(err).Msg("cannot connect to remote host, not adding.")
			return err
		}

		// once we are bootstrapped, read os-release info and re-write config
		if err = h.SetOSReleaseEnv(); err != nil {
			return
		}

		// set remote geneos dir:
		//   * if the user has set the path, use it without question
		//   * if the OS on the remote is different convert the path separators
		//   * use the user home dir with optional subdir if the last component is not the same
		if sshurl.Path != "" {
			h.Set(cmd.Execname, sshurl.Path)
		} else if runtime.GOOS != h.GetString("os") {
			geneosdir := h.GetString("homedir")
			if filepath.Base(geneosdir) != cmd.Execname {
				geneosdir = filepath.Join(geneosdir, cmd.Execname)
			}
			switch h.GetString("os") {
			case "windows":
				geneosdir = filepath.FromSlash(geneosdir)
			case "linux":
				geneosdir = filepath.ToSlash(geneosdir)
			}
			h.Set(cmd.Execname, geneosdir)
		} else {
			geneosdir := h.GetString("homedir")
			if filepath.Base(geneosdir) != cmd.Execname {
				geneosdir = filepath.Join(geneosdir, cmd.Execname)
			}
			h.Set(cmd.Execname, geneosdir)
		}

		// mark the host as valid at this point
		h.Valid()

		if err = geneos.SaveHostConfig(); err != nil {
			return err
		}

		if hostAddCmdInit {
			// initialise the remote directory structure, but perhaps ignore errors
			// as we may simply be adding an existing installation

			if err = geneos.Init(h,
				geneos.Force(true),
				geneos.UseRoot(h.GetString(cmd.Execname))); err != nil {
				return
			}
		}
		return
	},
}
