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
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
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

// AppendTrustedCertsFile appends the given root certificates to the
// root certificate file at the specified path on the given host. It
// ensures that all the given certificates are present in the file,
// appending any that are missing. If the file does not exist or is
// empty, it will be created with the provided certificates. Returns
// true if the file was updated (certificates added or file created),
// false if no changes were made.
//
// Returns an error if writing the file fails.
//
// Any non-root CA or expired certificates in the provided certs slice
// are ignored. Additionally, any non-root CA or expired certificates
// already present in the existing root certificate file are removed.
//
// The caller is responsible for locking access to the root cert file if
// concurrent access is possible.
func AppendTrustedCertsFile(h host.Host, path string, root ...*x509.Certificate) (updated bool, err error) {
	// remove nil, non-root CA or expired certificates from certs
	root = slices.DeleteFunc(root, func(c *x509.Certificate) bool {
		return c == nil || !ValidRootCA(c)
	})

	allCerts, err := ReadCertificates(h, path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("reading existing root certificates failed")
		return false, err
	}
	if allCerts == nil {
		return true, WriteCertificates(h, path, root...)
	}

	// remove non-root CA or expired certs
	allCerts = slices.DeleteFunc(allCerts, func(c *x509.Certificate) bool {
		return !ValidRootCA(c)
	})

	added := false
	for _, cert := range root {
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
		var output bytes.Buffer
		// validate cert as not expired, but it may still be before its
		// NotBefore date
		if cert == nil || cert.NotAfter.Before(time.Now()) {
			continue
		}

		output.WriteString("\n# Certificate\n#\n")
		output.WriteString("#   Subject: " + cert.Subject.String() + "\n")
		output.WriteString("#    Issuer: " + cert.Issuer.String() + "\n")
		output.WriteString("#   Expires: " + cert.NotAfter.Format(time.RFC3339) + "\n")
		output.WriteString("#    Serial: " + cert.SerialNumber.String() + "\n")
		output.WriteString("#      SHA1: " + fmt.Sprintf("%X", sha1.Sum(cert.Raw)) + "\n")
		output.WriteString("#    SHA256: " + fmt.Sprintf("%X", sha256.Sum256(cert.Raw)) + "\n#\n")

		data = append(data, output.Bytes()...)

		p := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		data = append(data, p...)
	}

	h.MkdirAll(path.Dir(certpath), 0755)
	return h.WriteFile(certpath, data, 0644)
}

// WriteNewRootCert creates a new root certificate and private key and
// writes it locally with dir and file basefilepath with .pem and .key
// extensions. If overwrite is true then any existing certificate and
// key is overwritten.
func WriteNewRootCert(basefilepath string, cn string, keytype KeyType) (root *x509.Certificate, err error) {
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

// WriteNewSignerCert creates a new signing certificate and private key
// with the path and file base name basefilepath. You must provide a
// valid root certificate and key in rootbasefilepath. If overwrite is
// true than any existing cert and key are overwritten.
//
// The certificate is returned on success, but the private key is not.
func WriteNewSignerCert(basefilepath string, rootbasefilepath string, cn string) (signer *x509.Certificate, err error) {
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

// WritePrivateKey writes a DER encoded private key as a PKCS#8 encoded
// PEM file to path on host h. sets file permissions to 0600 (before
// umask)
func WritePrivateKey(h host.Host, path string, key *memguard.Enclave) (err error) {
	var output bytes.Buffer

	l, _ := key.Open()
	defer l.Destroy()
	output.WriteString("\n# Private Key\n#\n")
	output.WriteString("#   Key Type: " + string(PrivateKeyType(key)) + "\n#\n")
	data := output.Bytes()

	data = append(data, pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: l.Bytes(),
	})...)

	return h.WriteFile(path, data, 0600)
}
