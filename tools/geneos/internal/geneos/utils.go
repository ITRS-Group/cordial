/*
Copyright © 2022 ITRS Group

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

package geneos

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/awnumar/memguard"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
)

// given a path return a cleaned version. If the cleaning results in and
// absolute path or one that tries to ascend the tree then return an
// error
func CleanRelativePath(path string) (clean string, err error) {
	clean = filepath.Clean(path)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "../") {
		log.Debug().Msgf("path %q must be relative and descending only", clean)
		return "", host.ErrInvalidArgs
	}

	return
}

// ReadKey reads a keyfile and saves the PEM encoded key in an
// enclave (without type or headers)
func (r *Host) ReadKey(path string) (key *memguard.Enclave, err error) {
	keyPEM, err := r.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(keyPEM)
		if p == nil {
			return nil, fmt.Errorf("cannot locate RSA private key in %s", path)
		}
		if p.Type == "RSA PRIVATE KEY" {
			key = memguard.NewEnclave(p.Bytes)
			return
		}
		keyPEM = rest
	}
}

// write a private key as PEM to path. sets file permissions to 0600 (before umask)
func (r *Host) WriteKey(path string, key *memguard.Enclave) (err error) {
	l, _ := key.Open()
	defer l.Destroy()
	p := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: l.Bytes(),
	})
	return r.WriteFile(path, p, 0600)
}

// read a PEM encoded cert from path, return the first found as a parsed certificate
func (r *Host) ReadCert(path string) (cert *x509.Certificate, err error) {
	certPEM, err := r.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(certPEM)
		if p == nil {
			return nil, fmt.Errorf("cannot locate certificate in %s", path)
		}
		if p.Type == "CERTIFICATE" {
			return x509.ParseCertificate(p.Bytes)
		}
		certPEM = rest
	}
}

// write cert as PEM to path
func (r *Host) WriteCert(path string, cert *x509.Certificate) (err error) {
	log.Debug().Msgf("write cert to %s", path)
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	return r.WriteFile(path, certPEM, 0644)
}

// concatenate certs and write to path
func (r *Host) WriteCerts(path string, certs ...*x509.Certificate) (err error) {
	log.Debug().Msgf("write certs to %s", path)
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
	return r.WriteFile(path, certsPEM, 0644)
}
