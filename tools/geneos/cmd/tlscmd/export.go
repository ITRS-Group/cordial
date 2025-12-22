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
	"encoding/pem"
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var exportCmdOutput string
var exportCmdNoRoot bool

func init() {
	tlsCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportCmdOutput, "output", "o", "", "Output destination, default to stdout")
	exportCmd.Flags().BoolVarP(&exportCmdNoRoot, "no-root", "N", false, "Do not include the root CA certificate")

	exportCmd.Flags().SortFlags = false
}

//go:embed _docs/export.md
var exportCmdDescription string

var exportCmd = &cobra.Command{
	Use:                   "export [flags]",
	Short:                 "Export certificates",
	Long:                  exportCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example: `
# export 
$ geneos tls export --output file.pem
`,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		confDir := config.AppConfigDir()
		if confDir == "" {
			return config.ErrNoUserConfigDir
		}
		// gather the rootCA cert, the geneos cert and key
		root, rootFile, err := geneos.ReadRootCert(true)
		if err != nil {
			err = fmt.Errorf("local root certificate (%s) not valid: %w", rootFile, err)
			return
		}
		signer, signerFile, err := geneos.ReadSigningCert(true)
		if err != nil {
			err = fmt.Errorf("local signing root certificate (%s) not valid: %w", signerFile, err)
			return
		}
		signingKey, err := certs.ReadPrivateKey(geneos.LOCAL, path.Join(confDir, geneos.SigningCertBasename+".key"))
		if err != nil {
			return
		}

		var pembytes []byte

		pembytes = pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: signer.Raw,
		})

		if !exportCmdNoRoot {
			pembytes = append(pembytes, pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: root.Raw,
			})...)
		}

		l, _ := signingKey.Open()
		pembytes = append(pembytes, pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: l.Bytes(),
		})...)
		l.Destroy()

		if exportCmdOutput != "" {
			return os.WriteFile(exportCmdOutput, pembytes, 0600)
		}

		fmt.Println(string(pembytes))
		return
	},
}
