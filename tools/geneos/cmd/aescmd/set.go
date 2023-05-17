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
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var aesSetCmdKeyfile config.KeyFile
var aesSetCmdCRC string
var aesSetCmdNoRoll bool

func init() {
	AesCmd.AddCommand(aesSetCmd)

	aesSetCmdKeyfile = cmd.DefaultUserKeyfile
	aesSetCmd.Flags().StringVarP(&aesSetCmdCRC, "crc", "c", "", "CRC of existing component shared keyfile to use (extension optional)")
	aesSetCmd.Flags().VarP(&aesSetCmdKeyfile, "keyfile", "k", "Key file to import and use")
	aesSetCmd.Flags().BoolVarP(&aesSetCmdNoRoll, "noroll", "N", false, "Do not roll any existing keyfile to previous keyfile setting")
}

var aesSetCmd = &cobra.Command{
	Use:   "set [flags] [TYPE] [NAME...]",
	Short: "Set active keyfile for instances",
	Long: strings.ReplaceAll(`
Set a key file for matching instances. The key file is saved to each
matching component's (default: all) shared directory and the
configuration set to that path.

The keyfile can be given as an existing CRC (with or without |.aes|
file extension) with the |--crc|/|-c| option or as a file path (which
can be prefixed |~/| for the user's home directory) or a URL with
|--keyfile|/|-k|. If neither option is given then the user's default
keyfile is used, if it exists.

If the |--crc|/|-c| flag is given and it matches an existing
keyfile in the component shared directory then that is used for
matching instances. When TYPE is not given, the keyfile will also be
copied to the shared directories of other component types if not
already present.

The |--keyfile|/|-k| flag value can be a local file (including a
prefix of |~/| to represent the home directory), a URL or a dash |-|
for STDIN. The given keyfile is evaluated and its CRC32 checksum
checked against existing keyfiles in the matching component shared
directories. The keyfile is only saved if one with the same checksum
does not already exist. 

For each instance any existing |keyfile| path is copied to a
|prevkeyfile| setting, unless the |--noroll|/|-N| option if given, to
support key file updating in Geneos GA6 and above.

Key files are only set on components that support them.

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
		var kv *config.KeyValues

		var keyfile config.KeyFile

		// if no CRC given then use the keyfile or the user's default one
		if aesSetCmdCRC == "" {
			keyfile = aesSetCmdKeyfile
		} else {
			// search for existing CRC in all shared dirs
			var path string
			crcfile := aesSetCmdCRC
			if filepath.Ext(crcfile) != "aes" {
				crcfile += ".aes"
			}
			for _, ct := range ct.OrList(componentsWithKeyfiles...) {
				path = ct.SharedPath(geneos.LOCAL, "keyfiles", crcfile)
				log.Debug().Msgf("looking for keyfile %s", path)
				if _, err := geneos.LOCAL.Stat(path); err == nil {
					break
				}
				path = ""
			}

			if path == "" {
				return fmt.Errorf("keyfile with CRC %q not found", aesSetCmdCRC)
			}
			keyfile = config.KeyFile(path)
		}

		crc, _, err := keyfile.Check(false)
		if err != nil {
			return err
		}
		crclist = []string{fmt.Sprintf("%08X", crc)}
		kv, err = keyfile.Read()
		if err != nil {
			return
		}

		// at this point we have a KeyValues and a CRC to use as a test
		// create 'keyfiles' directory as required
		for _, ct := range ct.OrList(componentsWithKeyfiles...) {
			for _, h := range geneos.AllHosts() {
				// only import if it is not found
				path := ct.SharedPath(h, "keyfiles", crclist[0]+".aes")
				if _, err := h.Stat(path); err != nil && errors.Is(err, fs.ErrNotExist) {
					aesImportSave(ct, h, kv)
				} else if err == nil {
					log.Debug().Msgf("not importing existing %q CRC named keyfile on %s", crclist[0], h)
				}
			}
		}

		// params[0] is the CRC
		for _, ct := range ct.OrList(componentsWithKeyfiles...) {
			instance.ForAll(ct, cmd.Hostname, aesSetAESInstance, args, crclist)
		}
		return nil
	},
}

func aesSetAESInstance(c geneos.Instance, params []string) (err error) {
	cf := c.Config()

	path := instance.SharedPath(c, "keyfiles", params[0]+".aes")

	// roll old file
	if !aesSetCmdNoRoll {
		p := cf.GetString("keyfile")
		if p != "" {
			if p == path {
				fmt.Printf("%s: new and existing keyfile have same CRC. Not updating\n", c)
			} else {
				cf.Set("keyfile", path)
				cf.Set("prevkeyfile", p)
				fmt.Printf("%s keyfile %s set, existing keyfile moved to prevkeyfile\n", c, params[0])
			}
		} else {
			cf.Set("keyfile", path)
			fmt.Printf("%s keyfile %s set\n", c, params[0])
		}
	} else {
		cf.Set("keyfile", path)
		fmt.Printf("%s keyfile %s set\n", c, params[0])
	}

	if cf.Type == "rc" {
		err = instance.Migrate(c)
	} else {
		err = cf.Save(c.Type().String(),
			config.Host(c.Host()),
			config.SaveDir(c.Type().InstancesDir(c.Host())),
			config.SetAppName(c.Name()),
		)
	}

	return
}
