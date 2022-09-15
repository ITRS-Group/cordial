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
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/spf13/cobra"
)

// aesDecodeCmd represents the aesDecode command
var aesDecodeCmd = &cobra.Command{
	Use:                   "decode [-k KEYFILE] [-p KEYFILE] [-P PASSWORD] [-s SOURCE] [TYPE] [NAME]",
	Short:                 "Decode an AES256 encoded password",
	Long:                  `Decode an AES256 encoded password given a keyfile (or previous keyfile). If no keyfiles are explicitly provided then all matching instances are checked for configured keyfiles and each one tried. An error is only returned if all attempts to decode fail. If the given password has a prefix of '+encs+' it is removed. If both -P and -s options are given then the -P argsument is used. To read a password from STDIN use '-s -'.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		var ciphertext string

		if aesDecodeCmdPassword != "" {
			ciphertext = strings.TrimPrefix(aesDecodeCmdPassword, "+encs+")
		} else if aesDecodeCmdSource != "" {
			b, err := geneos.ReadLocalFileOrURL(aesDecodeCmdSource)
			if err != nil {
				return err
			}
			ciphertext = strings.TrimPrefix(string(b), "+encs+")
		} else {
			return geneos.ErrInvalidArgs
		}

		for _, k := range []string{aesDecodeCmdAESFILE, aesDecodeCmdPrevAESFILE} {
			if k == "" {
				continue
			}
			// encode using  specific file
			r, _, err := geneos.OpenLocalFileOrURL(k)
			if err != nil {
				continue
			}
			defer r.Close()
			a, err := config.ReadAESValues(r)
			if err != nil {
				continue
			}
			e, err := a.DecodeAESString(ciphertext)
			if err != nil {
				continue
			}
			log.Printf("decoded: %s\n", e)
			return nil
		}

		if aesDecodeCmdAESFILE != "" || aesDecodeCmdPrevAESFILE != "" {
			return fmt.Errorf("decode failed with key file(s) provided")
		}

		ct, args, _ := cmdArgsParams(cmd)
		params := []string{ciphertext}
		return instance.ForAll(ct, aesDecodeInstance, args, params)
	},
}

var aesDecodeCmdAESFILE, aesDecodeCmdPrevAESFILE, aesDecodeCmdPassword, aesDecodeCmdSource string

func init() {
	aesCmd.AddCommand(aesDecodeCmd)

	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdAESFILE, "keyfile", "k", "", "Main AES key file to use")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdPrevAESFILE, "previous", "p", "", "Previous AES key file to use")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdPassword, "password", "s", "", "Password to decode")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdSource, "source", "S", "", "Source for password to use")
	aesDecodeCmd.Flags().SortFlags = false

}

func aesDecodeInstance(c geneos.Instance, params []string) (err error) {
	if c.Type() != &gateway.Gateway {
		return nil
	}
	aesfile := c.GetConfig().GetString("aesfile")
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
	e, err := a.DecodeAESString(params[0])
	if err != nil {
		return nil
	}
	log.Printf("%s: %q\n", c, e)
	return nil
}
