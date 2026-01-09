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
	"crypto/sha1"
	"crypto/sha256"
	_ "embed"
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"time"

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
	exportCmd.Flags().MarkDeprecated("no-root", "use --root instead to include the root CA certificate")

	exportCmd.Flags().SortFlags = false
}

//go:embed _docs/export.md
var exportCmdDescription string

var exportCmd = &cobra.Command{
	Use:                   "export [flags]",
	Short:                 "Export signer certificate and private key",
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
		root, rootFile, err := geneos.ReadRootCertificate()
		if err != nil {
			err = fmt.Errorf("local root certificate (%q) not valid: %w", rootFile, err)
			return
		}
		signer, signerFile, err := geneos.ReadSignerCertificate()
		if err != nil {
			err = fmt.Errorf("local signer certificate (%q) not valid: %w", signerFile, err)
			return
		}
		signerKey, err := certs.ReadPrivateKey(geneos.LOCAL, path.Join(confDir, geneos.SigningCertBasename+".key"))
		if err != nil {
			return
		}

		key, _ := signerKey.Open()
		pemKey := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: key.Bytes(),
		})
		defer key.Destroy()

		pemSigner := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: signer.Raw,
		})

		pemRoot := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: root.Raw,
		})

		output := "# Exported Geneos Root and Signer Certificates and Private Key\n"
		output += "#\n"
		output += "# Signer Private Key\n#\n"
		output += "#   Key Type: " + string(certs.PrivateKeyType(signerKey)) + "\n#\n"
		output += string(pemKey)
		output += "# Signer Certificate\n#\n"
		output += "#   Subject: " + signer.Subject.String() + "\n"
		output += "#    Issuer: " + signer.Issuer.String() + "\n"
		output += "#   Expires: " + signer.NotAfter.Format(time.RFC3339) + "\n"
		output += "#    Serial: " + signer.SerialNumber.String() + "\n"
		output += "#      SHA1: " + fmt.Sprintf("%X", sha1.Sum(signer.Raw)) + "\n"
		output += "#    SHA256: " + fmt.Sprintf("%X", sha256.Sum256(signer.Raw)) + "\n#\n"
		output += string(pemSigner)
		output += "# Root CA Certificate\n#\n"
		output += "#   Subject: " + root.Subject.String() + "\n"
		output += "#   Expires: " + root.NotAfter.Format(time.RFC3339) + "\n"
		output += "#    Serial: " + root.SerialNumber.String() + "\n"
		output += "#      SHA1: " + fmt.Sprintf("%X", sha1.Sum(root.Raw)) + "\n"
		output += "#    SHA256: " + fmt.Sprintf("%X", sha256.Sum256(root.Raw)) + "\n#\n"

		output += string(pemRoot)

		if exportCmdOutput != "" {
			return os.WriteFile(exportCmdOutput, []byte(output), 0600)
		}

		fmt.Println(output)
		return
	},
}
