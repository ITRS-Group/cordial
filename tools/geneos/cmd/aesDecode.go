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
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// aesDecodeCmd represents the aesDecode command
var aesDecodeCmd = &cobra.Command{
	Use:   "decode [-e STRING] [-k KEYFILE] [-p KEYFILE] [-P PASSWORD] [-s SOURCE] [TYPE] [NAME]",
	Short: "Decode an AES256 encoded value",
	Long: `Decode an AES256 encoded value.

If an expandable string is given with the '-e' option it must be of
the form '${enc:...}' and is then decoded using the keyfile and
ciphertext in the value. Other options are ignored.
	
Given a keyfile (or previous keyfile). If no keyfiles are explicitly
provided then all matching instances are checked for configured
keyfiles and each one tried or the default keyfile paths are tried.
An error is only returned if all attempts to decode fail. If the
given password has a prefix of '+encs+' it is removed. If both -P and
-s options are given then the -P argument is used. To read a password
from STDIN use '-s -'.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		var ciphertext string

		if strings.HasPrefix(aesDecodeCmdExpandString, "${enc:") {
			fmt.Println(config.GetConfig().ExpandString(aesDecodeCmdExpandString))
			return nil
		}

		if aesDecodeCmdExpandString != "" {
			return fmt.Errorf("%w: expandable string must be of the form '${enc:keyfile:ciphertext}'", ErrInvalidArgs)
		}

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
			// decode using specific file
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
			fmt.Printf("decoded: %s\n", e)
			return nil
		}

		if aesDecodeCmdAESFILE != defKeyFile || aesDecodeCmdPrevAESFILE != defPrevKeyFile {
			return fmt.Errorf("decode failed with key file(s) provided")
		}

		ct, args, _ := cmdArgsParams(cmd)
		params := []string{ciphertext}
		return instance.ForAll(ct, aesDecodeInstance, args, params)
	},
}

var aesDecodeCmdAESFILE, aesDecodeCmdPrevAESFILE, aesDecodeCmdPassword, aesDecodeCmdSource, aesDecodeCmdExpandString string
var defKeyFile, defPrevKeyFile string

func init() {
	aesCmd.AddCommand(aesDecodeCmd)

	defKeyFile = geneos.UserConfigFilePaths("keyfile.aes")[0]
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdExpandString, "expand", "e", "", "A string in ExpandString format (including '${...}') to decode")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdAESFILE, "keyfile", "k", defKeyFile, "Main AES key file to use")
	defPrevKeyFile = geneos.UserConfigFilePaths("prevkeyfile.aes")[0]
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdPrevAESFILE, "previous", "v", defPrevKeyFile, "Previous AES key file to use")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdPassword, "password", "p", "", "Password to decode")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdSource, "source", "s", "", "Source for password to use")
	aesDecodeCmd.Flags().SortFlags = false

}

func aesDecodeInstance(c geneos.Instance, params []string) (err error) {
	log.Debug().Msgf("trying to decode for instance %s", c)
	if c.Type() != &gateway.Gateway {
		return
	}
	path := instance.Filepath(c, "keyfile")
	if path == "" {
		return
	}
	r, err := c.Host().Open(path)
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
	fmt.Printf("%s: %q\n", c, e)
	return nil
}
