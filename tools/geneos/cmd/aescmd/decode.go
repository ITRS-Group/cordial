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
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
)

var aesDecodeCmdAESFILE, aesDecodeCmdPrevAESFILE, aesPrevUserKeyFile config.KeyFile
var aesDecodeCmdPassword, aesDecodeCmdSource, aesDecodeCmdExpandString string

func init() {
	AesCmd.AddCommand(aesDecodeCmd)

	cmd.UserKeyFile = cmd.DefaultUserKeyfile
	aesPrevUserKeyFile = config.KeyFile(geneos.UserConfigFilePaths("prevkeyfile.aes")[0])

	aesDecodeCmdAESFILE = cmd.UserKeyFile
	aesDecodeCmdPrevAESFILE = aesPrevUserKeyFile

	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdExpandString, "expand", "e", "", "A string in ExpandString format (including '${...}') to decode")
	aesDecodeCmd.Flags().VarP(&aesDecodeCmdAESFILE, "keyfile", "k", "Main AES key file to use")
	aesDecodeCmd.Flags().VarP(&aesDecodeCmdPrevAESFILE, "previous", "v", "Previous AES key file to use")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdPassword, "password", "p", "", "Password to decode")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdSource, "source", "s", "", "Source for password to use")

	aesDecodeCmd.Flags().SortFlags = false
}

var aesDecodeCmd = &cobra.Command{
	Use:   "decode [flags] [TYPE] [NAME...]",
	Short: "Decode a password using a Geneos compatible keyfile",
	Long: strings.ReplaceAll(`
Decode a Geneos-format AES256 encoded password using the keyfile(s)
given.

If no keyfiles are provided then all matching instances are checked
for configured keyfiles and each one tried or the default keyfile
paths are tried. An error is only returned if all attempts to decode
fail. The ciphertext may contain the optional prefix |+encs+|. If
both |-P| and |-s| options are given then the argument to the |-P|
flag is used. To read a ciphertext from STDIN use |-s -|.

If an |expandable| string is given with the |-e| option it must be of
the form |${enc:...}| (be careful to single-quote this string when
using a shell) and is then decoded using the keyfile and ciphertext
in the value. All other flags and arguments are ignored.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		var ciphertext string

		// XXX Allow -e to provide non-inline sources, e.g. stdin, file etc.
		if strings.HasPrefix(aesDecodeCmdExpandString, "${enc:") {
			fmt.Println(config.ExpandString(aesDecodeCmdExpandString))
			return nil
		}

		if aesDecodeCmdExpandString != "" {
			return fmt.Errorf("%w: expandable string must be of the form '${enc:keyfile:ciphertext}'", cmd.ErrInvalidArgs)
		}

		if aesDecodeCmdPassword != "" {
			ciphertext = aesDecodeCmdPassword
		} else if aesDecodeCmdSource != "" {
			b, err := geneos.ReadFrom(aesDecodeCmdSource)
			if err != nil {
				return err
			}
			ciphertext = string(b)
		} else {
			return geneos.ErrInvalidArgs
		}

		for _, k := range []config.KeyFile{aesDecodeCmdAESFILE, aesDecodeCmdPrevAESFILE} {
			if k == "" {
				continue
			}

			e, err := k.DecodeString(ciphertext)
			if err != nil {
				continue
			}
			fmt.Printf("decoded: %s\n", e)
			return nil
		}

		if aesDecodeCmdAESFILE != cmd.UserKeyFile || aesDecodeCmdPrevAESFILE != aesPrevUserKeyFile {
			return fmt.Errorf("decode failed with key file(s) provided")
		}

		ct, args, _ := cmd.CmdArgsParams(command)
		params := []string{ciphertext}
		return instance.ForAll(ct, aesDecodeInstance, args, params)
	},
}

func aesDecodeInstance(c geneos.Instance, params []string) (err error) {
	log.Debug().Msgf("trying to decode for instance %s", c)
	if !(c.Type() == &gateway.Gateway || c.Type() == &netprobe.Netprobe || c.Type() == &san.San) {
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
	a, err := config.Read(r)
	if err != nil {
		return
	}
	e, err := a.DecodeString(params[0])
	if err != nil {
		return nil
	}
	fmt.Printf("%s: %q\n", c, e)
	return nil
}
