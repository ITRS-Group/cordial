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
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var newCmdKeyfile config.KeyFile
var newCmdBackupSuffix string
var newCmdImportShared, newCmdSaveUser, newCmdOverwriteKeyfile, newCmdImportUpdate bool

// var aesDefaultKeyfile = geneos.UserConfigFilePaths("keyfile.aes")[0]

func init() {
	aesCmd.AddCommand(newCmd)

	newCmd.Flags().VarP(&newCmdKeyfile, "keyfile", "k", "Path to key file, defaults to STDOUT")
	newCmd.Flags().BoolVarP(&newCmdSaveUser, "user", "U", false, `Write to user key file (typically "${HOME}/.config/geneos/keyfile.aes")`)
	newCmd.Flags().StringVarP(&newCmdBackupSuffix, "backup", "b", ".old", "Backup existing keyfile with extension given")
	newCmd.Flags().BoolVarP(&newCmdOverwriteKeyfile, "force", "F", false, "Force overwriting an existing key file")
	newCmd.Flags().BoolVarP(&newCmdImportShared, "shared", "S", false, "Import the keyfile to component shared directories")
	newCmd.Flags().BoolVar(&newCmdImportUpdate, "update", false, "Update shared keyfile on matching instances")

	newCmd.MarkFlagsMutuallyExclusive("keyfile", "user")
	newCmd.Flags().SortFlags = false
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
		cmd.AnnotationWildcard:  "true",
		cmd.AnnotationNeedsHome: "true",
		cmd.AnnotationExpand:    "true",
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
		if err != nil {
			return
		}

		crcstr := fmt.Sprintf("%08X", crc)
		if newCmdKeyfile != "" {
			fmt.Printf("%s created, checksum %s\n", newCmdKeyfile, crcstr)
		}

		if newCmdImportShared {
			ct, names := cmd.ParseTypeNames(command)
			h := geneos.GetHost(cmd.Hostname)

			crc32, err := geneos.ImportSharedKeyValues(h, ct, kv)
			if err != nil {
				return err
			}
			fmt.Printf("imported keyfile with CRC %08X\n", crc32)

			if newCmdImportUpdate {
				for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
					instance.Do(h, ct, names, aesNewSetInstance, crcstr+".aes").Write(os.Stdout)
				}
			}
		}
		return
	},
}

func aesNewSetInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	keyfile, ok := params[0].(string)
	if !ok {
		panic("wrong type")
	}
	var rolled bool
	cf := i.Config()

	// roll old file
	// XXX - check keyfile still exists, do not update if not
	p := cf.GetString("keyfile")
	if p != "" {
		cf.Set("prevkeyfile", p)
		rolled = true
	}
	cf.Set("keyfile", instance.Shared(i, "keyfiles", keyfile))

	if cf.Type == "rc" {
		resp.Err = instance.Migrate(i)
	} else {
		resp.Err = instance.SaveConfig(i)
	}
	if resp.Err != nil {
		return
	}

	resp.Completed = append(resp.Completed, fmt.Sprintf("keyfile %s set", keyfile))
	if rolled {
		resp.Completed = append(resp.Completed, "existing keyfile moved to prevkeyfile")
	}
	return
}
