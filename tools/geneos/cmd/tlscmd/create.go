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
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var createCmdCN, createCmdDestDir string
var createCmdOverwrite bool
var createCmdSANs certSANs
var createCmdDays int

type certSANs struct {
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

func init() {
	tlsCmd.AddCommand(createCmd)

	createCmdSANs = certSANs{}

	createCmd.Flags().StringVarP(&createCmdDestDir, "out", "o", ".", "Destination `directory` to write certificate chain and private key to.\nFor bundles use a dash '-' for stdout.")

	createCmd.Flags().StringVarP(&createCmdCN, "cname", "c", "", "Common Name for certificate. Defaults to hostname")

	createCmd.Flags().IntVarP(&createCmdDays, "days", "D", 365, "Certificate duration in days")

	createCmd.Flags().VarP(&createCmdSANs.DNS, "san-dns", "s", "Subject-Alternative-Name DNS Name (repeat as required).")
	createCmd.Flags().VarP(&createCmdSANs.IP, "san-ip", "i", "Subject-Alternative-Name IP Address (repeat as required).")
	createCmd.Flags().VarP(&createCmdSANs.Email, "san-email", "e", "Subject-Alternative-Name Email Address (repeat as required).")
	createCmd.Flags().VarP(&createCmdSANs.URL, "san-url", "u", "Subject-Alternative-Name URL (repeat as required).")

	createCmd.Flags().BoolVarP(&createCmdOverwrite, "force", "F", false, "Runs \"tls init\" (but do not replace existing root and signer)\nand overwrite any existing file in the 'out' directory")

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
			if err = geneos.TLSInit(false, initCmdKeyType); err != nil {
				return
			}
		}

		if createCmdCN == "" {
			if len(createCmdSANs.DNS) > 0 {
				createCmdCN = createCmdSANs.DNS[0]
			} else {
				createCmdCN, err = os.Hostname()
				if err != nil {
					return err
				}
			}
		}

		if createCmdDestDir == "-" {
			fmt.Println("Console output only valid for bundles")
			return nil
		}

		if err = CreateCert(createCmdDestDir, createCmdOverwrite, createCmdDays, createCmdCN, createCmdSANs); err != nil {
			if errors.Is(err, os.ErrExist) && !createCmdOverwrite {
				fmt.Printf("Certificate already exists for CN=%q, use --force to overwrite\n", createCmdCN)
				return nil
			}
			return
		}
		fmt.Printf("Certificate and key created for CN=%q\n", createCmdCN)
		return
	},
}

// CreateCert creates a new certificate and private key
//
// skip if certificate exists and is valid
func CreateCert(destination string, overwrite bool, days int, cn string, san certSANs) (err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		return config.ErrNoUserConfigDir
	}
	basepath := path.Join(destination, strings.ReplaceAll(cn, " ", "-"))
	if _, err = os.Stat(basepath + ".pem"); err == nil && !overwrite {
		return os.ErrExist
	}
	template := certs.Template(cn,
		certs.Days(days),
		certs.DNSNames(san.DNS...),
		certs.IPAddresses(san.IP...),
		certs.EmailAddresses(san.Email...),
		certs.URIs(san.URL...),
	)
	expires := template.NotAfter

	signerCert, _, err := geneos.ReadSignerCertificate()
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	signerKey, err := certs.ReadPrivateKey(geneos.LOCAL, path.Join(confDir, geneos.SigningCertBasename+".key"))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	cert, key, err := certs.CreateCertificateAndKey(template, signerCert, signerKey)
	if err != nil {
		return
	}

	if err = certs.WriteCertificates(geneos.LOCAL, basepath+".pem", cert, signerCert); err != nil {
		return
	}

	if err = certs.WritePrivateKey(geneos.LOCAL, basepath+".key", key); err != nil {
		return
	}

	fmt.Printf("certificate created for %s\n", basepath)
	fmt.Printf("            Expiry: %s\n", expires.UTC().Format(time.RFC3339))
	fmt.Printf("  SHA1 Fingerprint: %X\n", sha1.Sum(cert.Raw))
	fmt.Printf("SHA256 Fingerprint: %X\n", sha256.Sum256(cert.Raw))

	return
}
