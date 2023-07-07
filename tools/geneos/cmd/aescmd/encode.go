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
	_ "embed"
	"encoding/base64"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var encodeCmdKeyfile config.KeyFile
var encodeCmdString *config.Plaintext
var encodeCmdSource string
var encodeCmdExpandable, encodeCmdAskOnce bool

func init() {
	aesCmd.AddCommand(encodeCmd)

	encodeCmdString = &config.Plaintext{}

	encodeCmd.Flags().BoolVarP(&encodeCmdExpandable, "expandable", "e", false, "Output in 'expandable' format")
	encodeCmd.Flags().VarP(&encodeCmdKeyfile, "keyfile", "k", "Path to keyfile")
	encodeCmd.Flags().VarP(encodeCmdString, "password", "p", "Plaintext password")
	encodeCmd.Flags().StringVarP(&encodeCmdSource, "source", "s", "", "Alternative source for plaintext password")
	encodeCmd.Flags().BoolVarP(&encodeCmdAskOnce, "once", "o", false, "Only prompt for password once, do not verify. Normally use '-s -' for stdin")

	encodeCmd.Flags().SortFlags = false
}

//go:embed _docs/encode.md
var encodeCmdDescription string

var encodeCmd = &cobra.Command{
	Use:   "encode [flags] [TYPE] [NAME...]",
	Short: "Encode plaintext to a Geneos AES256 password using a key file",
	Long:  encodeCmdDescription,
	Example: `
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, origargs []string) (err error) {
		var plaintext *config.Plaintext
		if !encodeCmdString.IsNil() {
			plaintext = encodeCmdString
		} else if encodeCmdSource != "" {
			pt, err := geneos.ReadFrom(encodeCmdSource)
			if err != nil {
				return err
			}
			plaintext = config.NewPlaintext(pt)
		} else {
			plaintext, err = config.ReadPasswordInput(!encodeCmdAskOnce, 0)
			if err != nil {
				return
			}
		}

		if encodeCmdKeyfile != "" {
			// encode using specific file
			e, err := encodeCmdKeyfile.Encode(plaintext, encodeCmdExpandable)
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
	if !c.Type().UsesKeyfiles {
		return nil
	}
	keyfile := config.KeyFile(instance.PathOf(c, "keyfile"))
	if keyfile == "" {
		return
	}

	pw, _ := base64.StdEncoding.DecodeString(params[0])
	plaintext := config.NewPlaintext(pw)
	e, err := keyfile.Encode(plaintext, encodeCmdExpandable)
	if err != nil {
		return
	}
	fmt.Printf("%s: %s\n", c, e)
	return nil
}
