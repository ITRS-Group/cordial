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
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var addCmdInit, addCmdPrompt bool
var addCmdPassword *config.Plaintext
var addCmdKeyfile config.KeyFile
var addCmdPrivateKeyfiles PrivateKeyFiles

type PrivateKeyFiles []string

func (i *PrivateKeyFiles) String() string {
	return ""
}

func (i *PrivateKeyFiles) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *PrivateKeyFiles) Type() string {
	return "PATH"
}

func init() {
	hostCmd.AddCommand(addCmd)

	addCmdPassword = &config.Plaintext{}
	addCmdKeyfile = cmd.DefaultUserKeyfile
	addCmd.Flags().BoolVarP(&addCmdInit, "init", "I", false, "Initialise the remote host directories and component files")
	addCmd.Flags().BoolVarP(&addCmdPrompt, "prompt", "p", false, "Prompt for password")
	addCmd.Flags().VarP(addCmdPassword, "password", "P", "Password")
	addCmd.Flags().VarP(&addCmdKeyfile, "keyfile", "k", "Keyfile for encryption of stored password")
	addCmd.Flags().VarP(&addCmdPrivateKeyfiles, "privatekey", "i", "Private key file")

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
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var sshurl *url.URL
		var name string
		var password string
		var pw = &config.Plaintext{}

		cf := config.New()

		_, args, params := cmd.ParseTypeNamesParams(command)
		args = append(args, params...)

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

		// validate name - almost anything but no double colons, which
		// may be an issue with future IPv6 addresses
		if strings.Contains(name, "::") {
			log.Error().Msg("a remote hostname may not contain `::`")
			return geneos.ErrInvalidArgs
		}

		cf.SetDefault("hostname", sshurl.Hostname())
		cf.SetDefault("port", 22)
		// default to remote user's home dir, not local, but that can't
		// be done until after a successful connection
		cf.SetDefault(cordial.ExecutableName(), geneos.LocalRoot())

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
			crc, created, err = addCmdKeyfile.ReadOrCreate(host.Localhost)
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
				cf.Set("password", password)
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
			cf.Set("hostname", cf.GetString("name"))
		}

		if sshurl.Port() != "" {
			cf.Set("port", sshurl.Port())
		}

		if sshurl.User.Username() != "" {
			cf.Set("username", sshurl.User.Username())
		}

		if len(addCmdPrivateKeyfiles) > 0 {
			cf.Set("privatekeys", []string(addCmdPrivateKeyfiles))
		}

		h := geneos.NewHost(name,
			host.Hostname(cf.GetString("hostname")),
			host.Username(cf.GetString("username")),
			host.Port(uint16(cf.GetInt("port"))),
			host.Password(pw.Enclave),
			host.PrivateKeyFiles(cf.GetStringSlice("privatekeys")...),
		)

		h.MergeConfigMap(cf.AllSettings())

		if h.Exists() {
			return fmt.Errorf("host %q already exists", name)
		}

		var ok bool
		if ok, err = h.IsAvailable(); !ok {
			log.Debug().Err(err).Msgf("cannot connect to remote host %s port %d as %s, not adding", cf.GetString("hostname"), cf.GetInt("port"), cf.GetString("username"))
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
			h.Set(cordial.ExecutableName(), sshurl.Path)
		} else {
			geneosdir := h.GetString("homedir")
			if path.Base(geneosdir) != cordial.ExecutableName() {
				geneosdir = path.Join(geneosdir, cordial.ExecutableName())
			}
			if runtime.GOOS == "windows" {
				geneosdir = filepath.ToSlash(geneosdir)
			}
			h.Set(cordial.ExecutableName(), geneosdir)
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
				geneos.UseRoot(h.GetString(cordial.ExecutableName()))); err != nil {
				return
			}
		}
		return
	},
}
