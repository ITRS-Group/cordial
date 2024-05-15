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
	"errors"
	"fmt"
	"os"

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
