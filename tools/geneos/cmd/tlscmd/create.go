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
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var createCmdCN, createCmdDestDir, createCmdSigner string
var createCmdOverwrite bool
var createCmdSANs = SubjectAltNames{}
var createCmdExpiry int

func init() {
	tlsCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&createCmdDestDir, "dest", "D", ".", "Destination `directory` to write certificate chain and private key to.\nFor bundles use a dash '-' for stdout.")

	createCmd.Flags().StringVarP(&createCmdCN, "cname", "c", "", "Common Name for certificate. Defaults to hostname except for --signer.\nIgnored for --signer.")

	createCmd.Flags().StringVarP(&createCmdSigner, "signer", "S", "", "Create a new signer certificate bundle with `NAME`\nas part of the Common Name, typically the hostname\nof the target machine this will be used on.")

	createCmd.Flags().IntVarP(&createCmdExpiry, "expiry", "E", 365, "Certificate expiry duration in days. Ignored for --signer")

	createCmd.Flags().BoolVarP(&createCmdOverwrite, "force", "F", false, "Runs \"tls init\" (but do not replace existing root and signer)\nand overwrite any existing file in the 'out' directory")

	createCmd.Flags().VarP(&createCmdSANs.DNS, "san-dns", "s", "Subject-Alternative-Name DNS Name (repeat as required).\nIgnored for --signer.")
	createCmd.Flags().VarP(&createCmdSANs.IP, "san-ip", "i", "Subject-Alternative-Name IP Address (repeat as required).\nIgnored for --signer.")
	createCmd.Flags().VarP(&createCmdSANs.Email, "san-email", "e", "Subject-Alternative-Name Email Address (repeat as required).\nIgnored for --signer.")
	createCmd.Flags().VarP(&createCmdSANs.URL, "san-url", "u", "Subject-Alternative-Name URL (repeat as required).\nIgnored for --signer.")

	createCmd.Flags().SortFlags = false
}

//go:embed _docs/create.md
var createCmdDescription string

var createCmd = &cobra.Command{
	Use:          "create",
	Short:        "Create standalone certificates and keys",
	Long:         createCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		if createCmdOverwrite {
			if err = geneos.TLSInit(geneos.LOCALHOST, false, initCmdKeyType); err != nil {
				return
			}
		}

		if createCmdCN == "" {
			createCmdCN = geneos.LOCALHOST
		}

		if createCmdSigner != "" {
			if err = CreateSignerCert(createCmdDestDir, createCmdOverwrite, createCmdSigner); err != nil {
				if errors.Is(err, os.ErrExist) && !createCmdOverwrite {
					fmt.Printf("Signer certificate already exists for %q, use --force to overwrite\n", createCmdSigner)
					return nil
				}
				return
			}
			return
		}

		if err = CreateCert(createCmdDestDir, createCmdOverwrite, createCmdExpiry, createCmdCN, createCmdSANs); err != nil {
			if errors.Is(err, os.ErrExist) && !createCmdOverwrite {
				fmt.Printf("Certificate already exists for CN=%q, use --force to overwrite\n", createCmdCN)
				return nil
			}
			return
		}
		return
	},
}

// CreateCert creates a new certificate and private key
//
// skip if certificate exists and is valid
func CreateCert(destination string, overwrite bool, days int, cn string, san SubjectAltNames) (err error) {
	var b bytes.Buffer
	var basepath string

	confDir := config.AppConfigDir()
	if confDir == "" {
		return config.ErrNoUserConfigDir
	}
	if destination != "-" {
		basepath = path.Join(destination, strings.ReplaceAll(cn, " ", "-"))
		if _, err = os.Stat(basepath + certs.PEMExtension); err == nil && !overwrite {
			return os.ErrExist
		}
	}
	template := certs.Template(cn,
		certs.Days(days),
		certs.DNSNames(san.DNS...),
		certs.IPAddresses(san.IP...),
		certs.EmailAddresses(san.Email...),
		certs.URIs(san.URL...),
	)

	signerCert, _, err := geneos.ReadSignerCertificate()
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	signerKey, err := certs.ReadPrivateKey(geneos.LOCAL, path.Join(confDir, geneos.SignerCertBasename+certs.KEYExtension))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	cert, key, err := certs.CreateCertificate(template, signerCert, signerKey)
	if err != nil {
		return
	}

	if _, err = certs.WritePrivateKeyTo(&b, key); err != nil {
		return
	}
	if _, err = certs.WriteCertificatesTo(&b, cert, signerCert); err != nil {
		return
	}

	if destination == "-" {
		fmt.Print(b.String())
		return
	}
	geneos.LOCAL.WriteFile(basepath+".pem", b.Bytes(), 0600)

	fmt.Printf("Certificate and private key created in %q\n", basepath+certs.PEMExtension)
	fmt.Print(string(certs.CertificateComments(cert)))
	return
}

func CreateSignerCert(destination string, overwrite bool, hostname string) (err error) {
	var b bytes.Buffer
	var basepath string

	cn := cordial.ExecutableName() + " " + geneos.SignerCertLabel + " (" + hostname + ")"

	confDir := config.AppConfigDir()
	if confDir == "" {
		return config.ErrNoUserConfigDir
	}
	if destination != "-" {
		basepath = path.Join(destination, strings.ReplaceAll(cn, " ", "-"))
		if _, err = os.Stat(basepath + certs.PEMExtension); err == nil && !overwrite {
			return os.ErrExist
		}
	}
	rootCert, _, err := geneos.ReadRootCertificate()
	if err != nil {
		err = fmt.Errorf("cannot read root CA: %w", err)
		return
	}
	rootKey, _, err := geneos.ReadRootPrivateKey()
	if err != nil {
		err = fmt.Errorf("cannot read root CA private key: %w", err)
		return
	}
	fmt.Fprintf(&b, "# Signer Certificate: %s\n#\n", cn)
	if _, err = certs.WriteNewSignerCertTo(&b, rootCert, rootKey, cn); err != nil {
		return
	}
	if _, err = certs.WriteCertificatesTo(&b, rootCert); err != nil {
		return
	}

	if destination == "-" {
		fmt.Print(b.String())
		return
	}
	geneos.LOCAL.WriteFile(basepath+certs.PEMExtension, b.Bytes(), 0600)
	fmt.Printf("Signer certificate and private key created in %q\n", basepath+certs.PEMExtension)
	return
}

type SubjectAltNames struct {
	DNS   values
	IP    values
	Email values
	URL   values
}

// attribute - name=value
type values []string

const TypesOptionsText = "A type NAME\n(Repeat as required, san only)"

func (i *values) String() string {
	return ""
}

func (i *values) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *values) Type() string {
	return "VALUE"
}
