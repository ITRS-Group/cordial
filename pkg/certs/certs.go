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
	"os"
	"path"
	"slices"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/host"
)

const (
	PEMExtension = "pem"
	KEYExtension = "key"
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
// host h at path. If the files cannot be read an error is returned. If no
// certificates are found in the file then no error is returned.
func ReadCertificates(h host.Host, path string) (certs []*x509.Certificate, err error) {
	if path == "" {
		err = os.ErrInvalid
		return
	}

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

	existingCerts, err := ReadCertificates(h, path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
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

	allCerts, err := ReadCertificates(h, path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("reading existing root certificates failed")
		return false, err
	}
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
func WriteCertificates(h host.Host, certpath string, certs ...*x509.Certificate) (err error) {
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
	h.MkdirAll(path.Dir(certpath), 0755)
	return h.WriteFile(certpath, data, 0644)
}

// ReadCertPool returns a certificate pool loaded from the file on host
// h at path. If there is any error a nil pointer is returned.
func ReadCertPool(h host.Host, path string) (pool *x509.CertPool) {
	pool = x509.NewCertPool()
	if data, err := h.ReadFile(path); err == nil {
		if ok := pool.AppendCertsFromPEM(data); !ok {
			return nil
		}
	}
	return
}

// ReadTrustedCertificates reads multiple trusted root certificates from
// the specified files and returns a combined *x509.CertPool. Any
// certificates that are not valid root CAs are ignored.
func ReadTrustedCertificates(certFiles ...string) (pool *x509.CertPool, n int) {
	pool = x509.NewCertPool()
	for _, cf := range certFiles {
		certSlice, err := ReadCertificates(host.Localhost, cf)
		if err != nil {
			continue
		}
		for _, c := range certSlice {
			if ValidRootCA(c) {
				pool.AddCert(c)
				n++
			}
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
// writes it locally with dir and file basefilepath with .pem and .key
// extensions. If overwrite is true then any existing certificate and
// key is overwritten.
func CreateRootCert(basefilepath string, cn string, keytype KeyType) (err error) {
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

	if err = WriteCertificates(host.Localhost, basefilepath+".pem", root); err != nil {
		return
	}
	if err = WritePrivateKey(host.Localhost, basefilepath+".key", key); err != nil {
		return
	}

	return
}

// CreateSigningCert creates a new signing certificate and private key
// with the path and file base name basefilepath. You must provide a
// valid root certificate and key in rootbasefilepath. If overwrite is
// true than any existing cert and key are overwritten.
func CreateSigningCert(basefilepath string, rootbasefilepath string, cn string) (err error) {
	var signer *x509.Certificate

	template := Template(cn,
		Days(5*365),
		IsCA(),
		BasicConstraintsValid(),
		KeyUsage(x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign|x509.KeyUsageCRLSign),
		ExtKeyUsage(),
		MaxPathLen(0),
	)

	rootCert, err := ReadCertificate(host.Localhost, rootbasefilepath+".pem")
	if err != nil {
		return
	}

	rootKey, err := ReadPrivateKey(host.Localhost, rootbasefilepath+".key")
	if err != nil {
		return
	}

	signer, key, err := CreateCertificateAndKey(template, rootCert, rootKey)
	if err != nil {
		return
	}

	if err = WriteCertificates(host.Localhost, basefilepath+".pem", signer); err != nil {
		return
	}
	if err = WritePrivateKey(host.Localhost, basefilepath+".key", key); err != nil {
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
		(cert.MaxPathLen != 0 || (cert.MaxPathLen == 0 && cert.MaxPathLenZero)) &&
		cert.NotBefore.Before(time.Now()) &&
		cert.NotAfter.After(time.Now())
}

// ValidLeafCert returns true if the provided certificate is a valid
// leaf certificate. A valid leaf certificate is not a CA, has the
// DigitalSignature key usage, has at least one extended key usage,
// and is between its NotBefore and NotAfter dates.
func ValidLeafCert(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}
	return !cert.IsCA &&
		cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 &&
		cert.ExtKeyUsage != nil &&
		cert.NotBefore.Before(time.Now()) &&
		cert.NotAfter.After(time.Now())
}

// Verify attempts to verify the provided certificate chain. It returns
// the leaf certificate, any intermediates and the root certificate
// found in the chain. It returns an error if verification fails. The
// first certificate in the certs provided is assumed to be the leaf
// certificate. The order of the remaining certificates does not matter.
func Verify(cert ...*x509.Certificate) (leaf *x509.Certificate, intermediates []*x509.Certificate, root *x509.Certificate, err error) {
	log.Debug().Msgf("verifying %d certificates", len(cert))

	for _, c := range cert {
		switch {
		case ValidLeafCert(c):
			log.Debug().Msgf("found valid leaf certificate: %s", c.Subject.CommonName)
			if leaf != nil {
				err = errors.New("multiple leaf certificates found")
				return
			}
			leaf = c
		case ValidRootCA(c):
			log.Debug().Msgf("found valid root CA certificate: %s", c.Subject.CommonName)
			if root != nil {
				err = errors.New("multiple root certificates found")
				return
			}
			root = c
		case ValidSigningCA(c):
			log.Debug().Msgf("found valid intermediate CA certificate: %s", c.Subject.CommonName)
			intermediates = append(intermediates, c)
		default:
			err = fmt.Errorf("certificate %q is not valid", c.Subject.CommonName)
			return
		}
	}

	if leaf == nil {
		err = errors.New("no valid leaf certificate found")
		return
	}

	opts := x509.VerifyOptions{}
	if root != nil {
		opts.Roots = x509.NewCertPool()
		opts.Roots.AddCert(root)
	}
	if len(intermediates) > 0 {
		opts.Intermediates = x509.NewCertPool()
		for _, ic := range intermediates {
			opts.Intermediates.AddCert(ic)
		}
	}

	chains, err := leaf.Verify(opts)
	if err != nil {
		log.Debug().Msgf("certificate verification failed: %v", err)
		return
	}
	if len(chains) == 0 || len(chains[0]) == 0 {
		err = errors.New("no valid certificate chain found")
		return
	}
	leaf = chains[0][0]
	if len(chains[0]) > 2 {
		intermediates = chains[0][1 : len(chains[0])-1]
	} else {
		intermediates = nil
	}
	if len(chains[0]) >= 2 {
		root = chains[0][len(chains[0])-1]
	} else {
		root = nil
	}
	return
}
