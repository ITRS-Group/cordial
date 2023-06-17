/*
Copyright © 2023 ITRS Group

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
	_ "embed"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
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
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		urlMatch := "itrsgroup.com"

		if loginCmdList {
			cr, _ := config.Load("credentials",
				config.SetAppName(Execname),
				config.UseDefaults(false),
				config.IgnoreWorkingDir(),
			)
			for d := range cr.GetStringMap("credentials") {
				fmt.Println(d)
			}

			return
		}

		if loginCmdUsername == "" {
			if loginCmdUsername, err = config.ReadUserInput("Username: "); err != nil {
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

		if crc, created, err := loginKeyfile.Check(createKeyfile); err != nil {
			return err
		} else if created {
			fmt.Printf("%s created, checksum %08X\n", loginKeyfile, crc)
		}

		var enc string
		if loginCmdPassword.IsNil() {
			// prompt for password
			enc, err = loginKeyfile.EncodePasswordInput(true)
			if err != nil {
				return
			}
		} else {
			enc, err = loginKeyfile.Encode(loginCmdPassword, true)
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

		log.Debug().Msgf("conf: %+v", config.GetConfig().AllSettings())
		return
	},
}
