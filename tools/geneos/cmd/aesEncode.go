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
	"fmt"
	"os"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/spf13/cobra"
)

var aesEncodeCmdAESFILE, aesEncodeCmdString, aesEncodeCmdSource string
var aesEncodeCmdExpandable, aesEncodeCmdAskOnce bool

var aesEncodeDefaultKeyfile string

func init() {
	aesCmd.AddCommand(aesEncodeCmd)

	aesEncodeDefaultKeyfile = geneos.UserConfigFilePaths("keyfile.aes")[0]

	aesEncodeCmd.Flags().StringVarP(&aesEncodeCmdAESFILE, "keyfile", "k", aesEncodeDefaultKeyfile, "Main AES key file to use")
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
but can provide a string or URL with the -p option. If TYPE and NAME
are given then the key files are checked for those instances. If
multiple instances match then the given password is encoded for each
keyfile found.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, origargs []string) error {
		var plaintext string

		if aesEncodeCmdString != "" {
			plaintext = aesEncodeCmdString
		} else if aesEncodeCmdSource != "" {
			b, err := geneos.ReadSource(aesEncodeCmdSource)
			if err != nil {
				return err
			}
			plaintext = string(b)
		} else {
			if aesEncodeCmdAskOnce {
				plaintext = utils.ReadPasswordPrompt()
			} else {
				var match bool
				for i := 0; i < 3; i++ {
					plaintext = utils.ReadPasswordPrompt()
					plaintext2 := utils.ReadPasswordPrompt("Re-enter Password")
					if plaintext == plaintext2 {
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

		if len(origargs) == 0 {
			// encode using specific file
			r, _, err := geneos.OpenSource(aesEncodeCmdAESFILE)
			if err != nil {
				return err
			}
			defer r.Close()
			a, err := config.ReadAESValues(r)
			if err != nil {
				return err
			}
			e, err := a.EncodeAESString(plaintext)
			if err != nil {
				return err
			}

			if !aesEncodeCmdExpandable {
				fmt.Printf("+encs+%s\n", e)
				return nil
			}

			home, _ := os.UserHomeDir()
			if strings.HasPrefix(aesEncodeCmdAESFILE, home) {
				aesEncodeCmdAESFILE = "~" + strings.TrimPrefix(aesEncodeCmdAESFILE, home)
			}
			fmt.Printf("${enc:%s:+encs+%s}\n", aesEncodeCmdAESFILE, e)
			return nil
		}

		ct, args, _ := cmdArgsParams(cmd)
		// override params ...
		params := []string{plaintext}
		return instance.ForAll(ct, aesEncodeInstance, args, params)
	},
}

func aesEncodeInstance(c geneos.Instance, params []string) (err error) {
	if c.Type() != &gateway.Gateway {
		return nil
	}
	keyfile := instance.Filepath(c, "keyfile")
	if keyfile == "" {
		return
	}

	r, err := c.Host().Open(keyfile)
	if err != nil {
		return
	}
	defer r.Close()
	a, err := config.ReadAESValues(r)
	if err != nil {
		return
	}
	e, err := a.EncodeAESString(params[0])
	if err != nil {
		return
	}

	if !aesEncodeCmdExpandable {
		fmt.Printf("%s: +encs+%s\n", c, e)
		return nil
	}
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(keyfile, home) {
		keyfile = "~" + strings.TrimPrefix(keyfile, home)
	}
	fmt.Printf("%s: ${enc:%s:+encs+%s}\n", c, keyfile, e)
	return nil
}
