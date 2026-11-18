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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	"github.com/itrs-group/cordial/pkg/host"
)

// ReadCertificates reads and parses all certificates from the PEM file
// on host h at path. If the files cannot be read an error is returned.
// If no certificates are found in the file then no error is returned.
// The returned certificates are not validated beyond the parsing
// functionality of the underlying Go crypto/x509 package.
func ReadCertificates(h host.Host, path string) (certs []*x509.Certificate, err error) {
	if path == "" {
		err = os.ErrInvalid
		return
	}

	data, err := h.ReadFile(path)
	if err != nil {
		return
	}

	return parseCertificates(data), nil
}

// ReadCertificatesFrom reads and parses all certificates from the given
// io.Reader. If the reader cannot be read an error is returned. If no
// certificates are found in the reader then no error is returned. The
// returned certificates are not validated beyond the parsing
// functionality of the underlying Go crypto/x509 package.
func ReadCertificatesFrom(r io.Reader) (certs []*x509.Certificate, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return
	}

	return parseCertificates(data), nil
}

func parseCertificates(data []byte) (certs []*x509.Certificate) {
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			return certs
		}
		if block.Type == "CERTIFICATE" {
			if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
				certs = append(certs, cert)
			}
		}
		data = rest
	}
}

// ReadCertPool returns a certificate pool loaded from the file on host
// h at path. If the file cannot be read or contains no certificates a
// nil pointer is returned.
func ReadCertPool(h host.Host, path string) (pool *x509.CertPool) {
	data, err := h.ReadFile(path)
	if err != nil {
		return nil
	}
	pool = x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(data); !ok {
		return nil
	}
	return
}

// ReadCACertPool reads a local PEM file containing CA certificates and
// returns a *x509.CertPool and the number of valid root CAs found. Any
// certificates that are not valid root CAs are skipped.
func ReadCACertPool(h host.Host, path string) (pool *x509.CertPool, ok bool) {
	pool = x509.NewCertPool()
	caBundle, err := ReadCertificates(h, path)
	if err != nil {
		return
	}
	for _, c := range caBundle {
		if IsValidRootCA(c) {
			pool.AddCert(c)
			ok = true
		}
	}
	return
}

// ReadPrivateKey reads file on host h as an unencrypted,
// PEM-encoded private key and returns the PKCS#8 DER format key. Keys
// in PKCS#1 (RSA) or SEC1 (EC) format are converted to PKCS#8.
// Encrypted private keys are not supported and result in an error.
func ReadPrivateKey(h host.Host, file string) (key PrivateKey, err error) {
	b, err := h.ReadFile(file)
	if err != nil {
		return
	}

	return ReadPrivateKeyFromPEM(b)
}

// ReadPrivateKeyFromPEM reads a PEM-encoded private key from the given
// byte slice and returns it as a PKCS#8 DER format PrivateKey. The input
// data should contain an unencrypted private key in PEM format, either
// PKCS#8, PKCS#1 (RSA) or SEC1 (EC), the last two of which are
// converted to PKCS#8. Encrypted private keys are not supported. If
// the key cannot be found or parsed, an error is returned.
func ReadPrivateKeyFromPEM(pemData []byte) (key PrivateKey, err error) {
	encrypted := false

	for {
		p, rest := pem.Decode(pemData)
		if p == nil {
			if encrypted {
				return nil, fmt.Errorf("cannot use private key: encrypted private keys are not supported")
			}
			return nil, fmt.Errorf("cannot locate private key")
		}
		switch p.Type {
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			if isEncryptedPEM(p) {
				encrypted = true
				break
			}
			if key, err = toPKCS8(p.Bytes); err != nil {
				return nil, fmt.Errorf("invalid %s block: %w", p.Type, err)
			}
			return key, nil
		case "ENCRYPTED PRIVATE KEY":
			encrypted = true
		}
		pemData = rest
	}
}
