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

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var setCmdKeyfile string
var setCmdCRC, setCmdBackupSuffix string
var setCmdShared, setCmdNoRoll bool

func init() {
	aesCmd.AddCommand(setCmd)

	setCmdKeyfile = string(cmd.DefaultUserKeyfile)

	setCmd.Flags().StringVarP(&setCmdKeyfile, "keyfile", "k", "-", "Key file to use. `PATH|URL|-`\nPath to a local file, a URL or a dash for STDIN.")
	setCmd.Flags().StringVarP(&setCmdCRC, "crc", "c", "", "`CRC` of an existing shared keyfile to use")
	setCmd.Flags().StringVarP(&setCmdBackupSuffix, "backup", "b", "-prev", "Backup any existing keyfile with extension given")

	setCmd.Flags().BoolVarP(&setCmdNoRoll, "no-roll", "N", false, "Do not roll any existing keyfile to previous keyfile setting")
	setCmd.Flags().BoolVarP(&setCmdShared, "shared", "s", false, "Set as a shared keyfile, using the CRC as the file name prefix")

	setCmd.Flags().SortFlags = false
}

//go:embed _docs/set.md
var setCmdDescription string

var setCmd = &cobra.Command{
	Use:          "set [flags] [TYPE] [NAME...]",
	Short:        "Set keyfile for instances",
	Long:         setCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdNoneMeansAll: "explicit",
		cmd.CmdRequireHome:  "true",
		cmd.CmdGlobNames:    "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names := cmd.ParseTypeNames(command)

		h := geneos.GetHost(cmd.Hostname)

		if setCmdShared {
			paths, crc, err := geneos.ImportSharedKey(h, ct, setCmdKeyfile, "Paste AES key file contents, end with newline and CTRL+D:")
			if err != nil {
				return err
			}
			for _, p := range paths {
				fmt.Printf("imported keyfile to %s\n", p)
			}

			instance.Do(h, ct, names, aesSetSharedAESInstance, crc).Write(os.Stdout)

			return nil
		}

		if setCmdCRC != "" && setCmdKeyfile == "" {
			// locate a shared keyfile for each matching host/type and set.
			instance.Do(h, ct, names, aesSetSharedAESInstance, setCmdCRC).Write(os.Stdout)
			return nil
		}

		if setCmdKeyfile != "" {
			kv, err := geneos.ReadKeyValues(setCmdKeyfile, "Paste AES key file contents, end with newline and CTRL+D:")
			if err != nil {
				return err
			}
			for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
				instance.Do(h, ct, names, aesSetAESInstance, kv).Write(os.Stdout)
			}
			return nil
		}

		// locate keyfile by CRC

		return
	},
}

func aesSetAESInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	kv, ok := params[0].(*config.KeyValues)
	if !ok {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	// roll any old file unless `--no-roll`
	if !setCmdNoRoll {
		keyfile, _, err := instance.RollAESKeyFile(i, kv, setCmdBackupSuffix)
		if err != nil {
			resp.Err = err
			return
		}

		pkp := i.Config().GetString("prevkeyfile")
		if pkp != "" {
			resp.Line = fmt.Sprintf("keyfile %s written, existing keyfile renamed to %s and marked a previous keyfile\n", keyfile, pkp)
		} else {
			resp.Line = fmt.Sprintf("keyfile %s written", keyfile)
		}
	} else {
		keyfile, _, err := instance.WriteAESKeyFile(i, kv)
		if err != nil {
			resp.Err = err
			return
		}
		i.Config().Set("keyfile", keyfile)
		resp.Line = fmt.Sprintf("keyfile %s written", keyfile)
	}
	return

}

func aesSetSharedAESInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	crc, ok := params[0].(string)
	if !ok {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	kp := instance.Shared(i, "keyfiles", crc+".aes")
	if st, err := i.Host().Stat(kp); err == nil && st.Mode().IsRegular() {
		// exists
		resp.Completed = append(resp.Completed, "already exists")
	}

	cf := i.Config()

	// roll old file
	if !setCmdNoRoll {
		p := cf.GetString("keyfile")
		if p != "" {
			if p == kp {
				resp.Line = fmt.Sprintf("new and existing keyfile have same CRC. Not updating")
			} else {
				cf.Set("keyfile", kp)
				cf.Set("prevkeyfile", p)
				resp.Line = fmt.Sprintf("keyfile %s set, existing keyfile moved to prevkeyfile", crc)
			}
		} else {
			cf.Set("keyfile", kp)
			resp.Line = fmt.Sprintf("keyfile %s set", crc)
		}
	} else {
		cf.Set("keyfile", kp)
		resp.Line = fmt.Sprintf("keyfile %s set", crc)
	}

	resp.Err = instance.SaveConfig(i)
	return
}
