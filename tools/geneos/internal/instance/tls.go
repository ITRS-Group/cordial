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

package instance

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// CreateCert creates a new certificate for an instance
//
// this also creates a new private key
//
// skip if certificate exists and is valid
func CreateCert(i geneos.Instance) (resp *Response) {
	resp = NewResponse(i)
	// skip if we can load an existing certificate
	if _, _, err := ReadCert(i); err == nil {
		return
	}

	hostname := i.Host().GetString("hostname")

	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		resp.Err = err
		return
	}
	expires := time.Now().AddDate(1, 0, 0).Truncate(24 * time.Hour)
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("geneos %s %s", i.Type(), i.Name()),
		},
		NotBefore:      time.Now().Add(-60 * time.Second),
		NotAfter:       expires,
		KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		MaxPathLenZero: true,
		DNSNames:       []string{hostname},
		// IPAddresses:    []net.IP{net.ParseIP("127.0.0.1")},
	}

	rootCert, err := ReadRootCert()
	if err != nil {
		resp.Err = err
		return
	}

	signingCert, err := ReadSigningCert()
	if err != nil {
		resp.Err = err
		return
	}
	signingKey, err := config.ReadPrivateKey(geneos.LOCAL, path.Join(config.AppConfigDir(), geneos.SigningCertFile+".key"))
	if err != nil {
		resp.Err = err
		return
	}

	cert, key, err := config.CreateCertificateAndKey(&template, signingCert, signingKey, nil)
	if err != nil {
		resp.Err = err
		return
	}

	if err = WriteCert(i, cert); err != nil {
		resp.Err = err
		return
	}

	if err = WriteKey(i, key); err != nil {
		resp.Err = err
		return
	}

	chainfile := PathOf(i, "certchain")
	if chainfile == "" {
		chainfile = path.Join(i.Home(), "chain.pem")
		i.Config().Set("certchain", chainfile)
	}
	if err = config.WriteCertChain(i.Host(), chainfile, signingCert, rootCert); err != nil {
		resp.Err = err
		return
	}

	if err = SaveConfig(i); err != nil {
		resp.Err = err
		return
	}

	resp.Line = fmt.Sprintf("certificate created for %s (expires %s)", i, expires.UTC())
	return
}

// WriteCert writes the certificate for the instance c and updates the
// "certificate" instance parameter. It does not save the instance
// configuration.
func WriteCert(i geneos.Instance, cert *x509.Certificate) (err error) {
	cf := i.Config()

	if i.Type() == nil {
		return geneos.ErrInvalidArgs
	}
	certfile := ComponentFilepath(i, "pem")
	if err = config.WriteCert(i.Host(), certfile, cert); err != nil {
		return
	}
	if cf.GetString("certificate") == certfile {
		return
	}
	cf.Set("certificate", certfile)
	return
}

// WriteKey writes the key for the instance c and updates the
// "privatekey" instance parameter. It does not save the instance
// configuration.
func WriteKey(i geneos.Instance, key *memguard.Enclave) (err error) {
	cf := i.Config()

	if i.Type() == nil {
		return geneos.ErrInvalidArgs
	}

	keyfile := ComponentFilepath(i, "key")
	if err = config.WritePrivateKey(i.Host(), keyfile, key); err != nil {
		return
	}
	if cf.GetString("privatekey") == keyfile {
		return
	}
	cf.Set("privatekey", keyfile)
	return
}

// ReadRootCert reads the root certificate from the user's app config
// directory. It "promotes" old cert and key files from the previous tls
// directory if files do not already exist in the user app config
// directory. If verify is true then the certificate is verified against
// itself as a root and if it fails an error is returned.
func ReadRootCert(verify ...bool) (cert *x509.Certificate, err error) {
	file := config.PromoteFile(host.Localhost, config.AppConfigDir(), geneos.LOCAL.PathTo("tls"), geneos.RootCAFile+".pem")
	log.Debug().Msgf("reading %s", file)
	if file == "" {
		err = fmt.Errorf("%w: root certificate file %s not found in %s", os.ErrNotExist, geneos.RootCAFile+".pem", config.AppConfigDir())
		return
	}
	config.PromoteFile(host.Localhost, config.AppConfigDir(), geneos.LOCAL.PathTo("tls"), geneos.RootCAFile+".key")
	cert, err = config.ParseCertificate(geneos.LOCAL, file)
	if err != nil {
		return
	}
	if len(verify) > 0 && verify[0] {
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
func ReadSigningCert(verify ...bool) (cert *x509.Certificate, err error) {
	file := config.PromoteFile(host.Localhost, config.AppConfigDir(), geneos.LOCAL.PathTo("tls", geneos.SigningCertFile+".pem"))
	log.Debug().Msgf("reading %s", file)
	if file == "" {
		err = fmt.Errorf("%w: signing certificate file %s not found in %s", os.ErrNotExist, geneos.SigningCertFile+".pem", config.AppConfigDir())
		return
	}
	config.PromoteFile(host.Localhost, config.AppConfigDir(), geneos.LOCAL.PathTo("tls", geneos.SigningCertFile+".key"))
	cert, err = config.ParseCertificate(geneos.LOCAL, file)
	if err != nil {
		return
	}
	if len(verify) > 0 && verify[0] {
		if !cert.BasicConstraintsValid || !cert.IsCA {
			err = errors.New("certificate not valid as a signing certificate")
			return
		}
		var root *x509.Certificate
		root, err = ReadRootCert(verify...)
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

// ReadCert reads the instance certificate for c. It verifies the
// certificate against any chain file and, if that fails, against system
// certificates.
func ReadCert(i geneos.Instance) (cert *x509.Certificate, valid bool, err error) {
	if i.Type() == nil {
		return nil, false, geneos.ErrInvalidArgs
	}

	if FileOf(i, "certificate") == "" {
		return nil, false, os.ErrNotExist
	}
	cert, err = config.ParseCertificate(i.Host(), PathOf(i, "certificate"))
	if err != nil {
		return
	}

	// validate against certificate chain file on the same host, expiry
	// etc.
	chainfile := PathOf(i, "certchain")
	if chainfile == "" {
		chainfile = config.PromoteFile(i.Host(), i.Host().PathTo("tls", geneos.ChainCertFile), i.Host().PathTo("tls", "chain.pem"))
	}

	if cp := config.ReadCertChain(i.Host(), chainfile); cp != nil {
		opts := x509.VerifyOptions{
			Roots:         cp,
			Intermediates: cp,
		}

		if _, err = cert.Verify(opts); err == nil { // return if no error
			log.Debug().Msgf("cert %q verified", cert.Subject.CommonName)
			return cert, true, err
		}
		log.Debug().Err(err).Msg("")
	}

	// if failed against internal certs, try system ones
	if _, err = cert.Verify(x509.VerifyOptions{}); err == nil { // return if no error
		valid = true
		log.Debug().Msgf("cert %q verified", cert.Subject.CommonName)
		return
	}

	log.Debug().Msgf("cert %q NOT verified: %s", cert.Subject.CommonName, err)
	return
}

// ReadKey reads the instance RSA private key
func ReadKey(i geneos.Instance) (key *memguard.Enclave, err error) {
	if i.Type() == nil || i.Config().GetString("privatekey") == "" {
		return nil, geneos.ErrInvalidArgs
	}

	return config.ReadPrivateKey(i.Host(), Abs(i, i.Config().GetString("privatekey")))
}
