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
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/host"
)

// ParseCertificate reads a PEM encoded cert from path on host h, return the
// first found as a parsed certificate
func ParseCertificate(h host.Host, pt string) (cert *x509.Certificate, err error) {
	certPEM, err := h.ReadFile(pt)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(certPEM)
		if p == nil {
			return nil, fmt.Errorf("cannot locate certificate in %s", pt)
		}
		if p.Type == "CERTIFICATE" {
			return x509.ParseCertificate(p.Bytes)
		}
		certPEM = rest
	}
}

// ParseCertificates reads a PEM encoded file from host h and returns
// all the certificates found (using the same rules as
// x509.ParseCertificates).
func ParseCertificates(h host.Host, p string) (certs []*x509.Certificate, err error) {
	certPEM, err := h.ReadFile(p)
	if err != nil {
		return
	}

	return x509.ParseCertificates(certPEM)
}

// ReadPrivateKey reads a unencrypted, PEM-encoded private key and saves
// the decoded, but unparsed, key in a memguard.Enclave
func ReadPrivateKey(h host.Host, pt string) (key *memguard.Enclave, err error) {
	keyPEM, err := h.ReadFile(pt)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(keyPEM)
		if p == nil {
			return nil, fmt.Errorf("cannot locate private key in %s", pt)
		}
		if strings.HasSuffix(p.Type, "PRIVATE KEY") {
			key = memguard.NewEnclave(p.Bytes)
			return
		}
		keyPEM = rest
	}
}

// ParseKey tries to parse the PEM encoded private key first as PKCS#8
// and then PKCS#1 if that fails. It returns the private and public keys
// or an error
func ParseKey(keyPEM *memguard.Enclave) (privateKey any, publickey crypto.PublicKey, err error) {
	k, err := keyPEM.Open()
	if err != nil {
		return
	}
	defer k.Destroy()
	if privateKey, err = x509.ParsePKCS8PrivateKey(k.Bytes()); err != nil {
		if privateKey, err = x509.ParsePKCS1PrivateKey(k.Bytes()); err != nil {
			return
		}
	}
	if k, ok := privateKey.(crypto.Signer); ok {
		publickey = k.Public()
	}
	return
}

// WriteCert writes cert as PEM to path on host h
func WriteCert(h host.Host, p string, cert *x509.Certificate) (err error) {
	log.Debug().Msgf("write cert to %s", p)
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	return h.WriteFile(p, certPEM, 0644)
}

// WriteCerts concatenate certs and writes to path on host h
func WriteCerts(h host.Host, p string, certs ...*x509.Certificate) (err error) {
	log.Debug().Msgf("write certs to %s", p)
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
	return h.WriteFile(p, certsPEM, 0644)
}

// WritePrivateKey writes a private key as PEM to path on host h. sets file
// permissions to 0600 (before umask)
func WritePrivateKey(h host.Host, pt string, key *memguard.Enclave) (err error) {
	l, _ := key.Open()
	defer l.Destroy()
	p := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: l.Bytes(),
	})
	return h.WriteFile(pt, p, 0600)
}

const DefaultKeyType = "ecdh"

// CreateCertificateAndKey is a wrapper to create a new certificate
// given the signing cert and key and an optional private key to (re)use
// for the certificate creation. Returns a certificate and private key.
// Keys are usually PKCS#8 encoded and so need parsing after unsealing.
func CreateCertificateAndKey(template, parent *x509.Certificate, signingKeyPEM, existingKeyPEM *memguard.Enclave) (cert *x509.Certificate, certKeyPEM *memguard.Enclave, err error) {
	var certBytes []byte
	// var certKey *rsa.PrivateKey

	if template != parent && signingKeyPEM == nil {
		err = errors.New("parent key empty but not self-signing")
		return
	}

	certKeyPEM = existingKeyPEM
	if certKeyPEM == nil {
		keytype := KeyType(signingKeyPEM)
		if keytype == "" {
			keytype = DefaultKeyType
		}
		certKeyPEM, err = NewPrivateKey(keytype)
		if err != nil {
			panic(err)
		}
	}

	// default the signingKey to the certKey (for self-signed root)
	signingKey, certPubKey, err := ParseKey(certKeyPEM)

	if signingKeyPEM != nil {
		signingKey, _, err = ParseKey(signingKeyPEM)
		if err != nil {
			certKeyPEM = nil
			return
		}
	}

	if certBytes, err = x509.CreateCertificate(rand.Reader, template, parent, certPubKey, signingKey); err != nil {
		certKeyPEM = nil
		return
	}

	if cert, err = x509.ParseCertificate(certBytes); err != nil {
		certKeyPEM = nil
		return
	}

	return
}

// KeyType returns the type of key, suitable for use to NewPrivateKey
func KeyType(key *memguard.Enclave) (keytype string) {
	if key == nil {
		return
	}
	privateKey, _, err := ParseKey(key)
	if err != nil {
		return
	}

	switch privateKey.(type) {
	case *rsa.PrivateKey:
		return "rsa"
	case *ecdsa.PrivateKey:
		return "ecdsa"
	case *ecdh.PrivateKey:
		return "ecdh"
	case ed25519.PrivateKey:
		return "ed2559"
	default:
		return ""
	}
}

// NewPrivateKey returns a PKCS8 encoded private key as an enclave.
func NewPrivateKey(keytype string) (k *memguard.Enclave, err error) {
	var privateKey any
	switch keytype {
	case "rsa":
		privateKey, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	case "ecdsa":
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	case "ed25519":
		_, privateKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	case "ecdh":
		ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		privateKey, err = ecdsaKey.ECDH()
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	default:
		log.Fatal().Msgf("unsupported key type %s", keytype)
	}

	key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	k = memguard.NewEnclave(key)
	return
}

// CreateRootCert creates a new root certificate and private key and
// saves it with dir and file basefilepath with .pem and .key extensions. If
// overwrite is true then any existing certificate and key is
// overwritten.
func CreateRootCert(h host.Host, basefilepath string, cn string, overwrite bool, keytype string) (err error) {
	// create rootCA.pem / rootCA.key
	var cert *x509.Certificate

	if !overwrite {
		if _, err = ParseCertificate(h, basefilepath+".pem"); err == nil {
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
			CommonName: cn,
		},
		NotBefore:             time.Now().Add(-60 * time.Second),
		NotAfter:              time.Now().AddDate(10, 0, 0).Truncate(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            2,
	}

	privateKeyPEM, err := NewPrivateKey(keytype)

	cert, key, err := CreateCertificateAndKey(template, template, privateKeyPEM, nil)
	if err != nil {
		return
	}

	if err = WriteCert(h, basefilepath+".pem", cert); err != nil {
		return
	}
	if err = WritePrivateKey(h, basefilepath+".key", key); err != nil {
		return
	}

	return
}

// CreateSigningCert creates a new signing certificate and private key
// with the path and file bane name basefilepath. You must provide a
// valid root certificate and key in rootbasefilepath. If overwrite is
// true than any existing cert and key are overwritten.
func CreateSigningCert(h host.Host, basefilepath string, rootbasefilepath string, cn string, overwrite bool) (err error) {
	var cert *x509.Certificate

	if !overwrite {
		if _, err = ParseCertificate(h, basefilepath+".pem"); err == nil {
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
			CommonName: cn,
		},
		NotBefore:             time.Now().Add(-60 * time.Second),
		NotAfter:              time.Now().AddDate(10, 0, 0).Truncate(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            1,
	}

	rootCert, err := ParseCertificate(h, rootbasefilepath+".pem")
	if err != nil {
		return
	}
	rootKey, err := ReadPrivateKey(h, rootbasefilepath+".key")
	if err != nil {
		return
	}

	cert, key, err := CreateCertificateAndKey(&template, rootCert, rootKey, nil)
	if err != nil {
		return
	}

	if err = WriteCert(h, basefilepath+".pem", cert); err != nil {
		return
	}
	if err = WritePrivateKey(h, basefilepath+".key", key); err != nil {
		return
	}

	return
}
