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

package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
	"github.com/spf13/cobra"
)

var aesEncodeCmdAESFILE, aesEncodeCmdString, aesEncodeCmdSource string
var aesEncodeCmdExpandable, aesEncodeCmdAskOnce bool

// var aesEncodeDefaultKeyfile string

var plaintext []byte

func init() {
	aesCmd.AddCommand(aesEncodeCmd)

	// aesEncodeDefaultKeyfile = geneos.UserConfigFilePaths("keyfile.aes")[0]

	aesEncodeCmd.Flags().StringVarP(&aesEncodeCmdAESFILE, "keyfile", "k", "", "Specific AES key file to use. Ignores matching instances")
	aesEncodeCmd.Flags().StringVarP(&aesEncodeCmdString, "password", "p", "", "Password string to use")
	aesEncodeCmd.Flags().StringVarP(&aesEncodeCmdSource, "source", "s", "", "Source for password to use")
	aesEncodeCmd.Flags().BoolVarP(&aesEncodeCmdExpandable, "expandable", "e", false, "Output in ExpandString format")
	aesEncodeCmd.Flags().BoolVarP(&aesEncodeCmdAskOnce, "once", "o", false, "Only prompt for password once. For scripts injecting passwords on stdin")

	aesEncodeCmd.Flags().SortFlags = false
}

var aesEncodeCmd = &cobra.Command{
	Use:   "encode [flags] [TYPE] [NAME...]",
	Short: "Encode a password using a Geneos AES file",
	Long: strings.ReplaceAll(`
Encode a password (or any other string) using the keyfile for a
Geneos Gateway. By default the user is prompted to enter a password
but can provide a string or URL with the |-p| option. If TYPE and NAME
are given then the key files are checked for those instances. If
multiple instances match then the given password is encoded for each
keyfile found.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, origargs []string) (err error) {
		if aesEncodeCmdString != "" {
			plaintext = []byte(aesEncodeCmdString)
		} else if aesEncodeCmdSource != "" {
			plaintext, err = geneos.ReadSource(aesEncodeCmdSource)
			if err != nil {
				return
			}
		} else {
			if aesEncodeCmdAskOnce {
				plaintext = config.ReadPasswordPrompt()
			} else {
				var match bool
				for i := 0; i < 3; i++ {
					plaintext = config.ReadPasswordPrompt()
					plaintext2 := config.ReadPasswordPrompt("Re-enter Password")
					if bytes.Equal(plaintext, plaintext2) {
						match = true
						break
					}
					fmt.Println("Passwords do not match. Please try again.")
				}
				if !match {
					return fmt.Errorf("too many attempts, giving up")
				}
			}
		}

		if aesEncodeCmdAESFILE != "" {
			// encode using specific file
			e, err := config.EncodeWithKeyfile(plaintext, aesEncodeCmdAESFILE, aesEncodeCmdExpandable)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", e)
			return nil
		}

		ct, args := cmdArgs(cmd)
		err = instance.ForAll(ct, aesEncodeInstance, args, []string{})
		plaintext = bytes.Repeat([]byte{0}, len(plaintext))
		return
	},
}

func aesEncodeInstance(c geneos.Instance, params []string) (err error) {
	if !(c.Type() == &gateway.Gateway || c.Type() == &netprobe.Netprobe || c.Type() == &san.San) {
		return nil
	}
	keyfile := instance.Filepath(c, "keyfile")
	if keyfile == "" {
		return
	}

	e, err := config.EncodeWithKeyfile(plaintext, keyfile, aesEncodeCmdExpandable)
	if err != nil {
		return
	}
	fmt.Printf("%s: %s\n", c, e)
	return nil
}
