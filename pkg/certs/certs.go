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
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/awnumar/memguard"

	"github.com/itrs-group/cordial/pkg/host"
)

// ReadCertificate reads a PEM encoded cert from path on host h, return
// the first one found. The returned certificate is not verified or
// validated beyond that of the underlying Go x509 package parsing
// functions.
func ReadCertificate(h host.Host, path string) (cert *x509.Certificate, err error) {
	data, err := h.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(data)
		if p == nil {
			return nil, fmt.Errorf("cannot locate certificate in %q", path)
		}
		if p.Type == "CERTIFICATE" {
			return x509.ParseCertificate(p.Bytes)
		}
		data = rest
	}
}

// ReadCertificates reads and decodes all certificates from the PEM file on
// host h at path. If the files cannot be read or no certificates found
// then an empty slice is returned.
func ReadCertificates(h host.Host, path string) (certs []*x509.Certificate) {
	data, err := h.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(data)
		if p == nil {
			break
		}
		if p.Type == "CERTIFICATE" {
			if c, err := x509.ParseCertificate(p.Bytes); err == nil { // no error
				certs = append(certs, c)
			}
		}
		data = rest
	}

	return
}

// UpdateCACertsFile updates the certificate chain file at the specified
// path on the given host. It ensures that all provided certificates are
// present in the file, appending any that are missing. If the file does
// not exist or is empty, it will be created with the provided
// certificates. Returns ok set to true if the file was updated
// (certificates added or file created), false if no changes were made.
// Returns an error if writing the certificate chain fails.
//
// Root CA or expired certificates in the provided certs slice are
// ignored. Additionally, any non-CA or expired certificates already
// present in the existing chain file are removed.
//
// The caller is responsible for locking access to the chain file if
// concurrent access is possible.
func UpdateCACertsFile(h host.Host, path string, certs ...*x509.Certificate) (ok bool, err error) {
	// remove nil, non-CA or expired certificates from certs
	certs = slices.DeleteFunc(certs, func(c *x509.Certificate) bool {
		return c == nil || !ValidSigningCA(c)
	})

	existingCerts := ReadCertificates(h, path)
	if len(existingCerts) == 0 {
		// if no existing certs just write the new ones
		return true, WriteCertificates(h, path, certs...)
	}

	// remove non-CA or expired certs
	existingCerts = slices.DeleteFunc(existingCerts, func(c *x509.Certificate) bool {
		return !ValidSigningCA(c)
	})

	added := false
	for _, cert := range certs {
		// skip duplicates
		if slices.ContainsFunc(existingCerts, func(c *x509.Certificate) bool {
			return cert.Equal(c)
		}) {
			continue
		}
		existingCerts = append(existingCerts, cert)
		added = true
	}
	if !added {
		return false, nil
	}

	return true, WriteCertificates(h, path, existingCerts...)
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
		return c == nil || !ValidRootCA(c)
	})

	allCerts := ReadCertificates(h, path)
	if allCerts == nil {
		return true, WriteCertificates(h, path, certs...)
	}

	// remove non-root CA or expired certs
	allCerts = slices.DeleteFunc(allCerts, func(c *x509.Certificate) bool {
		return !ValidRootCA(c)
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
// from writing the resulting file are returned. Certificates are written
// in the order provided.
func WriteCertificates(h host.Host, path string, certs ...*x509.Certificate) (err error) {
	var data []byte
	for _, cert := range certs {
		// validate cert as not expired, but it may still be before its
		// NotBefore date
		if cert == nil || cert.NotAfter.Before(time.Now()) {
			continue
		}

		p := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		data = append(data, p...)
	}
	return h.WriteFile(path, data, 0644)
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

// CreateCertificateAndKey is a wrapper to create a new certificate
// given the signing cert and key. Returns a certificate and private key.
// Keys are usually PKCS#8 encoded and so need parsing after unsealing.
func CreateCertificateAndKey(template, parent *x509.Certificate, signerKey *memguard.Enclave) (cert *x509.Certificate, key *memguard.Enclave, err error) {
	var certBytes []byte
	var pub crypto.PublicKey

	if template != parent && signerKey == nil {
		err = errors.New("parent key empty but not self-signing")
		return
	}

	// create a new key of the same type as the signing cert key or use a default type
	keytype := privateKeyType(signerKey)
	if keytype == "" {
		keytype = DefaultKeyType
	}

	key, pub, err = NewPrivateKey(keytype)
	if err != nil {
		panic(err)
	}

	priv, err := PrivateKey(signerKey)
	if err != nil {
		key = nil
		return
	}

	if certBytes, err = x509.CreateCertificate(rand.Reader, template, parent, pub, priv); err != nil {
		key = nil
		return
	}

	if cert, err = x509.ParseCertificate(certBytes); err != nil {
		key = nil
		return
	}

	return
}

// CreateRootCert creates a new root certificate and private key and
// saves it with dir and file basefilepath with .pem and .key extensions. If
// overwrite is true then any existing certificate and key is
// overwritten.
func CreateRootCert(h host.Host, basefilepath string, cn string, keytype KeyType) (err error) {
	var root *x509.Certificate

	template := Template(cn,
		Days(10*365),
		IsCA(),
		BasicConstraintsValid(),
		KeyUsage(x509.KeyUsageCertSign|x509.KeyUsageCRLSign),
		ExtKeyUsage(),
		MaxPathLen(1),
	)

	privateKey, _, err := NewPrivateKey(keytype)
	if err != nil {
		return
	}

	root, key, err := CreateCertificateAndKey(template, template, privateKey)
	if err != nil {
		return
	}

	if err = WriteCertificates(h, basefilepath+".pem", root); err != nil {
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
func CreateSigningCert(h host.Host, basefilepath string, rootbasefilepath string, cn string) (err error) {
	var signer *x509.Certificate

	template := Template(cn,
		Days(5*365),
		IsCA(),
		BasicConstraintsValid(),
		KeyUsage(x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign|x509.KeyUsageCRLSign),
		ExtKeyUsage(),
		MaxPathLen(0),
	)

	rootCert, err := ReadCertificate(h, rootbasefilepath+".pem")
	if err != nil {
		return
	}
	rootKey, err := ReadPrivateKey(h, rootbasefilepath+".key")
	if err != nil {
		return
	}

	signer, key, err := CreateCertificateAndKey(template, rootCert, rootKey)
	if err != nil {
		return
	}

	if err = WriteCertificates(h, basefilepath+".pem", signer); err != nil {
		return
	}
	if err = WritePrivateKey(h, basefilepath+".key", key); err != nil {
		return
	}

	return
}

// ValidRootCA returns true if the provided certificate is a valid root
// CA certificate. A valid root CA is self-signed and has appropriate
// basic constraints as well as either an empty AuthorityKeyId or one
// that matches the SubjectKeyId. It must also be between its NotBefore
// and NotAfter dates.
func ValidRootCA(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}
	return cert.IsCA &&
		cert.BasicConstraintsValid &&
		cert.Subject.CommonName == cert.Issuer.CommonName &&
		(cert.AuthorityKeyId == nil || slices.Equal(cert.AuthorityKeyId, cert.SubjectKeyId)) &&
		cert.NotBefore.Before(time.Now()) &&
		cert.NotAfter.After(time.Now())
}

// ValidSigningCA returns true if the provided certificate is a valid
// signing CA certificate. A valid signing CA is a CA certificate with
// appropriate basic constraints for signing (MaxPathLenZero) and is
// between its NotBefore and NotAfter dates.
func ValidSigningCA(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}
	return cert.IsCA &&
		cert.BasicConstraintsValid &&
		cert.MaxPathLen == 0 &&
		cert.MaxPathLenZero &&
		cert.NotBefore.Before(time.Now()) &&
		cert.NotAfter.After(time.Now())
}
