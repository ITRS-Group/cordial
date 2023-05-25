/*
Copyright © 2023 ITRS Group

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

package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/awnumar/memguard"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
)

// ReadKey reads a keyfile and saves the PEM encoded key in an
// enclave (without type or headers)
func ReadKey(h host.Host, path string) (key *memguard.Enclave, err error) {
	keyPEM, err := h.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(keyPEM)
		if p == nil {
			return nil, fmt.Errorf("cannot locate RSA private key in %s", path)
		}
		if p.Type == "RSA PRIVATE KEY" {
			key = memguard.NewEnclave(p.Bytes)
			return
		}
		keyPEM = rest
	}
}

// WriteKey writes a private key as PEM to path on host h. sets file
// permissions to 0600 (before umask)
func WriteKey(h host.Host, path string, key *memguard.Enclave) (err error) {
	l, _ := key.Open()
	defer l.Destroy()
	p := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: l.Bytes(),
	})
	return h.WriteFile(path, p, 0600)
}

// ReadCert reads a PEM encoded cert from path on host h, return the
// first found as a parsed certificate
func ReadCert(h host.Host, path string) (cert *x509.Certificate, err error) {
	certPEM, err := h.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(certPEM)
		if p == nil {
			return nil, fmt.Errorf("cannot locate certificate in %s", path)
		}
		if p.Type == "CERTIFICATE" {
			return x509.ParseCertificate(p.Bytes)
		}
		certPEM = rest
	}
}

// WriteCert writes cert as PEM to path on host h
func WriteCert(h host.Host, path string, cert *x509.Certificate) (err error) {
	log.Debug().Msgf("write cert to %s", path)
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	return h.WriteFile(path, certPEM, 0644)
}

// WriteCerts concatenate certs and writes to path on host h
func WriteCerts(h host.Host, path string, certs ...*x509.Certificate) (err error) {
	log.Debug().Msgf("write certs to %s", path)
	var certsPEM []byte
	for _, cert := range certs {
		if cert == nil {
			continue
		}
		p := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		certsPEM = append(certsPEM, p...)
	}
	return h.WriteFile(path, certsPEM, 0644)
}

// CreateCertKey is a wrapper to create a new certificate given the
// signing cert and private key and an optional private key to (re)use
// for the created certificate itself. returns a certificate and private
// key. Keys are in PEM format so need parsing after unsealing.
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

// NewPrivateKey returns a PKCS1 encoded RSA Private Key as an enclave.
// It is not PEM encoded.
func NewPrivateKey() *memguard.Enclave {
	certKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return memguard.NewEnclave(x509.MarshalPKCS1PrivateKey(certKey))
}

// CreateRootCert creates a new root certificate and private key and
// saves it with dir and file basefilepath with .pem and .key extensions. If
// overwrite is true then any existing certificate and key is
// overwritten.
func CreateRootCert(h host.Host, basefilepath string, overwrite bool) (err error) {
	// create rootCA.pem / rootCA.key
	var cert *x509.Certificate

	if !overwrite {
		if _, err = ReadCert(h, basefilepath+".pem"); err == nil {
			return host.ErrExists
		}
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

	cert, key, err := CreateCertKey(template, template, nil, nil)
	if err != nil {
		return
	}

	if err = WriteCert(h, basefilepath+".pem", cert); err != nil {
		return
	}
	if err = WriteKey(h, basefilepath+".key", key); err != nil {
		return
	}

	return
}

// CreateSigningCert creates a new signing certificate and private key
// with the path and file bane name basefilepath. You must provide a
// valid root certificate and key in rootbasefilepath. If overwrite is
// true than any existing cert and key are overwritten.
func CreateSigningCert(h host.Host, basefilepath string, rootbasefilepath string, overwrite bool) (err error) {
	var cert *x509.Certificate

	if !overwrite {
		if _, err = ReadCert(h, basefilepath+".pem"); err == nil {
			return host.ErrExists
		}
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

	rootCert, err := ReadCert(h, rootbasefilepath+".pem")
	if err != nil {
		return
	}
	rootKey, err := ReadKey(h, rootbasefilepath+".key")
	if err != nil {
		return
	}

	cert, key, err := CreateCertKey(&template, rootCert, rootKey, nil)
	if err != nil {
		return
	}

	if err = WriteCert(h, basefilepath+".pem", cert); err != nil {
		return
	}
	if err = WriteKey(h, basefilepath+".key", key); err != nil {
		return
	}

	return
}
