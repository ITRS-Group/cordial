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
)

var aesDecodeCmdAESFILE, aesDecodeCmdPrevAESFILE, aesPrevUserKeyFile config.KeyFile
var aesDecodeCmdPassword, aesDecodeCmdSource, aesDecodeCmdExpandString string

func init() {
	AesCmd.AddCommand(aesDecodeCmd)

	cmd.UserKeyFile = cmd.DefaultUserKeyfile
	aesPrevUserKeyFile = config.KeyFile(config.Path("prevkeyfile",
		config.SetAppName(cmd.Execname),
		config.SetFileFormat("aes"),
		config.IgnoreWorkingDir(),
	))

	aesDecodeCmdAESFILE = cmd.UserKeyFile
	aesDecodeCmdPrevAESFILE = aesPrevUserKeyFile

	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdExpandString, "expandable", "e", "", "The keyfile and ciphertext in expandable format (including '${...}')")
	aesDecodeCmd.Flags().VarP(&aesDecodeCmdAESFILE, "keyfile", "k", "Path to keyfile")
	aesDecodeCmd.Flags().VarP(&aesDecodeCmdPrevAESFILE, "previous", "v", "Path to previous keyfile")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdPassword, "password", "p", "", "'Geneos formatted AES256 password")
	aesDecodeCmd.Flags().StringVarP(&aesDecodeCmdSource, "source", "s", "", "Alternative source for password")

	aesDecodeCmd.Flags().SortFlags = false
}

var aesDecodeCmd = &cobra.Command{
	Use:   "decode [flags] [TYPE] [NAME...]",
	Short: "Decode a Geneos AES256 format password using a key file",
	Long: strings.ReplaceAll(`
Decode a Geneos AES256 format password using the keyfile(s) given.

If an |expandable| string is given with the |-e| option it must be of
the form |${enc:...}| (be careful to single-quote this string when
using a shell) and is then decoded using the keyfile(s) listed and
the ciphertext in the value. All other flags and arguments are
ignored.

The format of |expandable| strings is documented here:

<https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#ExpandString>

A specific key file can be given using the |-k| flag and an
alternative ("previous") key file with the |-v| flag. If either of
these key files are supplied then the command tries to decode the
given ciphertext and a value may be returned. An error is returned if
all attempts fail.

Finally, if no keyfiles are provided then matching instances are
checked for configured keyfiles and each one tried or the default
keyfile paths are tried. An error is only returned if all attempts to
decode fail. The ciphertext may contain the optional prefix |+encs+|.
If both |-p| and |-s| options are given then the argument to the |-p|
flag is used. To read a ciphertext from STDIN use |-s -|.
`, "|", "`"),
	Example: `
# don't forget to use single quotes to escape the ${...} from shell
# interpolation
geneos aes decode -e '${enc:~/.config/geneos/keyfile.aes:hexencodedciphertext}'

# decode from the environment variable "MY_ENCODED_PASSWORD"
geneos aes decode -e '${enc:~/.config/geneos/keyfile.aes:env:MY_ENCODED_PASSWORD}'

# try to decode using AES key file configured for all instances
geneos aes decode -p +encs+hexencodedciphertext

# try to decode using the AES key file associated with the 'Demo Gateway' instance
geneos aes decode gateway 'Demo Gateway' -p +encs+hexencodedciphertext
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		var ciphertext string

		// XXX Allow -e to provide non-inline sources, e.g. stdin, file etc.
		if strings.HasPrefix(aesDecodeCmdExpandString, "${enc:") {
			fmt.Println(config.ExpandString(aesDecodeCmdExpandString))
			return nil
		}

		if aesDecodeCmdExpandString != "" {
			return fmt.Errorf("%w: expandable string must be of the form '${enc:keyfile:ciphertext}'", geneos.ErrInvalidArgs)
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
		return instance.ForAll(ct, cmd.Hostname, aesDecodeInstance, args, params)
	},
}

func aesDecodeInstance(c geneos.Instance, params []string) (err error) {
	log.Debug().Msgf("trying to decode for instance %s", c)
	if !c.Type().UsesKeyfiles {
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
	a := config.Read(r)
	e, err := a.DecodeString(params[0])
	if err != nil {
		return nil
	}
	fmt.Printf("%s: %q\n", c, e)
	return nil
}
