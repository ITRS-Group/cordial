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
package cmd

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"path/filepath"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// tlsInitCmd represents the tlsInit command
var tlsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise the TLS environment",
	Long: `Initialise the TLS environment by creating a self-signed
root certificate to act as a CA and a signing certificate signed
by the root. Any instances will have certificates created for
them but configurations will not be rebuilt.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		// _, _, params := processArgsParams(cmd)
		return TLSInit()
	},
}

func init() {
	tlsCmd.AddCommand(tlsInitCmd)
	tlsInitCmd.Flags().SortFlags = false
}

// create the tls/ directory in Geneos and a CA / DCA as required
//
// later options to allow import of a DCA
func TLSInit() (err error) {
	tlsPath := filepath.Join(host.Geneos(), "tls")
	// directory permissions do not need to be restrictive
	err = host.LOCAL.MkdirAll(tlsPath, 0775)
	if err != nil {
		logError.Fatalln(err)
	}

	if err := newRootCA(tlsPath); err != nil {
		logError.Fatalln(err)
	}

	if err := newIntrCA(tlsPath); err != nil {
		logError.Fatalln(err)
	}

	return TLSSync()
}

func newRootCA(dir string) (err error) {
	// create rootCA.pem / rootCA.key
	var cert *x509.Certificate
	rootCertPath := filepath.Join(dir, geneos.RootCAFile+".pem")
	rootKeyPath := filepath.Join(dir, geneos.RootCAFile+".key")

	if _, err = instance.ReadRootCert(); err == nil {
		log.Println(geneos.RootCAFile, "already exists")
		return
	}
	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "geneos root CA",
		},
		NotBefore:             time.Now().Add(-60 * time.Second),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            2,
	}

	cert, key, err := instance.CreateCertKey(&template, &template, nil, nil)
	if err != nil {
		return
	}

	if err = host.LOCAL.WriteCert(rootCertPath, cert); err != nil {
		return
	}
	if err = host.LOCAL.WriteKey(rootKeyPath, key); err != nil {
		return
	}
	log.Println("CA certificate created for", geneos.RootCAFile)

	return
}

func newIntrCA(dir string) (err error) {
	var cert *x509.Certificate
	intrCertPath := filepath.Join(dir, geneos.SigningCertFile+".pem")
	intrKeyPath := filepath.Join(dir, geneos.SigningCertFile+".key")

	if _, err = instance.ReadSigningCert(); err == nil {
		log.Println(geneos.SigningCertFile, "already exists")
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
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            1,
	}

	rootCert, err := instance.ReadRootCert()
	if err != nil {
		return
	}
	rootKey, err := host.LOCAL.ReadKey(filepath.Join(dir, geneos.RootCAFile+".key"))
	if err != nil {
		return
	}

	cert, key, err := instance.CreateCertKey(&template, rootCert, rootKey, nil)
	if err != nil {
		return
	}

	if err = host.LOCAL.WriteCert(intrCertPath, cert); err != nil {
		return
	}
	if err = host.LOCAL.WriteKey(intrKeyPath, key); err != nil {
		return
	}

	log.Println("Signing certificate created for", geneos.SigningCertFile)

	return
}
