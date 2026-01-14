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
	"crypto/x509"
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
	exportCmd.Flags().MarkDeprecated("no-root", "root CA should always be in the exported bnundle")

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
geneos tls export --output file.pem
geneos tls export gateway mygateway
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
			err = fmt.Errorf("root certificate expected at %q is not valid: %w", rootFile, err)
			return
		}
		pemRoot := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: root.Raw,
		})

		signer, signerFile, err := geneos.ReadSignerCertificate()
		if err != nil {
			err = fmt.Errorf("signer certificate expected at %q is not valid: %w", signerFile, err)
			return
		}
		pemSigner := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: signer.Raw,
		})

		signerKey, err := certs.ReadPrivateKey(geneos.LOCAL, path.Join(confDir, geneos.SigningCertBasename+".key"))
		if err != nil {
			err = fmt.Errorf("signer private key expected at %q cannot be read: %w", path.Join(confDir, geneos.SigningCertBasename+".key"), err)
			return
		}
		key, _ := signerKey.Open()
		pemKey := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: key.Bytes(),
		})
		defer key.Destroy()

		output := []byte("# Geneos Root and Signer Certificates\n#\n")
		output = append(output, certs.PrivateKeyComments(signerKey, "Signer Private Key")...)
		output = append(output, pemKey...)
		output = append(output, certs.CertificateComments(signer, "Signer Certificate")...)
		output = append(output, pemSigner...)
		output = append(output, certs.CertificateComments(root, "Root CA Certificate")...)
		output = append(output, pemRoot...)

		if exportCmdOutput != "" {
			return os.WriteFile(exportCmdOutput, output, 0600)
		}

		fmt.Println(string(output))
		return
	},
}

func exportInstanceCert(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	h := i.Host()

	instanceCertChain, err := instance.ReadCertificates(i)
	if err != nil {
		resp.Err = fmt.Errorf("cannot read certificates for %s %q: %w", i.Type(), i.Name(), err)
		return
	}

	// build and test trust chain
	rootPool, ok := certs.ReadCACertPool(h, geneos.PathToCABundle(h))
	if !ok {
		// if there is no ca-bundle file on the host, try local root CA
		confDir := config.AppConfigDir()
		if confDir == "" {
			resp.Err = config.ErrNoUserConfigDir
			return
		}
		rootPool, ok = certs.ReadCACertPool(geneos.LOCAL, path.Join(confDir, geneos.RootCABasename+".pem"))
		if !ok {
			resp.Err = fmt.Errorf("no CA certificates found to verify %s %q", i.Type(), i.Name())
			return
		}
	}

	intermediatePool := x509.NewCertPool()
	for _, cert := range instanceCertChain[1:] {
		intermediatePool.AddCert(cert)
	}

	opts := x509.VerifyOptions{
		Roots:         rootPool,
		Intermediates: intermediatePool,
	}

	validatedCertChain, err := instanceCertChain[0].Verify(opts)
	if err != nil || len(validatedCertChain) == 0 {
		resp.Err = fmt.Errorf("cannot verify certificate chain for %s %q: %w", i.Type(), i.Name(), err)
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

	output := []byte(fmt.Sprintf("# Certificate and Private Key for %s %q\n#\n", i.Type(), i.Name()))
	output = append(output, certs.PrivateKeyComments(key)...)
	output = append(output, pemKey...)

	for _, cert := range validatedCertChain[0] {
		pemCert := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})

		output = append(output, certs.CertificateComments(cert)...)
		output = append(output, pemCert...)
	}

	if exportCmdOutput != "" {
		err = os.WriteFile(exportCmdOutput, output, 0600)
		if err != nil {
			resp.Err = err
			return
		}
		return
	}

	resp.Details = []string{"\n", string(output)}
	return
}
