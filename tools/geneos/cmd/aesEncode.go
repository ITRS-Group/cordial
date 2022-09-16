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
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/spf13/cobra"
)

// aesEncodeCmd represents the aesEncode command
var aesEncodeCmd = &cobra.Command{
	Use:                   "encode [-k KEYFILE] [-P STRING] [-s SOURCEPATH] [TYPE] [NAME]",
	Short:                 "Encode a password using a Geneos AES file",
	Long:                  `Encode a password (or any other string) using the AES file for a Geneos Gateway. By default the user is prompted to enter a password but can provide a string or URL with the -p option. If TYPE and NAME are given then the key files are checked for those instances. If multiple instances match then the given password is encoded for each key file found.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		var plaintext string

		if aesEncodeCmdString != "" {
			plaintext = aesEncodeCmdString
		} else if aesEncodeCmdSource != "" {
			b, err := geneos.ReadLocalFileOrURL(aesEncodeCmdSource)
			if err != nil {
				panic(err)
			}
			plaintext = string(b)
		} else {
			plaintext = utils.ReadPasswordPrompt()
		}

		if aesEncodeCmdAESFILE != "" {
			// encode using  specific file
			r, _, err := geneos.OpenLocalFileOrURL(aesEncodeCmdAESFILE)
			if err != nil {
				panic(err)
			}
			defer r.Close()
			a, err := config.ReadAESValues(r)
			if err != nil {
				panic(err)
			}
			e, err := a.EncodeAESString(plaintext)
			if err != nil {
				panic(err)
			}
			log.Printf("encoded: +encs+%s\n", e)
			return nil
		}

		ct, args, _ := cmdArgsParams(cmd)
		// override params ...
		params := []string{plaintext}
		return instance.ForAll(ct, aesEncodeInstance, args, params)
	},
}

var aesEncodeCmdAESFILE, aesEncodeCmdString, aesEncodeCmdSource string

func init() {
	aesCmd.AddCommand(aesEncodeCmd)

	aesEncodeCmd.Flags().StringVarP(&aesEncodeCmdAESFILE, "keyfile", "k", "", "Main AES key file to use")
	aesEncodeCmd.Flags().StringVarP(&aesEncodeCmdString, "password", "s", "", "Password string to use")
	aesEncodeCmd.Flags().StringVarP(&aesEncodeCmdSource, "source", "S", "", "Source for password to use")
	aesEncodeCmd.Flags().SortFlags = false
}

func aesEncodeInstance(c geneos.Instance, params []string) (err error) {
	if c.Type() != &gateway.Gateway {
		return nil
	}
	aesfile := c.Config().GetString("keyfile")
	if aesfile == "" {
		return nil
	}
	aespath := filepath.Join(c.Home(), aesfile)

	r, err := c.Host().Open(aespath)
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
	log.Printf("%s: +encs+%s\n", c, e)
	return nil
}
