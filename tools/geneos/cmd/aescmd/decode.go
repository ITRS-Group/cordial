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
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var decodeCmdAESFILE, decodeCmdPrevAESFILE, aesPrevUserKeyFile config.KeyFile
var decodeCmdPassword, decodeCmdSource, decodeCmdExpandString string

func init() {
	aesCmd.AddCommand(decodeCmd)

	cmd.UserKeyFile = cmd.DefaultUserKeyfile
	aesPrevUserKeyFile = config.KeyFile(
		config.Path("prevkeyfile",
			config.SetAppName(cmd.Execname),
			config.SetFileExtension("aes"),
			config.IgnoreWorkingDir(),
		))

	decodeCmdAESFILE = cmd.UserKeyFile
	decodeCmdPrevAESFILE = aesPrevUserKeyFile

	decodeCmd.Flags().StringVarP(&decodeCmdExpandString, "expandable", "e", "", "The keyfile and ciphertext in expandable format (including '${...}')")
	decodeCmd.Flags().VarP(&decodeCmdAESFILE, "keyfile", "k", "Path to keyfile")
	decodeCmd.Flags().VarP(&decodeCmdPrevAESFILE, "previous", "v", "Path to previous keyfile")
	decodeCmd.Flags().StringVarP(&decodeCmdPassword, "password", "p", "", "'Geneos formatted AES256 password")
	decodeCmd.Flags().StringVarP(&decodeCmdSource, "source", "s", "", "Alternative source for password")

	decodeCmd.Flags().SortFlags = false
}

//go:embed _docs/decode.md
var decodeCmdDescription string

var decodeCmd = &cobra.Command{
	Use:   "decode [flags] [TYPE] [NAME...]",
	Short: "Decode a Geneos AES256 format password using a key file",
	Long:  decodeCmdDescription,
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
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var ciphertext string

		// XXX Allow -e to provide non-inline sources, e.g. stdin, file etc.
		if strings.HasPrefix(decodeCmdExpandString, "${enc:") {
			fmt.Println(config.ExpandString(decodeCmdExpandString))
			return nil
		}

		if decodeCmdExpandString != "" {
			return fmt.Errorf("%w: expandable string must be of the form '${enc:keyfile:ciphertext}'", geneos.ErrInvalidArgs)
		}

		if decodeCmdPassword != "" {
			ciphertext = decodeCmdPassword
		} else if decodeCmdSource != "" {
			b, err := geneos.ReadFrom(decodeCmdSource)
			if err != nil {
				return err
			}
			ciphertext = string(b)
		} else {
			return geneos.ErrInvalidArgs
		}

		for _, k := range []config.KeyFile{decodeCmdAESFILE, decodeCmdPrevAESFILE} {
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

		if decodeCmdAESFILE != cmd.UserKeyFile || decodeCmdPrevAESFILE != aesPrevUserKeyFile {
			return fmt.Errorf("decode failed with key file(s) provided")
		}

		ct, args, _ := cmd.CmdArgsParams(command)
		params := []string{ciphertext}
		_, err = instance.ForAllWithParamStringSlice(geneos.GetHost(cmd.Hostname), ct, aesDecodeInstance, args, params)
		return
	},
}

func aesDecodeInstance(c geneos.Instance, params []string) (result any, err error) {
	log.Debug().Msgf("trying to decode for instance %s", c)
	if !c.Type().UsesKeyfiles {
		return
	}
	path := instance.PathOf(c, "keyfile")
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
		err = nil
		return
	}
	fmt.Printf("%s: %q\n", c, e)
	return
}
