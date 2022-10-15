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
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// tlsImportCmd represents the tlsImport command
var tlsImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import root and signing certificates",
	Long: strings.ReplaceAll(`
Import non-instance certificates. A root certificate is one where the
subject is the same as the issuer. All other certificates are
imported as signing certs. Only the last one, if multiple are given,
is used. Private keys must be supplied, either as individual files on
in the certificate files and cannot be password protected. Only
certificates with matching private keys are imported.
`, "|", "`"),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		_, args, _ := cmdArgsParams(cmd)
		return tlsImport(args...)
	},
}

func init() {
	tlsCmd.AddCommand(tlsImportCmd)
	tlsImportCmd.Flags().SortFlags = false
}

// import root and signing certs
//
// a root cert is one where subject == issuer
//
// no support for instance certs (yet)
func tlsImport(sources ...string) (err error) {
	log.Debug().Msgf("%v", sources)
	tlsPath := filepath.Join(host.Geneos(), "tls")
	err = host.LOCAL.MkdirAll(tlsPath, 0755)
	if err != nil {
		return
	}

	// save certs and keys into memory, then check certs for root / etc.
	// and then validate private keys against certs before saving
	// anything to disk
	var certs []*x509.Certificate
	var keys []*rsa.PrivateKey
	var f []byte

	for _, source := range sources {
		log.Debug().Msgf("importing %s", source)
		if f, err = geneos.ReadSource(source); err != nil {
			log.Error().Err(err).Msg("")
			err = nil
			continue
		}

		for {
			block, rest := pem.Decode(f)
			if block == nil {
				break
			}
			switch block.Type {
			case "CERTIFICATE":
				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return err
				}
				certs = append(certs, cert)
			case "RSA PRIVATE KEY":
				key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
				if err != nil {
					return err
				}
				if err = key.Validate(); err != nil {
					return err
				}
				keys = append(keys, key)
			default:
				return fmt.Errorf("unknown PEM type found: %s", block.Type)
			}
			f = rest
		}
	}

	var title, prefix string
	for _, cert := range certs {
		if bytes.Equal(cert.RawSubject, cert.RawIssuer) {
			// root cert
			title = "root"
			prefix = geneos.RootCAFile
		} else {
			// signing cert
			title = "signing"
			prefix = geneos.SigningCertFile
		}
		i, err := matchKey(cert, keys)
		if err != nil {
			log.Debug().Msgf("cert: no matching key found, ignoring %s", cert.Subject.String())
			continue
		}

		// pull out the matching key, write files
		key := keys[i]
		if len(keys) > i {
			keys = append(keys[:i], keys[i+1:]...)
		} else {
			keys = keys[:i]
		}

		if err = host.LOCAL.WriteCert(filepath.Join(tlsPath, prefix+".pem"), cert); err != nil {
			return err
		}
		fmt.Printf("imported %s certificate to %q\n", title, filepath.Join(tlsPath, prefix+".pem"))
		if err = host.LOCAL.WriteKey(filepath.Join(tlsPath, prefix+".key"), key); err != nil {
			return err
		}
		fmt.Printf("imported %s RSA private key to %q\n", title, filepath.Join(tlsPath, prefix+".pem"))
	}

	return
}

func matchKey(cert *x509.Certificate, keys []*rsa.PrivateKey) (index int, err error) {
	for i, key := range keys {
		if key.PublicKey.Equal(cert.PublicKey) {
			return i, nil
		}
	}
	return -1, os.ErrNotExist
}
