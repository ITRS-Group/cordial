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
	"fmt"
	"os"
	"path/filepath"
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
func CreateCert(c geneos.Instance) (err error) {
	// skip if we can load an existing certificate
	if _, _, err = ReadCert(c); err == nil {
		return
	}

	hostname, _ := os.Hostname()
	if !c.Host().IsLocal() {
		hostname = c.Host().GetString("hostname")
	}

	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	expires := time.Now().AddDate(1, 0, 0).Truncate(24 * time.Hour)
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("geneos %s %s", c.Type(), c.Name()),
		},
		NotBefore:      time.Now().Add(-60 * time.Second),
		NotAfter:       expires,
		KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		MaxPathLenZero: true,
		DNSNames:       []string{hostname},
		// IPAddresses:    []net.IP{net.ParseIP("127.0.0.1")},
	}

	intrCert, err := ReadSigningCert()
	if err != nil {
		return
	}
	intrKey, err := config.ReadPrivateKey(geneos.LOCAL, filepath.Join(config.AppConfigDir(), geneos.SigningCertFile+".key"))
	if err != nil {
		return
	}

	cert, key, err := config.CreateCertKey(&template, intrCert, intrKey, nil)
	if err != nil {
		return
	}

	if err = WriteCert(c, cert); err != nil {
		return
	}

	if err = WriteKey(c, key); err != nil {
		return
	}

	fmt.Printf("certificate created for %s (expires %s)\n", c, expires.UTC())

	return
}

// WriteCert writes the certificate for the instance c
func WriteCert(c geneos.Instance, cert *x509.Certificate) (err error) {
	cf := c.Config()

	if c.Type() == nil {
		return geneos.ErrInvalidArgs
	}
	certfile := c.Type().String() + ".pem"
	if err = config.WriteCert(c.Host(), filepath.Join(c.Home(), certfile), cert); err != nil {
		return
	}
	if cf.GetString("certificate") == certfile {
		return
	}
	cf.Set("certificate", certfile)

	return cf.Save(c.Type().String(),
		config.Host(c.Host()),
		config.SaveDir(ParentDirectory(c)),
		config.SetAppName(c.Name()),
	)
}

// WriteKey writes the key for the instance c
func WriteKey(c geneos.Instance, key *memguard.Enclave) (err error) {
	cf := c.Config()

	if c.Type() == nil {
		return geneos.ErrInvalidArgs
	}

	keyfile := c.Type().String() + ".key"
	if err = config.WritePrivateKey(c.Host(), filepath.Join(c.Home(), keyfile), key); err != nil {
		return
	}
	if cf.GetString("privatekey") == keyfile {
		return
	}
	cf.Set("privatekey", keyfile)
	return cf.Save(c.Type().String(),
		config.Host(c.Host()),
		config.SaveDir(ParentDirectory(c)),
		config.SetAppName(c.Name()),
	)
}

// ReadRootCert reads the root certificate from the user's app config
// directory. It "promotes" old cert and key files from the previous tls
// directory if files do not already exist in the user app config
// directory.
func ReadRootCert() (cert *x509.Certificate, err error) {
	file := config.PromoteFile(host.Localhost, config.AppConfigDir(), geneos.LOCAL.Filepath("tls"), geneos.RootCAFile+".pem")
	log.Debug().Msgf("reading %s", file)
	config.PromoteFile(host.Localhost, config.AppConfigDir(), geneos.LOCAL.Filepath("tls"), geneos.RootCAFile+".key")
	return config.ParseCertificate(geneos.LOCAL, file)
}

// ReadSigningCert reads the signing certificate from the user's app
// config directory. It "promotes" old cert and key files from the
// previous tls directory if files do not already exist in the user app
// config directory.
func ReadSigningCert() (cert *x509.Certificate, err error) {
	file := config.PromoteFile(host.Localhost, config.AppConfigDir(), geneos.LOCAL.Filepath("tls", geneos.SigningCertFile+".pem"))
	log.Debug().Msgf("reading %s", file)
	config.PromoteFile(host.Localhost, config.AppConfigDir(), geneos.LOCAL.Filepath("tls", geneos.SigningCertFile+".key"))
	return config.ParseCertificate(geneos.LOCAL, file)
}

// ReadCert reads the instance certificate
func ReadCert(c geneos.Instance) (cert *x509.Certificate, valid bool, err error) {
	if c.Type() == nil {
		return nil, false, geneos.ErrInvalidArgs
	}

	if Filename(c, "certificate") == "" {
		return nil, false, os.ErrNotExist
	}
	cert, err = config.ParseCertificate(c.Host(), Filepath(c, "certificate"))

	// validate against certificate chain file on the same host, expiry
	// etc.
	chainfile := config.PromoteFile(c.Host(), c.Host().Filepath("tls", geneos.ChainCertFile), c.Host().Filepath("tls", "chain.pem"))
	if chain, err := c.Host().ReadFile(chainfile); err == nil {
		cp := x509.NewCertPool()
		cp.AppendCertsFromPEM(chain)

		opts := x509.VerifyOptions{
			Roots: cp,
		}

		if _, err = cert.Verify(opts); err == nil { // return if no error
			log.Debug().Msgf("cert %q verified", cert.Subject.CommonName)
			return cert, true, err
		}
	}

	// if failed against internal certs, try system ones
	if _, err = cert.Verify(x509.VerifyOptions{}); err == nil { // return if no error
		valid = true
		log.Debug().Msgf("cert %q verified", cert.Subject.CommonName)
		return
	}

	log.Debug().Msgf("cert %q NOT verified", cert.Subject.CommonName)
	return
}

// ReadKey reads the instance RSA private key
func ReadKey(c geneos.Instance) (key *memguard.Enclave, err error) {
	if c.Type() == nil || c.Config().GetString("privatekey") == "" {
		return nil, geneos.ErrInvalidArgs
	}

	return config.ReadPrivateKey(c.Host(), Abs(c, c.Config().GetString("privatekey")))
}
