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
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var aesPasswordCmdString config.Plaintext
var aesPasswordCmdSource string

func init() {
	AesCmd.AddCommand(aesPasswordCmd)

	aesPasswordCmd.Flags().VarP(&aesPasswordCmdString, "password", "p", "A plaintext password")
	aesPasswordCmd.Flags().StringVarP(&aesPasswordCmdSource, "source", "s", "", "External source for plaintext `PATH|URL|-`")
}

// aesPasswordCmd represents the password command
var aesPasswordCmd = &cobra.Command{
	Use:   "password [flags]",
	Short: "Encode a password with an AES256 key file",
	Long: strings.ReplaceAll(`
Encode a password using the user's keyfile. If no keyfile exists it
is created. Output is in "expandable" format.

You will be prompted to enter the password (twice, for validation)
unless on of the flags is set.

To encode a plaintext password using a specific key file please use
the |geneos aes encode| command
`, "|", "`"),
	Aliases:      []string{"passwd"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var plaintext config.Plaintext

		crc, created, err := cmd.DefaultUserKeyfile.Check(true)
		if err != nil {
			return
		}

		if created {
			fmt.Printf("%s created, checksum %08X\n", cmd.DefaultUserKeyfile, crc)
		}

		if !aesPasswordCmdString.IsNil() {
			plaintext = aesPasswordCmdString
		} else if aesPasswordCmdSource != "" {
			var pt []byte
			pt, err = geneos.ReadFrom(aesPasswordCmdSource)
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
