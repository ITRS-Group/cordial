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

var aesNewCmdKeyfile config.KeyFile
var aesNewCmdBackupSuffix string
var aesNewCmdImportShared, aesNewCmdSaveUser, aesNewCmdOverwriteKeyfile bool

// var aesDefaultKeyfile = geneos.UserConfigFilePaths("keyfile.aes")[0]

func init() {
	aesCmd.AddCommand(aesNewCmd)

	aesNewCmd.Flags().VarP(&aesNewCmdKeyfile, "keyfile", "k", "Path to key file, defaults to STDOUT")
	aesNewCmd.Flags().BoolVarP(&aesNewCmdSaveUser, "user", "U", false, `New user key file (typically "${HOME}/.config/geneos/keyfile.aes")`)

	aesNewCmd.Flags().StringVarP(&aesNewCmdBackupSuffix, "backup", "b", ".old", "Backup existing keyfile with extension given")

	aesNewCmd.Flags().BoolVarP(&aesNewCmdOverwriteKeyfile, "force", "F", false, "Force overwriting an existing key file")

	aesNewCmd.Flags().BoolVarP(&aesNewCmdImportShared, "shared", "S", false, "Import the keyfile to component shared directories and set on instances")

	aesNewCmd.MarkFlagsMutuallyExclusive("keyfile", "user")
}

//go:embed _docs/new.md
var aesNewCmdDescription string

var aesNewCmd = &cobra.Command{
	Use:   "new [flags] [TYPE] [NAME...]",
	Short: "Create a new key file",
	Long:  aesNewCmdDescription,
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

		if aesNewCmdSaveUser {
			aesNewCmdKeyfile = cmd.DefaultUserKeyfile
		}

		if aesNewCmdKeyfile != "" {
			if _, err = aesNewCmdKeyfile.RollKeyfile(aesNewCmdBackupSuffix); err != nil {
				return
			}
			if kv, err = aesNewCmdKeyfile.Read(); err != nil {
				return
			}
		} else if !aesNewCmdImportShared {
			fmt.Print(kv.String())
		}

		crc, err = kv.Checksum()

		crcstr := fmt.Sprintf("%08X", crc)
		if aesNewCmdKeyfile != "" {
			fmt.Printf("%s created, checksum %s\n", aesNewCmdKeyfile, crcstr)
		}

		if aesNewCmdImportShared {
			ct, args, _ := cmd.CmdArgsParams(command)
			h := geneos.GetHost(cmd.Hostname)

			for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
				for _, h := range h.OrList(geneos.AllHosts()...) {
					if err = instance.SaveKeyFileShared(h, ct, kv); err != nil {
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
	cf.Set("keyfile", instance.SharedPath(c, "keyfiles", params[0]))

	if c.Config().Type == "rc" {
		err = instance.Migrate(c)
	} else {
		err = c.Config().Save(c.Type().String(),
			config.Host(c.Host()),
			config.SaveDir(instance.ParentDirectory(c)),
			config.SetAppName(c.Name()),
		)
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
