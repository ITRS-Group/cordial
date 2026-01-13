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
		return c == nil || !IsValidRootCA(c)
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
		return !IsValidRootCA(c)
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
		if cert == nil || cert.NotAfter.Before(time.Now()) {
			continue
		}

		data = append(data, CertificateComments(cert)...)

		p := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		data = append(data, p...)
	}

	if len(data) == 0 {
		return os.ErrInvalid
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

	roots, err := ReadCertificates(host.Localhost, rootbasefilepath+".pem")
	if err != nil {
		return
	}
	rootCert := roots[0]

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
	l, _ := key.Open()
	defer l.Destroy()

	data := PrivateKeyComments(key)
	data = append(data, pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: l.Bytes(),
	})...)

	return h.WriteFile(path, data, 0600)
}

// CertificateComments returns a byte slice containing comments about
// the given x509 certificate. If titles are provided they are included
// at the top of the comments, if not a default title of "Certificate"
// is used.
func CertificateComments(cert *x509.Certificate, titles ...string) []byte {
	output := &bytes.Buffer{}

	if len(titles) > 0 {
		for _, title := range titles {
			output.WriteString("# " + title + "\n")
		}
	} else {
		output.WriteString("# Certificate\n")
	}
	output.WriteString("#\n")
	output.WriteString("#   Subject: ")
	output.WriteString(cert.Subject.String())
	output.WriteString("\n#    Issuer: ")
	output.WriteString(cert.Issuer.String())
	output.WriteString("\n#   Expires: ")
	output.WriteString(cert.NotAfter.Format(time.RFC3339))
	output.WriteString("\n#    Serial: ")
	output.WriteString(cert.SerialNumber.String())
	output.WriteRune('\n')

	fmt.Fprintf(output, "#      SHA1: %X\n", sha1.Sum(cert.Raw))
	fmt.Fprintf(output, "#    SHA256: %X\n#\n", sha256.Sum256(cert.Raw))

	return output.Bytes()
}

// PrivateKeyComments returns a byte slice containing comments about
// the given private key. If titles are provided they are included
// at the top of the comments, if not a default title of "Private Key"
// is used.
func PrivateKeyComments(key *memguard.Enclave, titles ...string) []byte {
	output := &bytes.Buffer{}

	if len(titles) > 0 {
		for _, title := range titles {
			output.WriteString("# " + title + "\n")
		}
	} else {
		output.WriteString("# Private Key\n")
	}
	output.WriteString("#\n")
	output.WriteString("#   Key Type: " + string(PrivateKeyType(key)) + "\n#\n")

	return output.Bytes()
}
