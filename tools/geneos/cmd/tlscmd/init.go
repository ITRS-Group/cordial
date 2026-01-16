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

package tlscmd

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var initCmdOverwrite bool
var initCmdKeyType certs.KeyType

func init() {
	tlsCmd.AddCommand(initCmd)

	initCmd.Flags().VarP(&initCmdKeyType, "keytype", "K", "Key type for root. One of ecdh, ecdsa, ed25519 or rsa")
	initCmd.Flags().BoolVarP(&initCmdOverwrite, "force", "F", false, "Overwrite any existing root and signer certificates")

	initCmd.Flags().SortFlags = false
}

//go:embed _docs/init.md
var initCmdDescription string

var initCmd = &cobra.Command{
	Use:                   "init",
	Short:                 "Initialise the TLS environment",
	Long:                  initCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		return geneos.TLSInit(geneos.LOCALHOST, initCmdOverwrite, initCmdKeyType)
	},
}
