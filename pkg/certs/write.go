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
	"io"
	"os"
	"path"
	"slices"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/host"
)

// UpdateCACertsFiles appends the given root certificates to the
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
func UpdateCACertsFiles(h host.Host, basePath string, roots ...*x509.Certificate) (updated bool, err error) {
	// remove nil, non-root CA or expired certificates from certs
	roots = slices.DeleteFunc(roots, func(c *x509.Certificate) bool {
		return c == nil || !IsValidRootCA(c)
	})

	allCerts, err := ReadCertificates(h, basePath+PEMExtension)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("reading existing root certificates failed")
		return false, err
	}
	log.Debug().Msgf("found %d root certificates to sync", len(roots))
	if allCerts == nil {
		if err = WriteCertificates(h, basePath+PEMExtension, roots...); err != nil {
			return false, err
		}
		if err = WriteTrustStore(h, basePath+KeystoreExtension, nil, roots...); err != nil {
			return false, err
		}
		return true, nil
	}

	// remove non-root CA or expired certs
	allCerts = slices.DeleteFunc(allCerts, func(c *x509.Certificate) bool {
		return !IsValidRootCA(c)
	})
	log.Debug().Msgf("existing root certificates contains %d valid root CAs", len(allCerts))

	added := false
	for _, cert := range roots {
		// skip duplicates
		if slices.ContainsFunc(allCerts, func(c *x509.Certificate) bool {
			return cert.Equal(c)
		}) {
			continue
		}
		allCerts = append(allCerts, cert)
		added = true
	}

	log.Debug().Msg("updating truststore with root certificates")
	if err = WriteTrustStore(h, basePath+KeystoreExtension, nil, roots...); err != nil {
		return false, err
	}

	if !added {
		return false, nil
	}

	if err = WriteCertificates(h, basePath+PEMExtension, allCerts...); err != nil {
		return false, err
	}

	return true, nil
}

// WriteCertificates concatenate certs in PEM format and writes to path
// on host h. Certificates that are expired are skipped, only errors
// from writing the resulting file are returned. Certificates are written
// in the order provided. Directories in the path are created with 0755 permissions
// if they do not already exist. The certificate file is created with
// 0644 permissions (before umask).
func WriteCertificates(h host.Host, certpath string, certs ...*x509.Certificate) (err error) {
	var b bytes.Buffer
	if _, err = WriteCertificatesTo(&b, certs...); err != nil {
		return err
	}

	if err = h.MkdirAll(path.Dir(certpath), 0755); err != nil {
		return err
	}

	return h.WriteFile(certpath, b.Bytes(), 0644)
}

// WriteCertificatesTo writes the given certificates in PEM format to
// the provided io.Writer. Certificates that are expired are skipped.
// The total number of bytes written and any error encountered are
// returned.
func WriteCertificatesTo(w io.Writer, certs ...*x509.Certificate) (n int, err error) {
	var m int

	for _, cert := range certs {
		if cert == nil || cert.NotAfter.Before(time.Now()) {
			continue
		}

		if n, err = w.Write(CertificateComments(cert)); err != nil {
			return
		}

		p := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		if m, err = w.Write(p); err != nil {
			return
		}
		n += m
	}
	return
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

	privateKey, _, err := GenerateKey(keytype)
	if err != nil {
		return
	}

	root, key, err := CreateCertificate(template, template, privateKey)
	if err != nil {
		return
	}

	if err = WriteCertificates(host.Localhost, basefilepath+PEMExtension, root); err != nil {
		return
	}

	if err = WritePrivateKey(host.Localhost, basefilepath+KEYExtension, key); err != nil {
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
func WriteNewSignerCert(basefilepath string, rootCert *x509.Certificate, rootKey *memguard.Enclave, cn string) (signer *x509.Certificate, err error) {
	template := Template(cn,
		Days(5*365),
		IsCA(),
		BasicConstraintsValid(),
		KeyUsage(x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign|x509.KeyUsageCRLSign),
		ExtKeyUsage(),
		MaxPathLen(0),
	)

	signer, key, err := CreateCertificate(template, rootCert, rootKey)
	if err != nil {
		return
	}

	if err = WriteCertificates(host.Localhost, basefilepath+PEMExtension, signer); err != nil {
		return
	}

	if err = WritePrivateKey(host.Localhost, basefilepath+KEYExtension, key); err != nil {
		return
	}

	return
}

func WriteNewSignerCertTo(w io.Writer, rootCert *x509.Certificate, rootKey *memguard.Enclave, cn string) (n int, err error) {
	template := Template(cn,
		Days(5*365),
		IsCA(),
		BasicConstraintsValid(),
		KeyUsage(x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign|x509.KeyUsageCRLSign),
		ExtKeyUsage(),
		MaxPathLen(0),
	)

	signer, key, err := CreateCertificate(template, rootCert, rootKey)
	if err != nil {
		return
	}

	var m int
	if n, err = WriteCertificatesTo(w, signer); err != nil {
		return
	}

	if m, err = WritePrivateKeyTo(w, key); err != nil {
		return
	}
	n += m

	return
}

// WritePrivateKey writes a DER encoded private key as a PKCS#8 encoded
// PEM file to path on host h. sets file permissions to 0600 (before
// umask). Directories in the path are created with 0755 permissions if
// they do not already exist.
func WritePrivateKey(h host.Host, keypath string, key *memguard.Enclave) (err error) {
	var b bytes.Buffer
	if _, err = WritePrivateKeyTo(&b, key); err != nil {
		return err
	}

	if err = h.MkdirAll(path.Dir(keypath), 0755); err != nil {
		return err
	}
	return h.WriteFile(keypath, b.Bytes(), 0600)
}

// WritePrivateKeyTo writes the given private key in PEM format to
// the provided io.Writer. The total number of bytes written and any
// error encountered are returned.
func WritePrivateKeyTo(w io.Writer, key *memguard.Enclave) (n int, err error) {
	var m int

	if n, err = w.Write(PrivateKeyComments(key)); err != nil {
		return
	}

	l, _ := key.Open()
	defer l.Destroy()

	p := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: l.Bytes(),
	})
	if m, err = w.Write(p); err != nil {
		return
	}
	n += m

	return
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
