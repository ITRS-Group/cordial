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
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
)

var loginCmdUsername string
var loginCmdPassword config.Plaintext
var loginKeyfile config.KeyFile
var loginCmdList bool

func init() {
	GeneosCmd.AddCommand(loginCmd)

	loginCmd.Flags().StringVarP(&loginCmdUsername, "username", "u", "", "Username")
	loginCmd.Flags().VarP(&loginCmdPassword, "password", "p", "Password")
	loginCmd.Flags().VarP(&loginKeyfile, "keyfile", "k", "Key file to use")
	loginCmd.Flags().BoolVarP(&loginCmdList, "list", "l", false, "List the names of the currently stored credentials")

	loginCmd.Flags().SortFlags = false

}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:     "login [flags] [DOMAIN]",
	GroupID: GROUP_CREDENTIALS,
	Short:   "Store credentials related to Geneos",
	Long: strings.ReplaceAll(`
The login command will store credentials in your local configuration
directory for use with |geneos| and other tools from |cordial|.
Passwords are encrypted using a key file which is created if it does
not already exist.

If not given |DOMAIN| defaults to |itrsgroup.com|. When
credentials are used, the destination is checked against all stored
credentials and the longest match is selected.

A common use of stored credentials is for the download and
installation of Geneos packages via the |geneos package| subsystem.
Credentials are also used by the |geneos snapshot| command and the
|dv2email| program.

If no |--username|/|-u| option is given then the user is prompted for one.

If no |--password|/|-p| is given then the user is prompted to enter
the password twice and it is only accepted if both instances match.
After three failures to match password the program will terminate and
not save the credential.

The user's default key file is used unless the |--keyfile|/|-k| is
given. The path to the key file used is stored in the credential and
so if the key file is moved or overwritten then that credential
becomes unusable.

The credentials cannot be used without the original key file and each
set of credentials can use a separate key file.

The credentials file itself can be world readable as the security is
through the use of a protected key file. Running |geneos.exe| on
Windows does not currently protect the key file on creation.

Future releases will support extended credential sets, for example
SSH and 2-legged OAuth ClientID/ClientSecret (such as application
keys from cloud providers). Another addition may be the automatic
encryption of non-password data in credentials.
`, "|", "`"),
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
