/*
Copyright © 2023 ITRS Group

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

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var passwordCmdString = &config.Plaintext{}
var passwordCmdSource string

func init() {
	aesCmd.AddCommand(passwordCmd)

	passwordCmd.Flags().VarP(passwordCmdString, "password", "p", "A plaintext password")
	passwordCmd.Flags().StringVarP(&passwordCmdSource, "source", "s", "", "External source for plaintext `PATH|URL|-`")
}

//go:embed _docs/password.md
var passwordCmdDescription string

var passwordCmd = &cobra.Command{
	Use:          "password [flags]",
	Short:        "Encode a password with an AES256 key file",
	Long:         passwordCmdDescription,
	Aliases:      []string{"passwd"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdNoneMeansAll: "false",
		cmd.CmdRequireHome:  "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var plaintext *config.Plaintext

		crc, created, err := cmd.DefaultUserKeyfile.ReadOrCreate(host.Localhost, true)
		if err != nil {
			return
		}

		if created {
			fmt.Printf("%s created, checksum %08X\n", cmd.DefaultUserKeyfile, crc)
		}

		if !passwordCmdString.IsNil() {
			plaintext = passwordCmdString
		} else if passwordCmdSource != "" {
			var pt []byte
			pt, err = geneos.ReadAll(passwordCmdSource)
			if err != nil {
				return
			}
			plaintext = config.NewPlaintext(pt)
		} else {
			plaintext, err = config.ReadPasswordInput(true, 3)
			if err != nil {
				return
			}
		}
		e, err := cmd.DefaultUserKeyfile.Encode(host.Localhost, plaintext, true)
		if err != nil {
			return err
		}
		fmt.Println(e)
		return nil
	},
}
