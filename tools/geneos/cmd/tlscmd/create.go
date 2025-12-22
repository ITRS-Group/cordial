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
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var createCmdCN, createCmdDest string
var createCmdOverwrite bool
var createCmdSANs createCmdSAN
var createCmdDays int

func init() {
	tlsCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&createCmdDest, "out", "o", ".", "Output `directory` to write to.\nFor bundles use a dash '-' for stdout.")

	createCmd.Flags().StringVarP(&createCmdCN, "cname", "c", "", "Common Name for certificate. Defaults to hostname")
	createCmd.Flags().VarP(&createCmdSANs, "san", "s", "Subject-Alternative-Name (repeat for each one required). Defaults to hostname if none given")
	createCmd.Flags().BoolVarP(&createCmdOverwrite, "force", "F", false, "Run \"tls init\" and force overwrite any existing file in 'dest'")
	createCmd.Flags().IntVarP(&createCmdDays, "days", "D", 365, "Certificate duration in days")

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
		if len(createCmdSANs) == 0 {
			hostname, _ := os.Hostname()
			createCmdSANs = []string{hostname}
		}
		if createCmdOverwrite {
			if err = geneos.TLSInit(false, initCmdKeyType); err != nil {
				return
			}
		}
		if createCmdCN == "" {
			createCmdCN, err = os.Hostname()
			if err != nil {
				return err
			}
		}

		if createCmdDest == "-" {
			fmt.Println("Console output only valid for bundles")
			return nil
		}
		err = CreateCert(createCmdDest, createCmdOverwrite, 24*time.Hour*time.Duration(createCmdDays), createCmdCN, createCmdSANs...)
		if err != nil {
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
func CreateCert(destination string, overwrite bool, duration time.Duration, cn string, san ...string) (err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		return config.ErrNoUserConfigDir
	}
	basepath := path.Join(destination, strings.ReplaceAll(cn, " ", "-"))
	if _, err = os.Stat(basepath + ".pem"); err == nil && !overwrite {
		return os.ErrExist
	}
	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	if duration == 0 {
		duration = 365 * 24 * time.Hour
	}
	expires := time.Now().Add(duration)
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:      time.Now().Add(-60 * time.Second),
		NotAfter:       expires,
		KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		MaxPathLenZero: true,
		DNSNames:       san,
		// IPAddresses:    []net.IP{net.ParseIP("127.0.0.1")},
	}

	signingCert, _, err := geneos.ReadSigningCert()
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	signingKey, err := config.ReadPrivateKey(geneos.LOCAL, path.Join(confDir, geneos.SigningCertBasename+".key"))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	cert, key, err := config.CreateCertificateAndKey(&template, signingCert, signingKey)
	if err != nil {
		return
	}

	if err = config.WriteCertificates(geneos.LOCAL, basepath+".pem", cert, signingCert); err != nil {
		return
	}

	if err = config.WritePrivateKey(geneos.LOCAL, basepath+".key", key); err != nil {
		return
	}

	fmt.Printf("certificate created for %s\n", basepath)
	fmt.Printf("            Expiry: %s\n", expires.UTC().Format(time.RFC3339))
	fmt.Printf("  SHA1 Fingerprint: %X\n", sha1.Sum(cert.Raw))
	fmt.Printf("SHA256 Fingerprint: %X\n", sha256.Sum256(cert.Raw))

	return
}

type createCmdSAN []string

func (san *createCmdSAN) Set(name string) (err error) {
	*san = append(*san, name)
	return
}

func (san *createCmdSAN) String() string {
	return ""
}

func (san *createCmdSAN) Type() string {
	return "SAN"
}
