/*
Copyright © 2022 ITRS Group

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
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/floating"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
)

var aesEncodeCmdKeyfile config.KeyFile
var aesEncodeCmdString, aesEncodeCmdSource string
var aesEncodeCmdExpandable, aesEncodeCmdAskOnce bool

func init() {
	AesCmd.AddCommand(aesEncodeCmd)

	aesEncodeCmd.Flags().BoolVarP(&aesEncodeCmdExpandable, "expandable", "e", false, "Output in 'expandable' format")
	aesEncodeCmd.Flags().VarP(&aesEncodeCmdKeyfile, "keyfile", "k", "Path to keyfile")
	aesEncodeCmd.Flags().StringVarP(&aesEncodeCmdString, "password", "p", "", "Plaintext password")
	aesEncodeCmd.Flags().StringVarP(&aesEncodeCmdSource, "source", "s", "", "Alternative source for plaintext password")
	aesEncodeCmd.Flags().BoolVarP(&aesEncodeCmdAskOnce, "once", "o", false, "Only prompt for password once, do not verify. Normally use '-s -' for stdin")

	aesEncodeCmd.Flags().SortFlags = false
}

var aesEncodeCmd = &cobra.Command{
	Use:   "encode [flags] [TYPE] [NAME...]",
	Short: "Encode plaintext to a Geneos AES256 password using a key file",
	Long: strings.ReplaceAll(`
Encode plaintext to a Geneos AES256 format password using a key file.

A key file must either be provided using the |-k| option or otherwise all
matching instances that have a configured key file are used to produce an
encrypted password.

The plaintext password can be provided in three ways. The default is to
prompt for the plaintext and again to verify they match. Alternatively the
password can be provided directly on the command line using the |-p
plaintext| flag or from an external source using the |-s PATH| or |-s URL|
option where the contents of the file at PATH ir URL is read and used. If |-s
-| is used then the plaintext is read from STDIN.

It is important to note that no whitespace is trimmed from the
plaintext. This can have unexpected results if you do something like
this:

$ echo "test" | geneos aes encode -s -

rather then this:

$ echo -n "test" | geneos aes encode -s -
	`, "|", "`"),
	Example: `
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, origargs []string) (err error) {
		var plaintext config.Plaintext
		if aesEncodeCmdString != "" {
			plaintext = config.NewPlaintext([]byte(aesEncodeCmdString))
		} else if aesEncodeCmdSource != "" {
			pt, err := geneos.ReadFrom(aesEncodeCmdSource)
			if err != nil {
				return err
			}
			plaintext = config.NewPlaintext(pt)
		} else {
			plaintext, err = config.ReadPasswordInput(!aesEncodeCmdAskOnce, 0)
			if err != nil {
				return
			}
		}

		if aesEncodeCmdKeyfile != "" {
			// encode using specific file
			e, err := aesEncodeCmdKeyfile.Encode(plaintext, aesEncodeCmdExpandable)
			if err != nil {
				return err
			}
			fmt.Println(e)
			return nil
		}

		ct, args := cmd.CmdArgs(command)
		pw, _ := plaintext.Open()
		err = instance.ForAll(ct, cmd.Hostname, aesEncodeInstance, args, []string{base64.StdEncoding.EncodeToString(pw.Bytes())})
		pw.Destroy()
		return
	},
}

func aesEncodeInstance(c geneos.Instance, params []string) (err error) {
	if !(c.Type() == &gateway.Gateway || c.Type() == &netprobe.Netprobe || c.Type() == &san.San || c.Type() == &floating.Floating) {
		return nil
	}
	keyfile := config.KeyFile(instance.Filepath(c, "keyfile"))
	if keyfile == "" {
		return
	}

	pw, _ := base64.StdEncoding.DecodeString(params[0])
	plaintext := config.NewPlaintext(pw)
	e, err := keyfile.Encode(plaintext, aesEncodeCmdExpandable)
	if err != nil {
		return
	}
	fmt.Printf("%s: %s\n", c, e)
	return nil
}
