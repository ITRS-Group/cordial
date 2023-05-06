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

package aes

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var aesSetCmdKeyfile config.KeyFile
var aesSetCmdCRC string
var aesSetCmdNoRoll bool

func init() {
	AesCmd.AddCommand(aesSetCmd)

	aesSetCmdKeyfile = cmd.DefaultUserKeyfile
	aesSetCmd.Flags().StringVarP(&aesSetCmdCRC, "crc", "C", "", "CRC of existing component shared keyfile to use")
	aesSetCmd.Flags().VarP(&aesSetCmdKeyfile, "keyfile", "k", "Keyfile to import and use")
	aesSetCmd.Flags().BoolVarP(&aesSetCmdNoRoll, "noroll", "N", false, "Do not roll any existing keyfile to previous keyfile setting")
}

var aesSetCmd = &cobra.Command{
	Use:   "set [flags] [TYPE] [NAME...]",
	Short: "Set keyfile for instances",
	Long: strings.ReplaceAll(`
Set a keyfile for matching instances. The keyfile is saved to each
matching component shared directory and the configuration set to
that path.

The keyfile can be given as either an existing CRC (without file
extension) or as a path or URL. If neither |-C| or |-k| are given
then the user's default keyfile is used, if found.

If the |-C| flag is given and it identifies an existing keyfile in
the component shared directory then that is used for matching
instances. When TYPE is not given, the keyfile will also be copied to
the shared directories of other component types if not already
present.

The |-k| flag value can be a local file (including a prefix of |~/|
to represent the home directory), a URL or a dash |-| for STDIN. The
given keyfile is evaluated and its CRC32 checksum checked against
existing keyfiles in the matching component shared directories. The
keyfile is only saved if one with the same checksum does not already
exist. 

Any existing |keyfile| path is copied to a |prevkeyfile| setting,
unless the |-N| option if given, to support key file updating in
Geneos GA6.x.

Currently only Gateways and Netprobes (and SANs) are supported.

Only local keyfiles, unless given as a URL, can be copied to remote
hosts, not visa versa. Referencing a keyfile by CRC on a remote host
will not result in that file being copies to other hosts.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args := cmd.CmdArgs(command)

		var crclist []string
		var a config.KeyValues

		var keyfile config.KeyFile

		if aesSetCmdCRC == "" {
			keyfile = aesSetCmdKeyfile
		} else {
			// search for existing CRC in all shared dirs
			var path string
			for _, ct := range ct.Range(componentsWithKeyfiles...) {
				path = host.LOCAL.Filepath(ct, ct.String()+"_shared", "keyfiles", aesSetCmdCRC+".aes")
				log.Debug().Msgf("looking for keyfile %s", path)
				if _, err := host.LOCAL.Stat(path); err == nil {
					break
				}
				path = ""
			}

			if path == "" {
				return fmt.Errorf("keyfile with CRC %q not found locally", aesSetCmdCRC)
			}
			keyfile = config.KeyFile(path)
		}

		crc, _, err := keyfile.Check(false)
		if err != nil {
			return err
		}
		crclist = []string{fmt.Sprintf("%08X", crc)}

		// at this point we have an AESValue struct and a CRC to use as a test
		// create 'keyfiles' directory as required
		for _, ct := range ct.Range(componentsWithKeyfiles...) {
			for _, h := range host.AllHosts() {
				// only import if it is not found
				path := h.Filepath(ct, ct.String()+"_shared", "keyfiles", crclist[0]+".aes")
				if _, err := h.Stat(path); err != nil && errors.Is(err, fs.ErrNotExist) {
					aesImportSave(ct, h, &a)
				} else if err == nil {
					log.Debug().Msgf("not importing existing %q CRC named keyfile on %s", crclist[0], h)
				}
			}
		}

		// params[0] is the CRC
		for _, ct := range ct.Range(componentsWithKeyfiles...) {
			instance.ForAll(ct, aesSetAESInstance, args, crclist)
		}
		return nil
	},
}

func aesSetAESInstance(c geneos.Instance, params []string) (err error) {
	path := c.Host().Filepath(c.Type(), c.Type().String()+"_shared", "keyfiles", params[0]+".aes")

	// roll old file
	if !aesSetCmdNoRoll {
		p := c.Config().GetString("keyfile")
		if p != "" {
			if p == path {
				fmt.Printf("%s: new and existing keyfile have same CRC. Not updating\n", c)
			} else {
				c.Config().Set("keyfile", path)
				c.Config().Set("prevkeyfile", p)
				fmt.Printf("%s keyfile %s set, existing keyfile moved to prevkeyfile\n", c, params[0])
			}
		} else {
			c.Config().Set("keyfile", path)
			fmt.Printf("%s keyfile %s set\n", c, params[0])
		}
	} else {
		c.Config().Set("keyfile", path)
		fmt.Printf("%s keyfile %s set\n", c, params[0])
	}

	if err = instance.WriteConfig(c); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return
}
