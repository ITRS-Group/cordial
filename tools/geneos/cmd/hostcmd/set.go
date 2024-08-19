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
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var setCmdPrompt bool
var setCmdPassword *config.Plaintext
var setCmdKeyfile config.KeyFile

func init() {
	hostCmd.AddCommand(setCmd)

	setCmdPassword = &config.Plaintext{}

	setCmd.Flags().BoolVarP(&setCmdPrompt, "prompt", "p", false, "Prompt for password")
	setCmd.Flags().VarP(setCmdPassword, "password", "P", "password")
	setCmd.Flags().VarP(&setCmdKeyfile, "keyfile", "k", "Keyfile")

	setCmd.Flags().SortFlags = false
}

//go:embed _docs/set.md
var setCmdDescription string

var setCmd = &cobra.Command{
	Use:                   "set [flags] [NAME...] [KEY=VALUE...]",
	Short:                 "Set host configuration value",
	Long:                  setCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, origargs []string) (err error) {
		if len(origargs) == 0 && command.Flags().NFlag() == 0 {
			return command.Usage()
		}
		_, args, params := cmd.ParseTypeNamesParams(command)
		var password string
		var hosts []*geneos.Host

		if len(args) == 0 {
			hosts = geneos.RemoteHosts(false)
		} else {
			for _, a := range args {
				h := geneos.GetHost(a)
				if h.Exists() {
					hosts = append(hosts, h)
				}
			}
		}
		if len(hosts) == 0 {
			// nothing to do
			fmt.Println("nothing to do")
			return nil
		}

		if setCmdKeyfile == "" {
			setCmdKeyfile = cmd.DefaultUserKeyfile
		}

		// check for passwords
		if setCmdPrompt {
			if password, err = setCmdKeyfile.EncodePasswordInput(host.Localhost, true); err != nil {
				return
			}
		} else if !setCmdPassword.IsNil() {
			if password, err = setCmdKeyfile.Encode(host.Localhost, setCmdPassword, true); err != nil {
				return
			}
		}

		for _, h := range hosts {
			for _, set := range params {
				if !strings.Contains(set, "=") {
					continue
				}
				s := strings.SplitN(set, "=", 2)
				k, v := s[0], s[1]
				h.Set(k, v)
			}

			if password != "" {
				h.Set("password", password)
			}
		}

		if err = geneos.SaveHostConfig(); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		return
	},
}
