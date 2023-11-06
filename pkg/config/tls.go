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
	"os"
	"strings"
	"time"

	"github.com/awnumar/memguard"

	"github.com/itrs-group/cordial/pkg/host"
)

// DefaultKeyType is the default key type
const DefaultKeyType = "ecdh"

// ParseCertificate reads a PEM encoded cert from path on host h, return
// the first found as a parsed certificate. The returned certificate is
// not verified or validated beyond that of the underlying Go x509
// package parsing functions.
func ParseCertificate(h host.Host, certfile string) (cert *x509.Certificate, err error) {
	pembytes, err := h.ReadFile(certfile)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(pembytes)
		if p == nil {
			return nil, fmt.Errorf("cannot locate certificate in %s", certfile)
		}
		if p.Type == "CERTIFICATE" {
			return x509.ParseCertificate(p.Bytes)
		}
		pembytes = rest
	}
}

// ParseCertificates reads a PEM encoded file from host h and returns
// all the certificates found (using the same rules as
// x509.ParseCertificates). The returned certificates are not verified
// or validated beyond that of the underlying Go x509 package parsing
// functions.
func ParseCertificates(h host.Host, p string) (certs []*x509.Certificate, err error) {
	pembytes, err := h.ReadFile(p)
	if err != nil {
		return
	}

	return x509.ParseCertificates(pembytes)
}

// ParseKey tries to parse the DER encoded private key enclave, first as
// PKCS#8 and then as a PKCS#1 and finally as SEC1 (EC) if that fails.
// It returns the private and public keys or an error
func ParseKey(der *memguard.Enclave) (privateKey any, publickey crypto.PublicKey, err error) {
	k, err := der.Open()
	if err != nil {
		return
	}
	defer k.Destroy()
	if privateKey, err = x509.ParsePKCS8PrivateKey(k.Bytes()); err != nil {
		if privateKey, err = x509.ParsePKCS1PrivateKey(k.Bytes()); err != nil {
			if privateKey, err = x509.ParseECPrivateKey(k.Bytes()); err != nil {
				return
			}
		}
	}
	if k, ok := privateKey.(crypto.Signer); ok {
		publickey = k.Public()
	}
	return
}

// PublicKey parses the DER encoded private key enclave and returns the
// public key if successful. It will first try as PKCS#8 and then PKCS#1
// if that fails. Using this over the more general ParseKey() ensures
// the decoded private key is not returned to the caller when not
// required.
func PublicKey(der *memguard.Enclave) (publickey crypto.PublicKey, err error) {
	var pkey any

	k, err := der.Open()
	if err != nil {
		return
	}
	defer k.Destroy()
	if pkey, err = x509.ParsePKCS8PrivateKey(k.Bytes()); err != nil {
		if pkey, err = x509.ParsePKCS1PrivateKey(k.Bytes()); err != nil {
			if pkey, err = x509.ParseECPrivateKey(k.Bytes()); err != nil {
				return
			}
		}
	}
	if k, ok := pkey.(crypto.Signer); ok {
		publickey = k.Public()
	}

	return
}

// WriteCert writes cert as PEM to file p on host h
func WriteCert(h host.Host, p string, cert *x509.Certificate) (err error) {
	pembytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	return h.WriteFile(p, pembytes, 0644)
}

// WriteCertChain concatenate certs and writes to path on host h
func WriteCertChain(h host.Host, p string, certs ...*x509.Certificate) (err error) {
	var pembytes []byte
	for _, cert := range certs {
		if cert == nil {
			continue
		}
		p := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		pembytes = append(pembytes, p...)
	}
	return h.WriteFile(p, pembytes, 0644)
}

// ReadCertificatePEM reads a PEM encoded certificate from file at path p
// on host h and returns the block
func ReadCertificatePEM(h host.Host, pt string) (data []byte, err error) {
	pembytes, err := h.ReadFile(pt)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(pembytes)
		if p == nil {
			return nil, fmt.Errorf("cannot locate certificate in %s", p)
		}
		if p.Type == "CERTIFICATE" {
			return pem.EncodeToMemory(p), nil
		}
		pembytes = rest
	}
}

// ReadCertChain returns a certificate pool loaded from the file on host
// h at path p. If there is any error a nil pointer is returned.
func ReadCertChain(h host.Host, p string) (pool *x509.CertPool) {
	pool = x509.NewCertPool()
	if chain, err := h.ReadFile(p); err == nil {
		if ok := pool.AppendCertsFromPEM(chain); !ok {
			return nil
		}
	}
	return
}

// ReadPrivateKey reads a unencrypted, PEM-encoded private key and saves
// the der format key in a memguard.Enclave
func ReadPrivateKey(h host.Host, pt string) (key *memguard.Enclave, err error) {
	pembytes, err := h.ReadFile(pt)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(pembytes)
		if p == nil {
			return nil, fmt.Errorf("cannot locate private key in %s", pt)
		}
		if strings.HasSuffix(p.Type, "PRIVATE KEY") {
			key = memguard.NewEnclave(p.Bytes)
			return
		}
		pembytes = rest
	}
}

// ReadPrivateKeyPEM reads a unencrypted, PEM-encoded private key as a memguard Enclave
func ReadPrivateKeyPEM(h host.Host, pt string) (key *memguard.Enclave, err error) {
	pembytes, err := h.ReadFile(pt)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(pembytes)
		if p == nil {
			return nil, fmt.Errorf("cannot locate private key in %s", pt)
		}
		if strings.HasSuffix(p.Type, "PRIVATE KEY") {
			key = memguard.NewEnclave(pem.EncodeToMemory(p))
			return
		}
		pembytes = rest
	}
}

// WritePrivateKey writes a DER encoded private key as a PKCS#8 encoded
// PEM file to path on host h. sets file permissions to 0600 (before
// umask)
func WritePrivateKey(h host.Host, pt string, key *memguard.Enclave) (err error) {
	l, _ := key.Open()
	defer l.Destroy()
	pembytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: l.Bytes(),
	})
	return h.WriteFile(pt, pembytes, 0600)
}

// CreateCertificateAndKey is a wrapper to create a new certificate
// given the signing cert and key and an optional private key to (re)use
// for the certificate creation. Returns a certificate and private key.
// Keys are usually PKCS#8 encoded and so need parsing after unsealing.
func CreateCertificateAndKey(template, parent *x509.Certificate, signingKeyDER, existingKeyDER *memguard.Enclave) (cert *x509.Certificate, certKeyDER *memguard.Enclave, err error) {
	var certBytes []byte
	// var certKey *rsa.PrivateKey

	if template != parent && signingKeyDER == nil {
		err = errors.New("parent key empty but not self-signing")
		return
	}

	certKeyDER = existingKeyDER
	if certKeyDER == nil {
		keytype := PrivateKeyType(signingKeyDER)
		if keytype == "" {
			keytype = DefaultKeyType
		}
		certKeyDER, err = NewPrivateKey(keytype)
		if err != nil {
			panic(err)
		}
	}

	// default the signingKey to the certKey (for self-signed root)
	signingKey, certPubKey, err := ParseKey(certKeyDER)
	if err != nil {
		return
	}

	if signingKeyDER != nil {
		signingKey, _, err = ParseKey(signingKeyDER)
		if err != nil {
			certKeyDER = nil
			return
		}
	}

	if certBytes, err = x509.CreateCertificate(rand.Reader, template, parent, certPubKey, signingKey); err != nil {
		certKeyDER = nil
		return
	}

	if cert, err = x509.ParseCertificate(certBytes); err != nil {
		certKeyDER = nil
		return
	}

	return
}

// PrivateKeyType returns the type of the DER encoded private key,
// suitable for use to NewPrivateKey
func PrivateKeyType(der *memguard.Enclave) (keytype string) {
	if der == nil {
		return
	}
	key, _, err := ParseKey(der)
	if err != nil {
		return
	}

	switch key.(type) {
	case *rsa.PrivateKey:
		return "rsa"
	case *ecdsa.PrivateKey:
		return "ecdsa"
	case *ecdh.PrivateKey:
		return "ecdh"
	case ed25519.PrivateKey: // not a pointer
		return "ed2559"
	default:
		return ""
	}
}

// NewPrivateKey returns a PKCS#8 DER encoded private key as an enclave.
func NewPrivateKey(keytype string) (der *memguard.Enclave, err error) {
	var privateKey any
	switch keytype {
	case "rsa":
		privateKey, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return
		}
	case "ecdsa":
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return
		}
	case "ed25519":
		_, privateKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			return
		}
	case "ecdh":
		var ecdsaKey *ecdsa.PrivateKey
		ecdsaKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return
		}
		privateKey, err = ecdsaKey.ECDH()
		if err != nil {
			return
		}
	default:
		err = fmt.Errorf("%w unsupported key type %s", os.ErrInvalid, keytype)
		return
	}

	key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return
	}
	der = memguard.NewEnclave(key)
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
			return os.ErrExist
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
		MaxPathLen:            -1,
	}

	privateKeyPEM, err := NewPrivateKey(keytype)
	if err != nil {
		return
	}

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
			return os.ErrExist
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
		MaxPathLen:            0,
		MaxPathLenZero:        true,
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
