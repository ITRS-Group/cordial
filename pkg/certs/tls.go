/*
Copyright © 2025 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// The certs package provides functions for handling TLS certificates and keys.
package certs

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
	"slices"
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
	derbytes, err := h.ReadFile(certfile)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(derbytes)
		if p == nil {
			return nil, fmt.Errorf("cannot locate certificate in %s", certfile)
		}
		if p.Type == "CERTIFICATE" {
			return x509.ParseCertificate(p.Bytes)
		}
		derbytes = rest
	}
}

// ParseCertificates reads a PEM encoded file from host h and returns
// all the certificates found (using the same rules as
// x509.ParseCertificates). The returned certificates are not verified
// or validated beyond that of the underlying Go x509 package parsing
// functions.
func ParseCertificates(h host.Host, p string) (certs []*x509.Certificate, err error) {
	derbytes, err := h.ReadFile(p)
	if err != nil {
		return
	}

	return x509.ParseCertificates(derbytes)
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
// if that fails and finally as SEC1 (EC). Using this over the more
// general ParseKey() ensures the decoded private key is not returned to
// the caller when not required.
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

// MatchKey tests the slice DER encoded private keys against the x509
// cert and returns the index of the first match, or -1 if none of the
// keys match.
func MatchKey(cert *x509.Certificate, keys []*memguard.Enclave) int {
	for i, key := range keys {
		if pubkey, err := PublicKey(key); err == nil { // if ok then compare
			// ensure we have an Equal() method on the opaque key
			if k, ok := pubkey.(interface{ Equal(crypto.PublicKey) bool }); ok {
				if k.Equal(cert.PublicKey) {
					return i
				}
			}
		}
	}
	return -1
}

// UpdateCACertsFile updates the certificate chain file at the
// specified path on the given host. It ensures that all provided
// certificates are present in the file, appending any that are missing.
// If the file does not exist or is empty, it will be created with the
// provided certificates. Returns true if the file was updated
// (certificates added or file created), false if no changes were made.
// Returns an error if writing the certificate chain fails.
//
// The caller is responsible for locking access to the chain file if
// concurrent access is possible.
func UpdateCACertsFile(h host.Host, path string, certs ...*x509.Certificate) (updated bool, err error) {
	// remove nil, non-CA or expired certificates from certs
	certs = slices.DeleteFunc(certs, func(c *x509.Certificate) bool {
		return c == nil || !c.IsCA || c.NotAfter.Before(time.Now())
	})

	allCerts := ReadCertificates(h, path)
	if allCerts == nil {
		return true, WriteCertificates(h, path, certs...)
	}

	// remove non-CA or expired certs
	allCerts = slices.DeleteFunc(allCerts, func(c *x509.Certificate) bool {
		return !c.IsCA || c.NotAfter.Before(time.Now())
	})

	added := false
	for _, cert := range certs {
		// skip duplicates
		if slices.ContainsFunc(allCerts, func(c *x509.Certificate) bool {
			return cert.Equal(c)
		}) {
			continue
		}
		allCerts = append(allCerts, cert)
		added = true
	}
	if !added {
		return false, nil
	}

	return true, WriteCertificates(h, path, allCerts...)
}

// UpdateRootCertsFile updates the root certificate file at the
// specified path on the given host. It ensures that all provided
// certificates are present in the file, appending any that are missing.
// If the file does not exist or is empty, it will be created with the
// provided certificates. Returns true if the file was updated
// (certificates added or file created), false if no changes were made.
// Returns an error if writing the root certificate file fails.
//
// Any non-root CA or expired certificates in the provided certs slice
// are ignored. Additionally, any non-root CA or expired certificates already
// present in the existing root certificate file are removed.
//
// The caller is responsible for locking access to the root cert file if
// concurrent access is possible.
func UpdateRootCertsFile(h host.Host, path string, certs ...*x509.Certificate) (updated bool, err error) {
	// remove nil, non-root CA or expired certificates from certs
	certs = slices.DeleteFunc(certs, func(c *x509.Certificate) bool {
		return c == nil || !IsRootCA(c) || c.NotAfter.Before(time.Now())
	})

	allCerts := ReadCertificates(h, path)
	if allCerts == nil {
		return true, WriteCertificates(h, path, certs...)
	}

	// remove non-root CA or expired certs
	allCerts = slices.DeleteFunc(allCerts, func(c *x509.Certificate) bool {
		return !IsRootCA(c) || c.NotAfter.Before(time.Now())
	})

	added := false
	for _, cert := range certs {
		// skip duplicates
		if slices.ContainsFunc(allCerts, func(c *x509.Certificate) bool {
			return cert.Equal(c)
		}) {
			continue
		}
		allCerts = append(allCerts, cert)
		added = true
	}
	if !added {
		return false, nil
	}

	return true, WriteCertificates(h, path, allCerts...)
}

// WriteCertificates concatenate certs in PEM format and writes to path
// on host h. Certificates that are expired are skipped, only errors
// from writing the resulting file are returned
func WriteCertificates(h host.Host, path string, certs ...*x509.Certificate) (err error) {
	var pembytes []byte
	for _, cert := range certs {
		// validate cert as not expired
		if cert == nil || cert.NotAfter.Before(time.Now()) {
			continue
		}

		p := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		pembytes = append(pembytes, p...)
	}
	return h.WriteFile(path, pembytes, 0644)
}

// ReadCertificate reads the first PEM encoded certificate from file at
// path p on host h and returns the block of DER encoded data.
func ReadCertificate(h host.Host, path string) (der []byte, err error) {
	b, err := h.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(b)
		if p == nil {
			return nil, fmt.Errorf("cannot locate certificate in %s", p)
		}
		if p.Type == "CERTIFICATE" {
			// reencode for use
			return pem.EncodeToMemory(p), nil
		}
		b = rest
	}
}

// ReadCertPool returns a certificate pool loaded from the file on host
// h at path. If there is any error a nil pointer is returned.
func ReadCertPool(h host.Host, path string) (pool *x509.CertPool) {
	pool = x509.NewCertPool()
	if chain, err := h.ReadFile(path); err == nil {
		if ok := pool.AppendCertsFromPEM(chain); !ok {
			return nil
		}
	}
	return
}

// ReadCerts reads and decodes all certificates from the PEM file on
// host h at path. If the files cannot be read or no certificates found
// then an empty slice is returned.
func ReadCertificates(h host.Host, path string) (certs []*x509.Certificate) {
	pembytes, err := h.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(pembytes)
		if p == nil {
			break
		}
		if p.Type == "CERTIFICATE" {
			if c, err := x509.ParseCertificate(p.Bytes); err == nil { // no error
				certs = append(certs, c)
			}
		}
		pembytes = rest
	}

	return
}

// ReadPrivateKey reads file on host h as an unencrypted,
// PEM-encoded private key and saves the der format key in a
// memguard.Enclave
func ReadPrivateKey(h host.Host, file string) (key *memguard.Enclave, err error) {
	b, err := h.ReadFile(file)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(b)
		if p == nil {
			return nil, fmt.Errorf("cannot locate private key")
		}
		if strings.HasSuffix(p.Type, "PRIVATE KEY") {
			key = memguard.NewEnclave(p.Bytes)
			return
		}
		b = rest
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
// given the signing cert and key. Returns a certificate and private key.
// Keys are usually PKCS#8 encoded and so need parsing after unsealing.
func CreateCertificateAndKey(template, parent *x509.Certificate, signingKeyDER *memguard.Enclave) (cert *x509.Certificate, key *memguard.Enclave, err error) {
	var certBytes []byte
	// var certKey *rsa.PrivateKey

	if template != parent && signingKeyDER == nil {
		err = errors.New("parent key empty but not self-signing")
		return
	}

	// create a new key of the same type as the signing cert key or use a default type
	keytype := privateKeyType(signingKeyDER)
	if keytype == "" {
		keytype = DefaultKeyType
	}

	key, err = NewPrivateKey(keytype)
	if err != nil {
		panic(err)
	}

	// default the signingKey to the certKey (for self-signed root)
	signingKey, certPubKey, err := ParseKey(key)
	if err != nil {
		return
	}

	if signingKeyDER != nil {
		signingKey, _, err = ParseKey(signingKeyDER)
		if err != nil {
			key = nil
			return
		}
	}

	if certBytes, err = x509.CreateCertificate(rand.Reader, template, parent, certPubKey, signingKey); err != nil {
		key = nil
		return
	}

	if cert, err = x509.ParseCertificate(certBytes); err != nil {
		key = nil
		return
	}

	return
}

// privateKeyType returns the type of the DER encoded private key,
// suitable for use to NewPrivateKey
func privateKeyType(der *memguard.Enclave) (keytype string) {
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
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            -1,
	}

	privateKeyPEM, err := NewPrivateKey(keytype)
	if err != nil {
		return
	}

	cert, key, err := CreateCertificateAndKey(template, template, privateKeyPEM)
	if err != nil {
		return
	}

	if err = WriteCertificates(h, basefilepath+".pem", cert); err != nil {
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
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
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

	cert, key, err := CreateCertificateAndKey(&template, rootCert, rootKey)
	if err != nil {
		return
	}

	if err = WriteCertificates(h, basefilepath+".pem", cert); err != nil {
		return
	}
	if err = WritePrivateKey(h, basefilepath+".key", key); err != nil {
		return
	}

	return
}

// IsRootCA returns true if the provided certificate is a valid root
// CA certificate.
func IsRootCA(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}
	return cert.IsCA &&
		cert.BasicConstraintsValid &&
		cert.Subject.CommonName == cert.Issuer.CommonName &&
		(cert.AuthorityKeyId == nil || slices.Equal(cert.AuthorityKeyId, cert.SubjectKeyId))
}
