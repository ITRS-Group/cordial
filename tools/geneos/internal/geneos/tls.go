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

package geneos

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// ReadRootCert reads the root certificate from the user's app config
// directory. It "promotes" old cert and key files from the previous tls
// directory if files do not already exist in the user app config
// directory. If verify is true then the certificate is verified against
// itself as a root and if it fails an error is returned.
func ReadRootCert(verify ...bool) (cert *x509.Certificate, file string, err error) {
	file = config.PromoteFile(host.Localhost, config.AppConfigDir(), LOCAL.PathTo("tls"), RootCAFile+".pem")
	if file == "" {
		return
	}
	log.Debug().Msgf("reading %s", file)
	if file == "" {
		err = fmt.Errorf("%w: root certificate file %s not found in %s", os.ErrNotExist, RootCAFile+".pem", config.AppConfigDir())
		return
	}
	config.PromoteFile(host.Localhost, config.AppConfigDir(), LOCAL.PathTo("tls"), RootCAFile+".key")
	cert, err = config.ParseCertificate(LOCAL, file)
	if err != nil {
		return
	}
	if len(verify) > 0 && verify[0] {
		if !cert.BasicConstraintsValid || !cert.IsCA {
			err = errors.New("root certificate not valid as a signing certificate")
			return
		}
		roots := x509.NewCertPool()
		roots.AddCert(cert)
		_, err = cert.Verify(x509.VerifyOptions{
			Roots: roots,
		})
	}
	return
}

// ReadSigningCert reads the signing certificate from the user's app
// config directory. It "promotes" old cert and key files from the
// previous tls directory if files do not already exist in the user app
// config directory. If verify is true then the signing certificate is
// checked and verified against the default root certificate.
func ReadSigningCert(verify ...bool) (cert *x509.Certificate, file string, err error) {
	file = config.PromoteFile(host.Localhost, config.AppConfigDir(), LOCAL.PathTo("tls", SigningCertFile+".pem"))
	log.Debug().Msgf("reading %s", file)
	if file == "" {
		err = fmt.Errorf("%w: signing certificate file %s not found in %s", os.ErrNotExist, SigningCertFile+".pem", config.AppConfigDir())
		return
	}
	config.PromoteFile(host.Localhost, config.AppConfigDir(), LOCAL.PathTo("tls", SigningCertFile+".key"))
	cert, err = config.ParseCertificate(LOCAL, file)
	if err != nil {
		return
	}
	if len(verify) > 0 && verify[0] {
		if !cert.BasicConstraintsValid || !cert.IsCA {
			err = errors.New("certificate not valid as a signing certificate")
			return
		}
		var root *x509.Certificate
		root, _, err = ReadRootCert(verify...)
		if err != nil {
			return
		}
		roots := x509.NewCertPool()
		roots.AddCert(root)
		_, err = cert.Verify(x509.VerifyOptions{
			Roots: roots,
		})
	}
	return
}

// DecomposePEM parses PEM formatted data and extracts the leaf
// certificate, any CA certs as a chain and a private key as a DER
// encoded *memguard.Enclave. The key is matched to the leaf
// certificate.
func DecomposePEM(data ...string) (cert *x509.Certificate, der *memguard.Enclave, chain []*x509.Certificate, err error) {
	var certs []*x509.Certificate
	var leaf *x509.Certificate
	var derkeys []*memguard.Enclave

	if len(data) == 0 {
		err = fmt.Errorf("no PEM data process")
		return
	}

	for _, pemstring := range data {
		pembytes := []byte(pemstring)
		for {
			block, rest := pem.Decode(pembytes)
			if block == nil {
				break
			}
			switch block.Type {
			case "CERTIFICATE":
				var c *x509.Certificate
				c, err = x509.ParseCertificate(block.Bytes)
				if err != nil {
					return
				}
				if !c.BasicConstraintsValid {
					err = ErrInvalidArgs
					return
				}
				if c.IsCA {
					certs = append(certs, c)
				} else if leaf == nil {
					// save first leaf
					leaf = c
				}
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				// save all private keys for later matching
				derkeys = append(derkeys, memguard.NewEnclave(block.Bytes))
			default:
				err = fmt.Errorf("unsupported PEM type found: %s", block.Type)
				return
			}
			pembytes = rest
		}
	}

	if leaf == nil && len(certs) == 0 {
		err = fmt.Errorf("no certificates found")
		return
	}

	// if we got this far then we can start setting returns
	cert = leaf
	chain = certs

	// if we have no leaf certificate then user the first cert from the
	// chain BUT leave do not remove from the chain. order is not checked
	if cert == nil {
		cert = chain[0]
	}

	// are we good? check key and return a chain of valid CA certs
	if i := config.MatchKey(cert, derkeys); i != -1 {
		der = derkeys[i]
	}

	err = nil
	return
}
