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
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/awnumar/memguard"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// create a new certificate for an instance
//
// this also creates a new private key
//
// skip if certificate exists (no expiry check)
func CreateCert(c geneos.Instance) (err error) {
	tlsDir := filepath.Join(geneos.Root(), "tls")

	// skip if we can load an existing certificate
	if _, err = ReadCert(c); err == nil {
		return
	}

	hostname, _ := os.Hostname()
	if c.Host() != geneos.LOCAL {
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
	intrKey, err := geneos.LOCAL.ReadKey(filepath.Join(tlsDir, geneos.SigningCertFile+".key"))
	if err != nil {
		return
	}

	cert, key, err := CreateCertKey(&template, intrCert, intrKey, nil)
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

func WriteCert(c geneos.Instance, cert *x509.Certificate) (err error) {
	cf := c.Config()

	if c.Type() == nil {
		return geneos.ErrInvalidArgs
	}
	certfile := c.Type().String() + ".pem"
	if err = c.Host().WriteCert(filepath.Join(c.Home(), certfile), cert); err != nil {
		return
	}
	if cf.GetString("certificate") == certfile {
		return
	}
	cf.Set("certificate", certfile)

	return cf.Save(c.Type().String(),
		config.Host(c.Host()),
		config.SaveDir(c.Type().InstancesDir(c.Host())),
		config.SetAppName(c.Name()),
	)
}

func WriteKey(c geneos.Instance, key *memguard.Enclave) (err error) {
	cf := c.Config()

	if c.Type() == nil {
		return geneos.ErrInvalidArgs
	}

	keyfile := c.Type().String() + ".key"
	if err = c.Host().WriteKey(filepath.Join(c.Home(), keyfile), key); err != nil {
		return
	}
	if cf.GetString("privatekey") == keyfile {
		return
	}
	cf.Set("privatekey", keyfile)
	return cf.Save(c.Type().String(),
		config.Host(c.Host()),
		config.SaveDir(c.Type().InstancesDir(c.Host())),
		config.SetAppName(c.Name()),
	)
}

// read the rootCA certificate from the installation directory
func ReadRootCert() (cert *x509.Certificate, err error) {
	tlsDir := filepath.Join(geneos.Root(), "tls")
	return geneos.LOCAL.ReadCert(filepath.Join(tlsDir, geneos.RootCAFile+".pem"))
}

// read the signing certificate from the installation directory
func ReadSigningCert() (cert *x509.Certificate, err error) {
	tlsDir := filepath.Join(geneos.Root(), "tls")
	return geneos.LOCAL.ReadCert(filepath.Join(tlsDir, geneos.SigningCertFile+".pem"))
}

// read the instance certificate
func ReadCert(c geneos.Instance) (cert *x509.Certificate, err error) {
	if c.Type() == nil {
		return nil, geneos.ErrInvalidArgs
	}

	if Filename(c, "certificate") == "" {
		return nil, os.ErrNotExist
	}
	return c.Host().ReadCert(Filepath(c, "certificate"))
}

// read the instance RSA private key
func ReadKey(c geneos.Instance) (key *memguard.Enclave, err error) {
	if c.Type() == nil || c.Config().GetString("privatekey") == "" {
		return nil, geneos.ErrInvalidArgs
	}

	return c.Host().ReadKey(Abs(c, c.Config().GetString("privatekey")))
}

// NewPrivateKey returns a PKCS1 encoded RSA Private Key as an enclave. It is
// not PEM encoded.
func NewPrivateKey() *memguard.Enclave {
	certKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return memguard.NewEnclave(x509.MarshalPKCS1PrivateKey(certKey))
}

// wrapper to create a new certificate given the sign cert and private
// key and an optional private key to (re)use for the created
// certificate itself. returns a certificate and private key. Keys are
// in PEM format so need parsing after unsealing.
func CreateCertKey(template, parent *x509.Certificate, parentKeyPEM, existingKeyPEM *memguard.Enclave) (cert *x509.Certificate, keyPEM *memguard.Enclave, err error) {
	var certBytes []byte
	var certKey *rsa.PrivateKey

	if template != parent && parentKeyPEM == nil {
		err = errors.New("parent key empty but not self-signing")
		return
	}

	keyPEM = existingKeyPEM
	if keyPEM == nil {
		keyPEM = NewPrivateKey()
	}

	l, _ := keyPEM.Open()
	if certKey, err = x509.ParsePKCS1PrivateKey(l.Bytes()); err != nil {
		keyPEM = nil
		return
	}

	signingKey := certKey
	certPubKey := &certKey.PublicKey

	if parentKeyPEM != nil {
		pk, _ := parentKeyPEM.Open()
		if signingKey, err = x509.ParsePKCS1PrivateKey(pk.Bytes()); err != nil {
			keyPEM = nil
			return
		}
		pk.Destroy()
	}

	if certBytes, err = x509.CreateCertificate(rand.Reader, template, parent, certPubKey, signingKey); err != nil {
		keyPEM = nil
		l.Destroy()
		return
	}

	if cert, err = x509.ParseCertificate(certBytes); err != nil {
		keyPEM = nil
		l.Destroy()
		return
	}

	keyPEM = l.Seal()
	return
}
