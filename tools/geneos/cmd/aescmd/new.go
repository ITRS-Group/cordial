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

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var newCmdKeyfile config.KeyFile
var newCmdBackupSuffix string
var newCmdShared, newCmdSaveUser, newCmdOverwriteKeyfile, newCmdUpdate bool

// var aesDefaultKeyfile = geneos.UserConfigFilePaths("keyfile.aes")[0]

func init() {
	aesCmd.AddCommand(newCmd)

	newCmd.Flags().VarP(&newCmdKeyfile, "keyfile", "k", "Path to key file, defaults to STDOUT")
	newCmd.Flags().BoolVar(&newCmdSaveUser, "user", false, `Write to user key file (typically "${HOME}/.config/geneos/keyfile.aes")`)
	newCmd.Flags().StringVarP(&newCmdBackupSuffix, "backup", "b", "-prev", "Backup existing keyfile with extension given")

	newCmd.Flags().BoolVarP(&newCmdOverwriteKeyfile, "force", "F", false, "Force overwriting an existing key file")

	newCmd.Flags().BoolVarP(&newCmdShared, "shared", "S", false, "Write the keyfile to matching component shared directories")
	newCmd.Flags().BoolVarP(&newCmdUpdate, "update", "U", false, "Update keyfile settings on matching instances")

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
		// create new key values, may be overwritten later
		kv := config.NewRandomKeyValues()

		if newCmdSaveUser {
			newCmdKeyfile = cmd.DefaultUserKeyfile
		}

		if newCmdKeyfile != "" {
			if _, err = newCmdKeyfile.CreateWithBackup(host.Localhost, newCmdBackupSuffix); err != nil {
				return
			}
			if kv, err = newCmdKeyfile.Read(host.Localhost); err != nil {
				return
			}
			fmt.Printf("keyfile written to %s\n", newCmdKeyfile)
		}

		if newCmdShared {
			log.Debug().Msg("new shared keyfile")
			ct, names := cmd.ParseTypeNames(command)
			h := geneos.GetHost(cmd.Hostname)

			paths, _, err := geneos.WriteSharedKeyValues(h, ct, kv)
			if err != nil {
				return err
			}
			for _, p := range paths {
				fmt.Printf("keyfile written to %s\n", p)
			}

			if newCmdUpdate {
				crc, err := kv.ChecksumString()
				if err != nil {
					return err
				}
				for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
					instance.Do(h, ct, names, aesNewSetInstanceShared, crc).Write(os.Stdout)
				}
			}
		} else if newCmdUpdate {
			ct, names := cmd.ParseTypeNames(command)
			h := geneos.GetHost(cmd.Hostname)

			for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
				instance.Do(h, ct, names, aesNewSetInstance, kv).Write(os.Stdout)
			}
		} else {
			fmt.Print(kv.String())
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

	kv, ok := params[0].(*config.KeyValues)
	if !ok {
		resp.Err = fmt.Errorf("wrong parameter type %T", kv)
	}

	keyfile, _, err := instance.RollAESKeyFile(i, kv, newCmdBackupSuffix)
	if err != nil {
		resp.Err = err
		return
	}

	resp.Line = fmt.Sprintf("keyfile written to %q", keyfile)
	return
}

func aesNewSetInstanceShared(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	crc, ok := params[0].(string)
	if !ok {
		resp.Err = fmt.Errorf("wrong parameter type %T", crc)
		return
	}
	kp := instance.Shared(i, "keyfiles", crc+".aes")

	keyfile := config.KeyFile(kp)
	kv, err := keyfile.Read(i.Host())
	if err != nil {
		resp.Err = err
		return
	}

	instance.RollAESKeyFile(i, kv, "-prev")
	pkp := i.Config().GetString("prevkeyfile")
	if pkp != "" {
		resp.Line = fmt.Sprintf("keyfile %q written, existing keyfile renamed to %q and marked a previous keyfile", keyfile, pkp)
	} else {
		resp.Line = fmt.Sprintf("keyfile %q written\n", keyfile)
	}
	return
}
