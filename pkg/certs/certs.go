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
	"os"
	"slices"
	"time"

	"github.com/awnumar/memguard"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
	"software.sslmate.com/src/go-pkcs12"
)

const (
	PEMExtension      = ".pem"
	KEYExtension      = ".key"
	KeystoreExtension = ".db"
)

// CreateCertificate creates a new certificate and private key given the
// signing cert and key. Returns a certificate and private key. Keys are
// usually PKCS#8 encoded and so need parsing after unsealing.
func CreateCertificate(template, parent *x509.Certificate, signerKey *memguard.Enclave) (cert *x509.Certificate, key *memguard.Enclave, err error) {
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

	key, pub, err = GenerateKey(keytype)
	if err != nil {
		panic(err)
	}

	priv, err := ParsePrivateKey(signerKey)
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

// IsValidRootCA returns true if the provided certificate is a valid root
// CA certificate. A valid root CA is self-signed and has appropriate
// basic constraints as well as either an empty AuthorityKeyId or one
// that matches the SubjectKeyId. It must also be between its NotBefore
// and NotAfter dates.
func IsValidRootCA(cert *x509.Certificate) bool {
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

// IsValidSigningCA returns true if the provided certificate is a valid
// signing CA certificate. A valid signing CA is a CA certificate with
// appropriate basic constraints for signing (MaxPathLenZero) and is
// between its NotBefore and NotAfter dates.
func IsValidSigningCA(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}
	return cert.IsCA &&
		cert.BasicConstraintsValid &&
		(cert.MaxPathLen != 0 || (cert.MaxPathLen == 0 && cert.MaxPathLenZero)) &&
		cert.NotBefore.Before(time.Now()) &&
		cert.NotAfter.After(time.Now())
}

// IsValidLeafCert returns true if the provided certificate is a valid
// leaf certificate. A valid leaf certificate is not a CA, has the
// DigitalSignature key usage, has at least one extended key usage,
// and is between its NotBefore and NotAfter dates.
func IsValidLeafCert(cert *x509.Certificate) bool {
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

// Verify verifies the provided certificates as a chain. It
// returns true if verification is successful. Only one leaf, zero or
// more intermediates and zero or one root certificate should be
// provided. If there is no leaf certificate then the first intermediate
// is used as the leaf. If there is only one or more root certificates
// then the function returns true.
func Verify(certs ...*x509.Certificate) (ok bool) {
	var leaf *x509.Certificate
	var intermediates []*x509.Certificate
	var roots []*x509.Certificate
	var err error

	log.Debug().Msgf("verifying %d certificates", len(certs))

	if len(certs) == 0 {
		return
	}

	for _, c := range certs {
		switch {
		case IsValidLeafCert(c):
			log.Debug().Msgf("found valid leaf certificate: %s", c.Subject.CommonName)
			if leaf != nil && !leaf.Equal(c) {
				err = errors.New("multiple leaf certificates found")
				log.Debug().Err(err).Msg("")
				return
			}
			leaf = c
		case IsValidRootCA(c):
			log.Debug().Msgf("found root CA certificate: %s", c.Subject.CommonName)
			roots = append(roots, c)
		case IsValidSigningCA(c):
			log.Debug().Msgf("found intermediate CA certificate: %s", c.Subject.CommonName)
			intermediates = append(intermediates, c)
		default:
			err = fmt.Errorf("certificate %q is not valid", c.Subject.CommonName)
			return
		}
	}

	if leaf == nil {
		if len(intermediates) > 0 {
			leaf = intermediates[0]
			intermediates = intermediates[1:]
		} else if len(roots) > 0 {
			log.Debug().Msg("only root certificate found, verifying self-signed root")
			return true
		}
	}

	opts := x509.VerifyOptions{}
	if len(roots) > 0 {
		opts.Roots = x509.NewCertPool()
		for _, rc := range roots {
			log.Debug().Msgf("adding root CA certificate %s to pool", rc.Subject.CommonName)
			opts.Roots.AddCert(rc)
		}
	}
	if len(intermediates) > 0 {
		opts.Intermediates = x509.NewCertPool()
		for _, ic := range intermediates {
			log.Debug().Msgf("adding intermediate CA certificate %s to pool", ic.Subject.CommonName)
			opts.Intermediates.AddCert(ic)
		}
	}

	_, err = leaf.Verify(opts)
	if err != nil {
		log.Debug().Msgf("certificate verification failed: %v", err)
		return
	}

	log.Debug().Msg("certificate verification succeeded")
	return true
}

// ParseCertChain tries to verify the provided certificate chain. It
// returns the leaf certificate, any intermediates and any root
// certificate found in the chain. It returns an error if verification
// fails. The first certificate in the certs provided is assumed to be
// the leaf certificate. The order of the remaining certificates does
// not matter.
func ParseCertChain(cert ...*x509.Certificate) (leaf *x509.Certificate, intermediates []*x509.Certificate, root *x509.Certificate, err error) {
	log.Debug().Msgf("parsing %d certificates", len(cert))

	for _, c := range cert {
		switch {
		case IsValidLeafCert(c):
			log.Debug().Msgf("found valid leaf certificate: %s", c.Subject.CommonName)
			if leaf != nil && !leaf.Equal(c) {
				err = errors.New("multiple leaf certificates found")
				return
			}
			leaf = c
		case IsValidRootCA(c):
			log.Debug().Msgf("found root CA certificate: %s", c.Subject.CommonName)
			if root != nil && !root.Equal(c) {
				err = errors.New("multiple root certificates found")
				return
			}
			root = c
		case IsValidSigningCA(c):
			log.Debug().Msgf("found intermediate CA certificate: %s", c.Subject.CommonName)
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

// DecodePEM decodes PEM formatted data and returns the first leaf
// certificate and slices of certificates (intermediates, roots) and
// private keys found. Encrypted private keys and other types of blocks
// are not supported and will be skipped. If there are no certificates
// or private keys found then empty clients are returned without error.
func DecodePEM(data ...[]byte) (leaf *x509.Certificate, intermediates, roots []*x509.Certificate, keys []*memguard.Enclave, err error) {
	var block *pem.Block

	for _, d := range data {
		for {
			block, d = pem.Decode(d)
			if block == nil {
				break
			}

			switch block.Type {
			case "CERTIFICATE":
				var c *x509.Certificate
				c, err = x509.ParseCertificate(block.Bytes)
				if err != nil {
					return
				}
				if IsValidRootCA(c) {
					roots = append(roots, c)
				} else if IsValidSigningCA(c) {
					intermediates = append(intermediates, c)
				} else if IsValidLeafCert(c) {
					// save first leaf, ignore others
					if leaf == nil {
						leaf = c
					}
				} else {
					log.Warn().Msgf("certificate %q is not valid, skipping", c.Subject.CommonName)
				}
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				keys = append(keys, memguard.NewEnclave(block.Bytes))
			default:
				log.Warn().Msgf("unsupported PEM type found: %s, skipping", block.Type)
			}
		}
	}

	return
}

// ParsePEM parses PEM formatted data blocks and returns a
// CertificateBundle. If it contains a verifiable chain it then it sets
// Valid to true. Only if Valid is true can the order of the
// certificates be relied upon.
//
// Encrypted private keys are not supported and will be skipped.
func ParsePEM(data ...[]byte) (bundle *CertificateBundle, err error) {
	var roots []*x509.Certificate
	var chains [][]*x509.Certificate
	var keys []*memguard.Enclave

	if len(data) == 0 {
		err = fmt.Errorf("no data to process")
		return
	}

	bundle = &CertificateBundle{}

	bundle.Leaf, bundle.FullChain, roots, keys, err = DecodePEM(data...)

	if len(roots) > 0 {
		bundle.Root = roots[0]
	}

	if bundle.Leaf == nil && len(bundle.FullChain) == 0 && len(roots) == 0 {
		err = fmt.Errorf("no certificates found")
		return
	}
	if bundle.Leaf == nil {
		switch {
		case len(bundle.FullChain) > 0:
			bundle.Leaf = bundle.FullChain[0]
		case len(roots) > 0:
			bundle.Leaf = bundle.Root
		default:
			err = fmt.Errorf("no certificates available for verification")
			return
		}
	}

	// set the key if we can find one that matches the designated leaf cert
	if i := IndexPrivateKey(keys, bundle.Leaf); i != -1 {
		bundle.Key = keys[i]
	}

	// set up cert pools for verification
	ip := x509.NewCertPool()
	for _, c := range bundle.FullChain {
		ip.AddCert(c)
	}
	rp := x509.NewCertPool()
	for _, c := range roots {
		rp.AddCert(c)
	}

	verifyOpts := x509.VerifyOptions{
		Intermediates: ip,
		Roots:         rp,
	}

	if chains, err = bundle.Leaf.Verify(verifyOpts); err != nil || len(chains) == 0 || len(chains[0]) == 0 {
		// return an unverified bundle
		log.Debug().Msgf("certificate verification for %q failed: %v", bundle.Leaf.Subject.String(), err)
		err = nil
		return
	}

	bundle.Valid = true
	bundle.Leaf = chains[0][0]
	// FullChain does not include root
	bundle.FullChain = append([]*x509.Certificate{}, chains[0][0:len(chains[0])-1]...)
	bundle.Root = chains[0][len(chains[0])-1]

	return
}

func P12ToCertBundle(pfxPath string, password *config.Plaintext) (certBundle *CertificateBundle, err error) {
	pfxData, err := os.ReadFile(pfxPath)
	if err != nil {
		err = fmt.Errorf("failed to read PFX file: %w", err)
		return
	}
	key, c, caCerts, err := pkcs12.DecodeChain(pfxData, password.String())
	if err != nil {
		err = fmt.Errorf("failed to decode PFX file: %w", err)
		return
	}
	pk, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		err = fmt.Errorf("failed to marshal private key: %w", err)
		return
	}
	k := memguard.NewEnclave(pk)

	leaf, intermediates, root, err := ParseCertChain(append([]*x509.Certificate{c}, caCerts...)...)
	if err != nil {
		err = fmt.Errorf("failed to decompose certificate chain: %w", err)
		return
	}
	if leaf == nil || k == nil || !CheckKeyMatch(k, leaf) {
		err = fmt.Errorf("no leaf certificate and/or matching key found in instance bundle")
		return
	}
	certBundle = &CertificateBundle{
		Leaf:      leaf,
		Key:       k,
		FullChain: append([]*x509.Certificate{leaf}, intermediates...),
		Root:      root,
		Valid:     true,
	}
	return
}
