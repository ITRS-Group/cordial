/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package tlscmd

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var createCmdCN string
var createCmdOverwrite bool
var createCmdSANs createCmdSAN

func init() {
	tlsCmd.AddCommand(createCmd)

	hostname, _ := os.Hostname()
	createCmd.Flags().StringVarP(&createCmdCN, "cname", "c", hostname, "Common Name for certificate. Defaults to hostname")
	createCmd.Flags().VarP(&createCmdSANs, "san", "s", "Subject-Alternative-Name (repeat for each one required). Defaults to hostname if none given")
	createCmd.Flags().BoolVarP(&createCmdOverwrite, "force", "F", false, "Force overwrite existing certificate (but not root and intermediate)")
}

//go:embed _docs/create.md
var createCmdDescription string

var createCmd = &cobra.Command{
	Use:          "create",
	Short:        "Create new certificates, independent of instances",
	Long:         createCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		if len(createCmdSANs) == 0 {
			hostname, _ := os.Hostname()
			createCmdSANs = []string{hostname}
		}
		tlsInit(false)
		err = CreateCert(".", createCmdOverwrite, createCmdCN, createCmdSANs...)
		if err != nil {
			if errors.Is(err, host.ErrExists) && !createCmdOverwrite {
				fmt.Printf("Certficate already exists for CN=%q, use --force to overwrite\n", createCmdCN)
				return nil
			}
			return
		}
		fmt.Printf("Certificate and key created for CN=%q\n", createCmdCN)
		return
	},
}

// CreateCert creates a new certificate
//
// this also creates a new private key
//
// skip if certificate exists and is valid
func CreateCert(dir string, overwrite bool, cn string, san ...string) (err error) {
	basepath := path.Join(dir, strings.ReplaceAll(cn, " ", "-"))
	if _, err = os.Stat(basepath + ".pem"); err == nil && !overwrite {
		return host.ErrExists
	}
	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	expires := time.Now().AddDate(1, 0, 0).Truncate(24 * time.Hour)
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

	signingCert, err := instance.ReadSigningCert()
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	signingKey, err := config.ReadPrivateKey(geneos.LOCAL, path.Join(config.AppConfigDir(), geneos.SigningCertFile+".key"))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	cert, key, err := config.CreateCertificateAndKey(&template, signingCert, signingKey, nil)
	if err != nil {
		return
	}

	if err = config.WriteCert(geneos.LOCAL, basepath+".pem", cert); err != nil {
		return
	}

	if err = config.WritePrivateKey(geneos.LOCAL, basepath+".key", key); err != nil {
		return
	}

	fmt.Printf("certificate created for %s (expires %s)\n", basepath, expires.UTC())

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
