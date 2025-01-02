/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
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
				for ct := range ct.OrList(geneos.UsesKeyFiles()...) {
					instance.Do(h, ct, names, aesNewSetInstanceShared, crc).Write(os.Stdout)
				}
			}
		} else if newCmdUpdate {
			ct, names := cmd.ParseTypeNames(command)
			h := geneos.GetHost(cmd.Hostname)

			for ct := range ct.OrList(geneos.UsesKeyFiles()...) {
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
