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
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
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
	Use:                   "export [flags] [TYPE] [NAME...]",
	Short:                 "Export signer certificate and private key",
	Long:                  exportCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example: `
# export 
$ geneos tls export --output file.pem
`,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "false",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names := cmd.ParseTypeNames(command)

		if len(names) > 0 || ct != nil {
			instance.Do(geneos.GetHost(cmd.Hostname), ct, names, exportInstanceCert).Report(os.Stdout)
			return
		}

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

		output := "# Geneos Root and Signer Certificates\n#\n"
		output += string(certs.PrivateKeyComments(signerKey, "Signer Private Key"))
		output += string(pemKey)
		output += string(certs.CertificateComments(signer, "Signer Certificate"))
		output += string(pemSigner)
		output += string(certs.CertificateComments(root, "Root CA Certificate"))

		output += string(pemRoot)

		if exportCmdOutput != "" {
			return os.WriteFile(exportCmdOutput, []byte(output), 0600)
		}

		fmt.Println(output)
		return
	},
}

func exportInstanceCert(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	certChain, err := instance.ReadCertificates(i)
	if err != nil {
		resp.Err = fmt.Errorf("cannot read certificates for %s %q: %w", i.Type(), i.Name(), err)
		return
	}
	key, err := instance.ReadPrivateKey(i)
	if err != nil {
		resp.Err = fmt.Errorf("cannot read private key for %s %q: %w", i.Type(), i.Name(), err)
		return
	}

	keyData, _ := key.Open()
	pemKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyData.Bytes(),
	})
	defer keyData.Destroy()

	output := fmt.Sprintf("# Certificate and Private Key for %s %q\n#\n", i.Type(), i.Name())
	output += string(certs.PrivateKeyComments(key))
	output += string(pemKey)

	for _, cert := range certChain {
		pemCert := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})

		output += string(certs.CertificateComments(cert))
		output += string(pemCert)
	}

	if exportCmdOutput != "" {
		err = os.WriteFile(exportCmdOutput, []byte(output), 0600)
		if err != nil {
			resp.Err = err
			return
		}
		return
	}

	resp.Details = []string{output}
	return
}
