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

package hostcmd

import (
	_ "embed"
	"fmt"
	"net/url"
	"path"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var addCmdInit, addCmdPrompt bool
var addCmdPassword *config.Plaintext
var addCmdKeyfile config.KeyFile

func init() {
	hostCmd.AddCommand(addCmd)

	addCmdPassword = &config.Plaintext{}
	addCmdKeyfile = cmd.DefaultUserKeyfile
	addCmd.Flags().BoolVarP(&addCmdInit, "init", "I", false, "Initialise the remote host directories and component files")
	addCmd.Flags().BoolVarP(&addCmdPrompt, "prompt", "p", false, "Prompt for password")
	addCmd.Flags().VarP(addCmdPassword, "password", "P", "Password")
	addCmd.Flags().VarP(&addCmdKeyfile, "keyfile", "k", "Keyfile")

	addCmd.Flags().SortFlags = false
}

//go:embed _docs/add.md
var addCmdDescription string

var addCmd = &cobra.Command{
	Use:   "add [flags] [NAME] [SSHURL]",
	Short: "Add a remote host",
	Long:  addCmdDescription,
	Example: strings.ReplaceAll(`
geneos host add server1
geneos host add ssh://server2:50122
geneos host add remote1 ssh://server.example.com/opt/geneos
`, "|", "`"),

	SilenceUsage: true,
	Args:         cobra.RangeArgs(1, 2),
	Annotations: map[string]string{
		cmd.CmdNoneMeansAll: "false",
		cmd.CmdRequireHome:  "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		_, args, params := cmd.ParseTypeNamesParams(command)
		args = append(args, params...)

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
		default:
			log.Fatal().Msgf("wrong number of args: %d", len(args))
		}

		// validate name - almost anything but no double colons
		if strings.Contains(name, "::") {
			log.Error().Msg("a remote hostname may not contain `::`")
			return geneos.ErrInvalidArgs
		}

		log.Debug().Msgf("hostname: %s", sshurl.Hostname())

		hostcf.SetDefault("hostname", sshurl.Hostname())

		hostcf.SetDefault("port", 22)
		// XXX default to remote user's home dir, not local
		hostcf.SetDefault(cmd.Execname, geneos.LocalRoot())

		var password string
		var pw = &config.Plaintext{}

		if addCmdPrompt {
			pw, err = config.ReadPasswordInput(true, 3)
			if err != nil {
				return
			}
		} else if !addCmdPassword.IsNil() {
			pw = addCmdPassword
		}

		if !pw.IsNil() && pw.Size() > 0 {
			var crc uint32
			var created bool
			crc, created, err = addCmdKeyfile.ReadOrCreate(host.Localhost, true)
			if err != nil {
				return
			}
			if created {
				fmt.Printf("%s created, checksum %08X\n", addCmdKeyfile, crc)
			}
			if password, err = addCmdKeyfile.Encode(host.Localhost, pw, true); err != nil {
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

		h := geneos.NewHost(name,
			host.Hostname(hostcf.GetString("hostname")),
			host.Username(hostcf.GetString("username")),
			host.Port(uint16(hostcf.GetInt("port"))),
			host.Password(pw.Enclave),
		)

		h.MergeConfigMap(hostcf.AllSettings())

		if h.Exists() {
			return fmt.Errorf("host %q already exists", name)
		}

		var ok bool
		if ok, err = h.IsAvailable(); !ok {
			log.Debug().Err(err).Msg("cannot connect to remote host, not adding.")
			return
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
			if path.Base(geneosdir) != cmd.Execname {
				geneosdir = path.Join(geneosdir, cmd.Execname)
			}
			// switch h.GetString("os") {
			// case "windows":
			// 	geneosdir = filepath.FromSlash(geneosdir)
			// case "linux":
			// 	geneosdir = filepath.ToSlash(geneosdir)
			// }
			h.Set(cmd.Execname, geneosdir)
		} else {
			geneosdir := h.GetString("homedir")
			if path.Base(geneosdir) != cmd.Execname {
				geneosdir = path.Join(geneosdir, cmd.Execname)
			}
			h.Set(cmd.Execname, geneosdir)
		}

		// mark the host as valid at this point
		h.Valid()

		if err = geneos.SaveHostConfig(); err != nil {
			return err
		}

		if addCmdInit {
			// initialise the remote directory structure, but perhaps ignore errors
			// as we may simply be adding an existing installation

			if err = geneos.Initialise(h,
				geneos.Force(true),
				geneos.UseRoot(h.GetString(cmd.Execname))); err != nil {
				return
			}
		}
		return
	},
}
