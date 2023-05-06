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
	"sort"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var loginCmdSiteURL, loginCmdUsername, loginCmdPassword string
var loginKeyfile config.KeyFile

func init() {
	RootCmd.AddCommand(loginCmd)

	loginKeyfile = UserKeyFile
	loginCmd.Flags().StringVarP(&loginCmdUsername, "username", "u", "", "Username")
	loginCmd.Flags().StringVarP(&loginCmdPassword, "password", "p", "", "Password")
	loginCmd.Flags().VarP(&loginKeyfile, "keyfile", "k", "Keyfile to use")

	loginCmd.Flags().SortFlags = false

}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login [flags] [URLPATTERN]",
	Short: "Store credentials for software downloads",
	Long: strings.ReplaceAll(`
Prompt for and stored credentials for later use by commands.

Typical use is for downloading release archives from the official ITRS web
site.

If not given |URLPATTERN| defaults to |itrsgroup.com|. When credentials are
used, the destination is checked against all stored credentials and the
longest match is selected.

If no |-u USERNAME| is given then the user is prompted for a username.

If no |-p PASSWORD| is given then the user is prompted for the password,
which is not echoed, twice and it is only accepted if both instances match.

The credentials are encrypted with the keyfile specified with |-k KEYFILE|
and if not given then the user's default keyfile is used - and created if it
does not exist. See |geneos aes new| for details.

The credentials cannot be used without the keyfile and each set of
credentials can use a separate keyfile.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
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

		if crc, created, err := loginKeyfile.Check(createKeyfile); err != nil {
			return err
		} else if created {
			fmt.Printf("%s created, checksum %08X\n", loginKeyfile, crc)
		}

		var enc string
		if loginCmdPassword == "" {
			// prompt for password
			enc, err = loginKeyfile.EncodePasswordInput(true)
		}

		urlMatch := "itrsgroup.com"

		// default URL pattern
		if len(args) > 0 {
			urlMatch = args[0]
		}

		fmt.Printf("username=%s, password=%s\n", loginCmdUsername, enc)
		setCredentials(urlMatch, loginCmdUsername, enc)
		log.Debug().Msgf("conf: %+v", config.GetConfig().AllSettings())
		return
	},
}

type auth struct {
	Username string
	Password string
}

// var creds map[string]auth

func getCredentials(path string) (a auth) {
	cf := config.GetConfig()
	creds := cf.GetStringMap("credentials")
	paths := []string{}
	for k, _ := range creds {
		if strings.Contains(path, k) {
			paths = append(paths, k)
		}
	}
	if len(paths) == 0 {
		return
	}
	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i]) < len(paths[j])
	})

	log.Debug().Msgf("paths: %v", paths)
	switch v := creds[paths[0]].(type) {
	case auth:
		return v
	default:
		return
	}
	return
}

func setCredentials(urlmatch, username, password string) {
	cf := config.GetConfig()
	creds := cf.GetStringMap("credentials")
	a := auth{
		Username: username,
		Password: password,
	}
	creds[urlmatch] = a
	cf.Set("credentials", creds)
}
