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
	"os"
	"strings"

	"github.com/awnumar/memguard"
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

// ReadRootCertPool reads a local PEM file containing trusted
// root certificates returns a *x509.CertPool and the number of valid
// root CAs found. Any certificates that are not valid root CAs are
// skipped.
func ReadRootCertPool(h host.Host, path string) (pool *x509.CertPool, ok bool) {
	pool = x509.NewCertPool()
	trustedCerts, err := ReadCertificates(h, path)
	if err != nil {
		return
	}
	for _, c := range trustedCerts {
		if IsValidRootCA(c) {
			pool.AddCert(c)
			ok = true
		}
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
