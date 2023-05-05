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

	"github.com/spf13/cobra"
)

var loginCmdSiteURL, loginCmdUsername, loginCmdPassword, loginKeyfile string

func init() {
	RootCmd.AddCommand(loginCmd)

	loginCmd.Flags().StringVarP(&loginCmdUsername, "username", "u", "", "Username")
	loginCmd.Flags().StringVarP(&loginCmdPassword, "password", "p", "", "Password")
	loginCmd.Flags().StringVarP(&loginKeyfile, "keyfile", "k", UserKeyFile, "Keyfile to use")

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
	Run: func(cmd *cobra.Command, args []string) {
		if loginCmdUsername == "" {
			// prompt for username
		}

		if loginCmdPassword == "" {
			// prompt for password
		}

		if loginKeyfile == "" {
			// use default, create if none
			loginKeyfile = DefaultUserKeyfile
		}

		if len(args) == 0 {
			// default URL pattern
		}

		// enc, err := config.EncodePasswordPrompt(loginKeyfile, true)

		// save
		fmt.Println("login called")
	},
}
