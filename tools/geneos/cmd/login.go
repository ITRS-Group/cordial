/*
Copyright © 2023 ITRS Group

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

package cmd

import (
	_ "embed"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

var loginCmdUsername string
var loginCmdPassword *config.Plaintext
var loginKeyfile config.KeyFile
var loginCmdList bool

func init() {
	GeneosCmd.AddCommand(loginCmd)

	loginCmdPassword = &config.Plaintext{}

	loginCmd.Flags().StringVarP(&loginCmdUsername, "username", "u", "", "Username")
	loginCmd.Flags().VarP(loginCmdPassword, "password", "p", "Password")
	loginCmd.Flags().VarP(&loginKeyfile, "keyfile", "k", "Key file to use")
	loginCmd.Flags().BoolVarP(&loginCmdList, "list", "l", false, "List the names of the currently stored credentials")

	loginCmd.Flags().SortFlags = false

}

//go:embed _docs/login.md
var loginCmdDescription string

var loginCmd = &cobra.Command{
	Use:          "login [flags] [DOMAIN]",
	GroupID:      CommandGroupCredentials,
	Short:        "Enter Credentials",
	Long:         loginCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:      "false",
		CmdRequireHome: "false",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		urlMatch := "itrsgroup.com"

		if loginCmdList {
			var err2 error
			cr, err2 := config.Load("credentials",
				config.SetAppName(Execname),
				config.UseDefaults(false),
				config.IgnoreWorkingDir(),
			)
			if err2 != nil {
				return err2
			}
			for d := range cr.GetStringMap("credentials") {
				fmt.Println(d)
			}

			return
		}

		if loginCmdUsername == "" {
			if loginCmdUsername, err = config.ReadUserInputLine("Username: "); err != nil {
				return
			}
		}

		var createKeyfile bool
		if loginKeyfile == "" {
			// use default, create if none
			loginKeyfile = DefaultUserKeyfile
			createKeyfile = true
		}

		log.Debug().Msgf("checking keyfile %q, default file %q", loginKeyfile, DefaultUserKeyfile)

		if crc, created, err := loginKeyfile.ReadOrCreate(host.Localhost, createKeyfile); err != nil {
			return err
		} else if created {
			fmt.Printf("%s created, checksum %08X\n", loginKeyfile, crc)
		}

		var enc string
		if loginCmdPassword.IsNil() {
			// prompt for password
			enc, err = loginKeyfile.EncodePasswordInput(host.Localhost, true)
			if err != nil {
				return
			}
		} else {
			enc, err = loginKeyfile.Encode(host.Localhost, loginCmdPassword, true)
			if err != nil {
				return
			}
		}

		// default URL pattern
		if len(args) > 0 {
			urlMatch = args[0]
		}

		if err = config.AddCreds(config.Credentials{
			Domain:   urlMatch,
			Username: loginCmdUsername,
			Password: enc,
		}, config.SetAppName(Execname)); err != nil {
			return err
		}

		return
	},
}
