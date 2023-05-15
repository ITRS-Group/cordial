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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

func init() {
	TLSCmd.AddCommand(tlsInitCmd)
	tlsInitCmd.Flags().SortFlags = false
}

var tlsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise the TLS environment",
	Long: strings.ReplaceAll(`Initialise the TLS environment by creating a self-signed
root certificate to act as a CA and a signing certificate signed
by the root. Any instances will have certificates created for
them but configurations will not be rebuilt.
`, "|", "`"),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		// _, _, params := processArgsParams(cmd)
		return tlsInit()
	},
}

// create the tls/ directory in Geneos and a CA / DCA as required
//
// later options to allow import of a DCA
//
// This is also called from `init`
func tlsInit() (err error) {
	tlsPath := filepath.Join(geneos.Root(), "tls")
	// directory permissions do not need to be restrictive
	err = geneos.LOCAL.MkdirAll(tlsPath, 0775)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if err := newRootCA(tlsPath); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if err := newIntrCA(tlsPath); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return tlsSync()
}

func newRootCA(dir string) (err error) {
	// create rootCA.pem / rootCA.key
	var cert *x509.Certificate
	rootCertPath := filepath.Join(dir, geneos.RootCAFile+".pem")
	rootKeyPath := filepath.Join(dir, geneos.RootCAFile+".key")

	if _, err = instance.ReadRootCert(); err == nil {
		log.Error().Msgf("%s already exists", geneos.RootCAFile)
		return
	}
	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "geneos root CA",
		},
		NotBefore:             time.Now().Add(-60 * time.Second),
		NotAfter:              time.Now().AddDate(10, 0, 0).Truncate(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            2,
	}

	cert, key, err := instance.CreateCertKey(template, template, nil, nil)
	if err != nil {
		return
	}

	if err = geneos.LOCAL.WriteCert(rootCertPath, cert); err != nil {
		return
	}
	if err = geneos.LOCAL.WriteKey(rootKeyPath, key); err != nil {
		return
	}
	fmt.Printf("CA certificate created for %s\n", geneos.RootCAFile)

	return
}

func newIntrCA(dir string) (err error) {
	var cert *x509.Certificate
	intrCertPath := filepath.Join(dir, geneos.SigningCertFile+".pem")
	intrKeyPath := filepath.Join(dir, geneos.SigningCertFile+".key")

	if _, err = instance.ReadSigningCert(); err == nil {
		log.Error().Msgf("%s already exists", geneos.SigningCertFile)
		return
	}

	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "geneos intermediate CA",
		},
		NotBefore:             time.Now().Add(-60 * time.Second),
		NotAfter:              time.Now().AddDate(10, 0, 0).Truncate(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            1,
	}

	rootCert, err := instance.ReadRootCert()
	if err != nil {
		return
	}
	rootKey, err := geneos.LOCAL.ReadKey(filepath.Join(dir, geneos.RootCAFile+".key"))
	if err != nil {
		return
	}

	cert, key, err := instance.CreateCertKey(&template, rootCert, rootKey, nil)
	if err != nil {
		return
	}

	if err = geneos.LOCAL.WriteCert(intrCertPath, cert); err != nil {
		return
	}
	if err = geneos.LOCAL.WriteKey(intrKeyPath, key); err != nil {
		return
	}

	fmt.Printf("Signing certificate created for %s\n", geneos.SigningCertFile)

	return
}
