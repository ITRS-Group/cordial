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
	aesCmd.AddCommand(aesSetCmd)

	aesSetCmdKeyfile = cmd.DefaultUserKeyfile
	aesSetCmd.Flags().StringVarP(&aesSetCmdCRC, "crc", "c", "", "CRC of existing component shared keyfile to use (extension optional)")
	aesSetCmd.Flags().VarP(&aesSetCmdKeyfile, "keyfile", "k", "Key file to import and use")
	aesSetCmd.Flags().BoolVarP(&aesSetCmdNoRoll, "noroll", "N", false, "Do not roll any existing keyfile to previous keyfile setting")
}

//go:embed _docs/set.md
var aesSetCmdDescription string

var aesSetCmd = &cobra.Command{
	Use:          "set [flags] [TYPE] [NAME...]",
	Short:        "Set active keyfile for instances",
	Long:         aesSetCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args := cmd.CmdArgs(command)

		h := geneos.GetHost(cmd.Hostname)

		crc32, created, err := aesSetCmdKeyfile.Check(true)
		if err != nil {
			return
		}

		if created {
			fmt.Printf("%s created, checksum %08X\n", aesSetCmdKeyfile, crc32)
		}

		crc, err := instance.UseKeyFile(h, ct, aesSetCmdKeyfile, aesSetCmdCRC)
		if err != nil {
			return
		}
		crc = instance.KeyFileNormalise(crc)
		crclist := []string{crc}
		// params[0] is the CRC
		for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
			instance.ForAll(ct, cmd.Hostname, aesSetAESInstance, args, crclist)
		}
		return nil
	},
}

func aesSetAESInstance(c geneos.Instance, params []string) (err error) {
	cf := c.Config()

	path := instance.SharedPath(c, "keyfiles", params[0])

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
			config.SaveDir(instance.ParentDirectory(c)),
			config.SetAppName(c.Name()),
		)
	}

	return
}
