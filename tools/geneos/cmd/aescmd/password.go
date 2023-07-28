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

package aescmd

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var passwordCmdString = &config.Plaintext{}
var passwordCmdSource string

func init() {
	aesCmd.AddCommand(passwordCmd)

	passwordCmd.Flags().VarP(passwordCmdString, "password", "p", "A plaintext password")
	passwordCmd.Flags().StringVarP(&passwordCmdSource, "source", "s", "", "External source for plaintext `PATH|URL|-`")
}

//go:embed _docs/password.md
var passwordCmdDescription string

var passwordCmd = &cobra.Command{
	Use:          "password [flags]",
	Short:        "Encode a password with an AES256 key file",
	Long:         passwordCmdDescription,
	Aliases:      []string{"passwd"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var plaintext *config.Plaintext

		crc, created, err := cmd.DefaultUserKeyfile.Check(true)
		if err != nil {
			return
		}

		if created {
			fmt.Printf("%s created, checksum %08X\n", cmd.DefaultUserKeyfile, crc)
		}

		if !passwordCmdString.IsNil() {
			plaintext = passwordCmdString
		} else if passwordCmdSource != "" {
			var pt []byte
			pt, err = geneos.ReadFrom(passwordCmdSource)
			if err != nil {
				return
			}
			plaintext = config.NewPlaintext(pt)
		} else {
			plaintext, err = config.ReadPasswordInput(true, 3)
			if err != nil {
				return
			}
		}
		e, err := cmd.DefaultUserKeyfile.Encode(plaintext, true)
		if err != nil {
			return err
		}
		fmt.Println(e)
		return nil
	},
}
