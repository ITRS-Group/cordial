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
	"crypto/sha1"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"
)

const (
	PEMExtension = "pem"
	KEYExtension = "key"
)

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
	keytype := PrivateKeyType(signerKey)
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

	// add a subject key identifier if missing. this is optional but
	// considered good practise for leaf certs
	// (https://www.rfc-editor.org/rfc/rfc5280.html#section-4.2.1.2)
	if !template.IsCA && template.SubjectKeyId == nil {
		var skid struct {
			Identifier []byte `asn1:"ia5"`
		}
		pubBytes, err := x509.MarshalPKIXPublicKey(pub)
		if err != nil {
			// handle error
		}
		_, err = asn1.Unmarshal(pubBytes, &skid)
		if err != nil {
			// handle error
		}
		hash := sha1.Sum(skid.Identifier)
		template.SubjectKeyId = hash[:]
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

// CertificateBundle holds a leaf certificate and a certificate chain
// along with an optional private key (of the first cert in the chain,
// which is normally a leaf) and a root certificate. If there is no leaf
// certificate then Leaf is nil and FullChain[0] is the first
// intermediate, if any.
//
// Valid should be set to true if the bundle has been verified as
// consistent from FullChain[0] to root and the private key applies
// FullChain[0] (but without revocation checks).
type CertificateBundle struct {
	Leaf      *x509.Certificate
	FullChain []*x509.Certificate
	Root      *x509.Certificate
	Key       *memguard.Enclave
	Valid     bool
}

// ParseCertChain tries to verify the provided certificate chain. It
// returns the leaf certificate, any intermediates and any root
// certificate found in the chain. It returns an error if verification
// fails. The first certificate in the certs provided is assumed to be
// the leaf certificate. The order of the remaining certificates does
// not matter.
func ParseCertChain(cert ...*x509.Certificate) (leaf *x509.Certificate, intermediates []*x509.Certificate, root *x509.Certificate, err error) {
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

// ParsePEM parses PEM formatted data blocks and extracts the first leaf
// (if any) certificate, any CA certs as a chain and a private key as a
// DER encoded *memguard.Enclave.
//
// If no leaf certificate is found, but CA certificates are found, only
// the chain is returned and cert is nil.
//
// The key is matched either to the leaf certificate or, if no leaf,
// then the first member of the chain that matches any private key in
// the bundle.
//
// Encrypted private keys are not supported and will be skipped.
func ParsePEM(data ...[]byte) (cert *x509.Certificate, key *memguard.Enclave, chain []*x509.Certificate, err error) {
	var leaf *x509.Certificate
	var keys []*memguard.Enclave

	if len(data) == 0 {
		err = fmt.Errorf("no data to process")
		return
	}

	for _, pembytes := range data {
		for {
			p, rest := pem.Decode(pembytes)
			if p == nil {
				break
			}

			switch p.Type {
			case "CERTIFICATE":
				var c *x509.Certificate
				c, err = x509.ParseCertificate(p.Bytes)
				if err != nil {
					return
				}
				if c.IsCA {
					chain = append(chain, c)
				} else if leaf == nil {
					// save first leaf
					leaf = c
				}
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				// save all private keys for later matching
				keys = append(keys, memguard.NewEnclave(p.Bytes))
			default:
				err = fmt.Errorf("unsupported PEM type found: %s, skipping", p.Type)
				continue
			}
			pembytes = rest
		}
	}

	if leaf == nil && len(chain) == 0 {
		err = fmt.Errorf("no certificates found")
		return
	}

	// reorder chain to have intermediates in-order first, then root last
	if leaf != nil {
		if _, intermediates, roots, err := ParseCertChain(chain...); err == nil {
			chain = append(intermediates, roots)
		}
	}

	// if we got this far then we can start setting returns
	cert = leaf // which may be nil

	// match key if we have a leaf cert
	if cert != nil {
		if i := IndexPrivateKey(keys, cert); i != -1 {
			key = keys[i]
		}
	} else {
		// no leaf, try to match first cert in chain
		for _, c := range chain {
			if i := IndexPrivateKey(keys, c); i != -1 {
				key = keys[i]
				break
			}
		}
	}

	err = nil
	return
}

// ParsePEM2 parses PEM formatted data blocks and returns a
// CertificateBundle.
//
// Encrypted private keys are not supported and will be skipped.
func ParsePEM2(data ...[]byte) (bundle *CertificateBundle, err error) {
	var leaf *x509.Certificate
	var intermediates []*x509.Certificate
	var roots []*x509.Certificate
	var keys []*memguard.Enclave

	if len(data) == 0 {
		err = fmt.Errorf("no data to process")
		return
	}

	bundle = &CertificateBundle{}

	for _, pembytes := range data {
		for {
			p, rest := pem.Decode(pembytes)
			if p == nil {
				break
			}

			switch p.Type {
			case "CERTIFICATE":
				var c *x509.Certificate
				c, err = x509.ParseCertificate(p.Bytes)
				if err != nil {
					return
				}
				if ValidRootCA(c) {
					roots = append(roots, c)
				} else if ValidSigningCA(c) {
					intermediates = append(intermediates, c)
				} else if ValidLeafCert(c) {
					// save first leaf, ignore others
					if leaf == nil {
						leaf = c
					}
				} else {
					log.Warn().Msgf("certificate %q is not valid, skipping", c.Subject.CommonName)
				}
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				// save all private keys for later matching
				keys = append(keys, memguard.NewEnclave(p.Bytes))
			default:
				err = fmt.Errorf("unsupported PEM type found: %s, skipping", p.Type)
				continue
			}
			pembytes = rest
		}
	}

	if leaf == nil && len(intermediates) == 0 {
		err = fmt.Errorf("no certificates found")
		return
	}

	// prepare for validation, regardless of leaf presence

	var chains [][]*x509.Certificate

	// set up cert pools for verification
	ip := x509.NewCertPool()
	for _, c := range intermediates {
		ip.AddCert(c)
	}
	rp := x509.NewCertPool()
	for _, c := range roots {
		rp.AddCert(c)
	}

	vopts := x509.VerifyOptions{
		Intermediates: ip,
		Roots:         rp,
	}

	firstCert := leaf
	if firstCert == nil && len(intermediates) > 0 {
		firstCert = intermediates[0]
	}
	if firstCert == nil && len(roots) > 0 {
		firstCert = roots[0]
	}

	if firstCert != nil {
		if chains, err = firstCert.Verify(vopts); err != nil {
			log.Debug().Msgf("leaf certificate verification failed: %v", err)
			return
		}

		if len(chains) == 0 || len(chains[0]) == 0 {
			err = errors.New("no valid certificate chain found")
			return
		}

		bundle.Valid = true
		bundle.Leaf = chains[0][0]
		bundle.Root = chains[0][len(chains[0])-1]
		// FullChain does not include root
		bundle.FullChain = append([]*x509.Certificate{}, chains[0][0:len(chains[0])-1]...)
		if i := IndexPrivateKey(keys, bundle.Leaf); i != -1 {
			bundle.Key = keys[i]
		}
	}

	err = nil
	return
}

// func ParseCertificatesAndKeys(data ...[]byte) (certificates []*x509.Certificate, keys []*memguard.Enclave, err error) {
// 	if len(data) == 0 {
// 		return
// 	}

// 	// store them unsorted for now
// 	var ks []*memguard.Enclave
// 	var leaves, intermediates, roots []*x509.Certificate

// 	for _, b := range data {
// 		pembytes := b
// 		for {
// 			p, rest := pem.Decode(pembytes)
// 			if p == nil {
// 				break
// 			}

// 			switch p.Type {
// 			case "CERTIFICATE":
// 				var c *x509.Certificate
// 				c, err = x509.ParseCertificate(p.Bytes)
// 				if err != nil {
// 					return
// 				}
// 				if ValidRootCA(c) {
// 					roots = append(roots, c)
// 				} else if ValidSigningCA(c) {
// 					intermediates = append(intermediates, c)
// 				} else {
// 					leaves = append(leaves, c)
// 				}
// 			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
// 				// save first private key found
// 				ks = append(ks, memguard.NewEnclave(p.Bytes))
// 			default:
// 				err = fmt.Errorf("unsupported PEM type found: %s", p.Type)
// 				return
// 			}
// 			pembytes = rest
// 		}
// 	}

// 	ip := x509.NewCertPool()
// 	for _, c := range intermediates {
// 		ip.AddCert(c)
// 	}
// 	rp := x509.NewCertPool()
// 	for _, c := range roots {
// 		rp.AddCert(c)
// 	}
// 	// sort certs then keys
// 	vopts := x509.VerifyOptions{
// 		Intermediates: ip,
// 		Roots:         rp,
// 	}
// 	for _, c := range leaves {
// 		_, err = c.Verify(vopts)
// 		if err != nil {
// 			continue
// 		}
// 		certificates = append(certificates, c)
// 	}
// 	for _, c := range intermediates {
// 		_, err = c.Verify(vopts)
// 		if err != nil {
// 			continue
// 		}
// 		certificates = append(certificates, c)
// 	}
// 	for _, c := range roots {
// 		_, err = c.Verify(vopts)
// 		if err != nil {
// 			continue
// 		}
// 		certificates = append(certificates, c)
// 	}

// 	return
// }
