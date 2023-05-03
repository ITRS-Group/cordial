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

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Store credentials for software downloads",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("login called")
	},
}

var loginCmdSiteURL, loginCmdUsername, loginCmdPassword, loginKeyfile string

func init() {
	RootCmd.AddCommand(loginCmd)

	loginCmd.Flags().StringVarP(&loginCmdSiteURL, "url", "U", config.GetString("download.url"), `URL for download site for these credentials`)
	loginCmd.Flags().StringVarP(&loginCmdUsername, "username", "u", "", "Username for downloads, defaults to configuration value in download.username")
	loginCmd.Flags().StringVarP(&loginCmdPassword, "password", "P", "", "Password for downloads, defaults to configuration value in download.password or otherwise prompts")
	loginCmd.Flags().StringVarP(&loginKeyfile, "keyfile", "k", UserKeyFile, "Keyfile to use")

	loginCmd.Flags().SortFlags = false

}
