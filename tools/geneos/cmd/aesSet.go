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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var aesSetCmdKeyfile, aesSetCmdCRC string
var aesSetCmdNoRoll bool

func init() {
	aesCmd.AddCommand(aesSetCmd)

	defKeyFile := geneos.UserConfigFilePaths("keyfile.aes")[0]

	aesSetCmd.Flags().StringVarP(&aesSetCmdKeyfile, "keyfile", "k", defKeyFile, "Keyfile to use")
	aesSetCmd.Flags().StringVarP(&aesSetCmdCRC, "crc", "C", "", "CRC of existing component shared keyfile to use")
	aesSetCmd.Flags().BoolVarP(&aesSetCmdNoRoll, "noroll", "N", false, "Do not roll any existing keyfile to previous keyfile setting")
}

var aesSetCmd = &cobra.Command{
	Use:   "set [flags] [TYPE] [NAME...]",
	Short: "Set keyfile for instances",
	Long: strings.ReplaceAll(`
Set keyfile for matching instances.

Either a path or URL to a keyfile, the CRC of an existing keyfile in
a local component's shared directory can be given or, if it exists, the
user's default keyfile is used. Unless the keyfile is referenced by
the CRC it is saved to the component shared directories and the
configuration set to reference that path.

Unless the |-N| flag is given any existing keyfile path is copied to
a 'prevkeyfile' setting to support key file updating in Geneos GA6.x.

If the |-C| flag is used and it identifies an existing keyfile in the
component keyfile directory then that is used for matching instances.

The argument given with the |-k| flag can be a local file (including
a prefix of |~/| to represent the home directory), a URL or a dash
|-| for STDIN.

Currently only Gateways and Netprobes (and SANs) are supported.

Only local keyfiles, unless given as a URL, can be copied to remote
hosts, not visa versa.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, args := cmdArgs(cmd)

		var crclist []string
		var f io.ReadCloser
		var a config.AESValues

		if aesSetCmdCRC == "" {
			f, _, err = geneos.OpenSource(aesSetCmdKeyfile)
			if err != nil {
				return err
			}
		} else {
			// search for existing CRC in all shread dirs
			var path string
			for _, ct := range ct.Range(&gateway.Gateway, &netprobe.Netprobe, &san.San) {
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

			f, _, err = geneos.OpenSource(path)
			if err != nil {
				return err
			}
		}
		defer f.Close()

		a, err = config.ReadAESValues(f)
		if err != nil {
			return err
		}
		crc, err := a.Checksum()
		if err != nil {
			return err
		}
		crclist = []string{fmt.Sprintf("%08X", crc)}

		// at this point we have an AESValue struct and a CRC to use as a test
		// create 'keyfiles' directory as required
		for _, ct := range ct.Range(&gateway.Gateway, &netprobe.Netprobe, &san.San) {
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
		for _, ct := range ct.Range(&gateway.Gateway, &netprobe.Netprobe, &san.San) {
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
