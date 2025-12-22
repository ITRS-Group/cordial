/*
Copyright © 2025 ITRS Group

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

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

type trustCmdRemoveCerts []string

var trustCmdRemove *trustCmdRemoveCerts

func (r *trustCmdRemoveCerts) String() string {
	return ""
}

func (r *trustCmdRemoveCerts) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func (r *trustCmdRemoveCerts) Type() string {
	return "CN/SHA1/SHA256"
}

func init() {
	tlsCmd.AddCommand(trustCmd)

	trustCmdRemove = new(trustCmdRemoveCerts)

	trustCmd.Flags().VarP(trustCmdRemove, "remove", "r", "Remove trusted certificates instead of adding them\nThe argument is either the certificate's CommonName or SHA1 or SHA256 fingerprint\nThis flag can be specified multiple times to remove multiple certificates")

	trustCmd.Flags().SortFlags = false
}

//go:embed _docs/trust.md
var trustCmdDescription string

var trustCmd = &cobra.Command{
	Use:                   "trust [flags] [PATH...]",
	Short:                 "Import trusted certificates",
	Long:                  trustCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               ``,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		return
	},
}
