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

var newCmdKeyfile config.KeyFile
var newCmdBackupSuffix string
var newCmdImportShared, newCmdSaveUser, newCmdOverwriteKeyfile bool

// var aesDefaultKeyfile = geneos.UserConfigFilePaths("keyfile.aes")[0]

func init() {
	aesCmd.AddCommand(newCmd)

	newCmd.Flags().VarP(&newCmdKeyfile, "keyfile", "k", "Path to key file, defaults to STDOUT")
	newCmd.Flags().BoolVarP(&newCmdSaveUser, "user", "U", false, `New user key file (typically "${HOME}/.config/geneos/keyfile.aes")`)

	newCmd.Flags().StringVarP(&newCmdBackupSuffix, "backup", "b", ".old", "Backup existing keyfile with extension given")

	newCmd.Flags().BoolVarP(&newCmdOverwriteKeyfile, "force", "F", false, "Force overwriting an existing key file")

	newCmd.Flags().BoolVarP(&newCmdImportShared, "shared", "S", false, "Import the keyfile to component shared directories and set on instances")

	newCmd.MarkFlagsMutuallyExclusive("keyfile", "user")
}

//go:embed _docs/new.md
var newCmdDescription string

var newCmd = &cobra.Command{
	Use:   "new [flags] [TYPE] [NAME...]",
	Short: "Create a new key file",
	Long:  newCmdDescription,
	Example: `
geneos aes new
geneos aes new -F ~/keyfile.aes
geneos aes new -S gateway
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var crc uint32

		kv := config.NewRandomKeyValues()

		if newCmdSaveUser {
			newCmdKeyfile = cmd.DefaultUserKeyfile
		}

		if newCmdKeyfile != "" {
			if _, err = newCmdKeyfile.RollKeyfile(newCmdBackupSuffix); err != nil {
				return
			}
			if kv, err = newCmdKeyfile.Read(); err != nil {
				return
			}
		} else if !newCmdImportShared {
			fmt.Print(kv.String())
		}

		crc, err = kv.Checksum()

		crcstr := fmt.Sprintf("%08X", crc)
		if newCmdKeyfile != "" {
			fmt.Printf("%s created, checksum %s\n", newCmdKeyfile, crcstr)
		}

		if newCmdImportShared {
			ct, args, _ := cmd.CmdArgsParams(command)
			h := geneos.GetHost(cmd.Hostname)

			for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
				for _, h := range h.OrList(geneos.AllHosts()...) {
					if err = geneos.SaveKeyFileShared(h, ct, kv); err != nil {
						return
					}
					params := []string{crcstr + ".aes"}
					instance.ForAll(ct, cmd.Hostname, aesNewSetInstance, args, params)
				}
			}

			return
		}
		return
	},
}

func aesNewSetInstance(c geneos.Instance, params []string) (err error) {
	var rolled bool
	cf := c.Config()

	// roll old file
	// XXX - check keyfile still exists, do not update if not
	p := cf.GetString("keyfile")
	if p != "" {
		cf.Set("prevkeyfile", p)
		rolled = true
	}
	cf.Set("keyfile", instance.Shared(c, "keyfiles", params[0]))

	if cf.Type == "rc" {
		err = instance.Migrate(c)
	} else {
		err = instance.SaveConfig(c)
	}
	if err != nil {
		return
	}

	fmt.Printf("%s keyfile %s set", c, params[0])
	if rolled {
		fmt.Printf(", existing keyfile moved to prevkeyfile\n")
	} else {
		fmt.Println()
	}
	return
}
