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
	"crypto"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"software.sslmate.com/src/go-pkcs12"
)

const (
	// PEMExtension TODO: Describe its purpose and intended use.
	PEMExtension = ".pem"
	// KEYExtension TODO: Describe its purpose and intended use.
	KEYExtension = ".key"
	// KeystoreExtension TODO: Describe its purpose and intended use.
	KeystoreExtension = ".db"
)

// CreateCertificate creates a new certificate and private key given the
// signing cert and key. Returns a certificate and private key. Keys are
// usually PKCS#8 encoded and so need parsing after unsealing.
//
// If template and parent are the same pointer then a self-signed
// certificate is created and is signed with the newly generated key.
// In this case signingKey is only used to select the key type and may
// be nil. The template is never modified.
//
// To create a private key of a specific type, use the WithKeyType
// option. If the signing key is not nil and the WithKeyType option is
// not used then the new key will be of the same type as the signing
// key. If there is no signing key and the WithKeyType option is not
// used then the default key type will be used. See [KeyType] for how
// the "ecdh" key type is handled.
//
// If the template has no SubjectKeyId then one is derived from the new
// public key.
func CreateCertificate(template, parent *x509.Certificate, signingKey PrivateKey, options ...CreateCertOption) (cert *x509.Certificate, key PrivateKey, err error) {
	opts, err := evalCreateCertOptions(options...)
	if err != nil {
		return
	}
	if template == nil || parent == nil {
		err = fmt.Errorf("%w: no certificate template or parent (see NewTemplate for details of template errors)", os.ErrInvalid)
		return
	}

	selfSigned := template == parent
	if !selfSigned && signingKey == nil {
		err = errors.New("parent key empty but not self-signing")
		return
	}

	keytype := opts.keytype
	var signer crypto.Signer
	if signingKey != nil {
		priv, privKeyType, perr := ParsePrivateKey(signingKey)
		if perr != nil {
			err = perr
			return
		}
		if keytype == "" {
			keytype = privKeyType
		}
		if !selfSigned {
			var ok bool
			if signer, ok = priv.(crypto.Signer); !ok {
				err = fmt.Errorf("signing key of type %T cannot sign certificates", priv)
				return
			}
		}
	}
	if keytype == "" {
		keytype = DefaultKeyType
	}

	newKeyDER, pub, err := GenerateKey(keytype)
	if err != nil {
		return nil, nil, fmt.Errorf("generate private key: %w", err)
	}
	defer func() {
		if err != nil {
			clear(newKeyDER)
		}
	}()

	if selfSigned {
		newKey, _, perr := ParsePrivateKey(newKeyDER)
		if perr != nil {
			return nil, nil, perr
		}
		signer = newKey.(crypto.Signer)
	}

	// work on a copy so that the caller's template can be reused
	tmpl := *template
	if len(tmpl.SubjectKeyId) == 0 {
		if tmpl.SubjectKeyId, err = subjectKeyID(pub); err != nil {
			return nil, nil, err
		}
	}
	if selfSigned {
		parent = &tmpl
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, parent, pub, signer)
	if err != nil {
		return nil, nil, err
	}

	if cert, err = x509.ParseCertificate(certBytes); err != nil {
		return nil, nil, err
	}

	return cert, newKeyDER, nil
}

// subjectKeyID returns the SHA-1 hash of the subjectPublicKey bit
// string of pub, as described in RFC 5280, section 4.2.1.2 (1).
func subjectKeyID(pub crypto.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	var spki struct {
		Algorithm        pkix.AlgorithmIdentifier
		SubjectPublicKey asn1.BitString
	}
	if _, err = asn1.Unmarshal(der, &spki); err != nil {
		return nil, err
	}
	hash := sha1.Sum(spki.SubjectPublicKey.Bytes)
	return hash[:], nil
}

func validAt(cert *x509.Certificate, now time.Time) bool {
	return !now.Before(cert.NotBefore) && !now.After(cert.NotAfter)
}

// IsValidRootCA returns true if the provided certificate is a valid root
// CA certificate. A valid root CA is self-signed (the raw subject and
// issuer are identical) and has appropriate basic constraints as well
// as either an empty AuthorityKeyId or one that matches the
// SubjectKeyId. If the certificate has a key usage extension then it
// must include certificate signing. It must also be between its
// NotBefore and NotAfter dates.
//
// The signature of the certificate is not checked, as root certificates
// created by earlier versions of this package are not validly
// self-signed.
func IsValidRootCA(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}
	return cert.IsCA &&
		cert.BasicConstraintsValid &&
		bytes.Equal(cert.RawSubject, cert.RawIssuer) &&
		(cert.KeyUsage == 0 || cert.KeyUsage&x509.KeyUsageCertSign != 0) &&
		(cert.AuthorityKeyId == nil || slices.Equal(cert.AuthorityKeyId, cert.SubjectKeyId)) &&
		validAt(cert, time.Now())
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
		validAt(cert, time.Now())
}

// IsValidLeafCert returns true if the provided certificate is a valid
// leaf certificate. A valid leaf certificate is not a CA, is usable
// for TLS (it either has no key usage extension or one that includes
// DigitalSignature, KeyEncipherment or KeyAgreement) and is between its
// NotBefore and NotAfter dates. Extended key usages are not checked, as
// a certificate without any is valid for all purposes.
func IsValidLeafCert(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}
	const tlsUsages = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageKeyAgreement
	return !cert.IsCA &&
		(cert.KeyUsage == 0 || cert.KeyUsage&tlsUsages != 0) &&
		validAt(cert, time.Now())
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
	Key       PrivateKey
	Valid     bool

	// RootIsSystem is true if the root certificate is from the system
	// pool. We still add certs to the ca-bundle as the core Geneos
	// components can only use one chain file.
	RootIsSystem bool
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

	if len(certs) == 0 {
		return
	}

	for _, c := range certs {
		switch {
		case IsValidLeafCert(c):
			if leaf != nil && !leaf.Equal(c) {
				return false
			}
			leaf = c
		case IsValidRootCA(c):
			roots = append(roots, c)
		case IsValidSigningCA(c):
			intermediates = append(intermediates, c)
		default:
			return false
		}
	}

	if leaf == nil {
		if len(intermediates) > 0 {
			leaf = intermediates[0]
			intermediates = intermediates[1:]
		} else if len(roots) > 0 {
			return true
		} else {
			return false
		}
	}

	_, err := leaf.Verify(verifyOptions(roots, intermediates))
	return err == nil
}

// verifyOptions returns options to verify a chain using the given roots
// and intermediates. If there are no roots then the system roots are
// used. Any extended key usage is accepted as the purpose of the
// certificates is not known here.
func verifyOptions(roots, intermediates []*x509.Certificate) x509.VerifyOptions {
	opts := x509.VerifyOptions{
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}
	if len(roots) > 0 {
		opts.Roots = x509.NewCertPool()
		for _, rc := range roots {
			opts.Roots.AddCert(rc)
		}
	}
	if len(intermediates) > 0 {
		opts.Intermediates = x509.NewCertPool()
		for _, ic := range intermediates {
			opts.Intermediates.AddCert(ic)
		}
	}
	return opts
}

// ParseCertChain tries to verify the provided certificates into a
// chain. It returns the leaf certificate, any intermediates and any
// root certificate found in the chain. It returns an error if
// verification fails. The first certificate in the certs provided is
// assumed to be the leaf certificate. The order of the remaining
// certificates does not matter.
func ParseCertChain(cert ...*x509.Certificate) (leaf *x509.Certificate, intermediates []*x509.Certificate, root *x509.Certificate, err error) {
	var roots []*x509.Certificate

	for _, c := range cert {
		switch {
		case c == nil:
			err = errors.New("nil certificate")
			return
		case IsValidLeafCert(c):
			if leaf != nil && !leaf.Equal(c) {
				err = errors.New("multiple leaf certificates found")
				return
			}
			leaf = c
		case IsValidRootCA(c):
			roots = append(roots, c)
		case IsValidSigningCA(c):
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

	chains, err := leaf.Verify(verifyOptions(roots, intermediates))
	if err != nil {
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
// private keys found. Private keys are always returned as PKCS#8,
// converted from PKCS#1 or SEC1 if necessary. Encrypted private keys
// and other types of blocks are not supported and will be skipped, as
// will any certificates that are expired, not yet valid or that are
// neither a valid root, signing CA or leaf certificate. If there are
// no certificates or private keys found then empty values are returned
// without error. An error is returned if a certificate or an
// unencrypted private key block cannot be parsed.
func DecodePEM(data ...[]byte) (leaf *x509.Certificate, intermediates, roots []*x509.Certificate, keys []PrivateKey, err error) {
	leaf, intermediates, roots, keys, _, err = decodePEM(data...)
	return
}

// decodePEM is DecodePEM that also returns a description of each of
// the certificates and private keys that were skipped.
func decodePEM(data ...[]byte) (leaf *x509.Certificate, intermediates, roots []*x509.Certificate, keys []PrivateKey, skipped []string, err error) {
	var block *pem.Block
	now := time.Now()

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
				switch {
				case IsValidRootCA(c):
					roots = append(roots, c)
				case IsValidSigningCA(c):
					intermediates = append(intermediates, c)
				case IsValidLeafCert(c):
					// save first leaf, ignore others
					if leaf == nil {
						leaf = c
					}
				case now.After(c.NotAfter):
					skipped = append(skipped, fmt.Sprintf("certificate %q is expired", c.Subject.CommonName))
				case now.Before(c.NotBefore):
					skipped = append(skipped, fmt.Sprintf("certificate %q is not yet valid", c.Subject.CommonName))
				default:
					skipped = append(skipped, fmt.Sprintf("certificate %q is not a valid root, signing or leaf certificate", c.Subject.CommonName))
				}
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				if isEncryptedPEM(block) {
					skipped = append(skipped, "encrypted private key")
					continue
				}
				var key PrivateKey
				if key, err = toPKCS8(block.Bytes); err != nil {
					err = fmt.Errorf("invalid %s block: %w", block.Type, err)
					return
				}
				keys = append(keys, key)
			case "ENCRYPTED PRIVATE KEY":
				skipped = append(skipped, "encrypted private key")
			default:
				// unknown PEM blocks are skipped
			}
		}
	}

	return
}

// isEncryptedPEM returns true if the block is an encrypted private key,
// either PKCS#8 or using the legacy encrypted PEM headers.
func isEncryptedPEM(block *pem.Block) bool {
	return block.Type == "ENCRYPTED PRIVATE KEY" || strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED")
}

// ParsePEM parses PEM formatted data blocks and returns a
// CertificateBundle. If it contains a verifiable chain and a private
// key that matches the leaf certificate it then it sets Valid to true.
// Only if Valid is true can the order of the certificates be relied
// upon.
//
// Encrypted private keys are not supported and will be skipped. If
// there are no usable certificates then the error describes any
// certificates that were skipped, for example because they have
// expired.
func ParsePEM(data ...[]byte) (bundle *CertificateBundle, err error) {
	var roots []*x509.Certificate
	var chains [][]*x509.Certificate
	var keys []PrivateKey
	var skipped []string

	if len(data) == 0 {
		err = fmt.Errorf("no data to process")
		return
	}

	bundle = &CertificateBundle{}

	bundle.Leaf, bundle.FullChain, roots, keys, skipped, err = decodePEM(data...)
	if err != nil {
		return
	}
	if len(roots) > 0 {
		bundle.Root = roots[0]
	}

	if bundle.Leaf == nil && len(bundle.FullChain) == 0 && len(roots) == 0 {
		if len(skipped) > 0 {
			err = fmt.Errorf("no valid certificates found (%s)", strings.Join(skipped, "; "))
		} else {
			err = fmt.Errorf("no certificates found")
		}
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
	rp := systemCertPool()
	for _, c := range roots {
		rp.AddCert(c)
	}

	verifyOpts := x509.VerifyOptions{
		Intermediates: ip,
		Roots:         rp,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	if chains, err = bundle.Leaf.Verify(verifyOpts); err != nil || len(chains) == 0 || len(chains[0]) == 0 || bundle.Key == nil {
		// return an unverified bundle
		err = nil
		return
	}

	bundle.Root = chains[0][len(chains[0])-1]
	bundle.Valid = true
	bundle.Leaf = chains[0][0]
	// FullChain does not include root
	bundle.FullChain = append([]*x509.Certificate{}, chains[0][0:len(chains[0])-1]...)

	if bundle.Root != nil {
		verifyOpts := x509.VerifyOptions{
			Roots:     systemCertPool(),
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		}
		if chain, err := bundle.Root.Verify(verifyOpts); err == nil {
			for _, c := range chain {
				if len(c) == 0 {
					continue
				}
				if c[0].Equal(bundle.Root) {
					bundle.RootIsSystem = true
					break
				}
			}
		}
	}

	return
}

// systemCertPool returns a copy of the system certificate pool or an
// empty pool if there is none or it cannot be loaded.
func systemCertPool() *x509.CertPool {
	if pool, err := x509.SystemCertPool(); err == nil && pool != nil {
		return pool
	}
	return x509.NewCertPool()
}

// P12ToRootCerts reads a PKCS#12 (PFX) file and returns the valid root
// CA certificates it contains. It accepts both a normal bundle (a
// private key with its certificate chain) and a certificate-only trust
// store. others is the number of certificates that are not valid root
// CAs (including expired ones) and keys is the number of private keys
// found, neither of which are returned. Certificate-only files must be
// PKCS#12 trust stores in the form written by Java keytool, as other
// certificate-only files are not supported by the PKCS#12 library.
func P12ToRootCerts(pfxPath string, password config.Secret) (roots []*x509.Certificate, others, keys int, err error) {
	pfxData, err := os.ReadFile(config.ResolveHome(pfxPath))
	if err != nil {
		err = fmt.Errorf("failed to read PFX file: %w", err)
		return
	}

	var all []*x509.Certificate
	_, c, caCerts, chainErr := pkcs12.DecodeChain(pfxData, string(password))
	if chainErr == nil {
		keys = 1
		all = append([]*x509.Certificate{c}, caCerts...)
	} else if errors.Is(chainErr, pkcs12.ErrIncorrectPassword) {
		err = fmt.Errorf("failed to decode PFX file: %w", chainErr)
		return
	} else {
		var storeErr error
		all, storeErr = pkcs12.DecodeTrustStore(pfxData, string(password))
		if storeErr != nil {
			err = fmt.Errorf("failed to decode PFX file as a key and certificate bundle (%v) or as a trust store (%v)", chainErr, storeErr)
			return
		}
	}

	for _, cert := range all {
		if IsValidRootCA(cert) {
			roots = append(roots, cert)
		} else {
			others++
		}
	}
	return
}

// P12ToCertBundle converts a PKCS#12 (PFX) file to a CertificateBundle.
// It reads the PFX file, extracts the private key and certificate
// chain, and constructs a CertificateBundle with the leaf certificate,
// full chain, root certificate, and private key. The password must be
// supplied and there is no default.
func P12ToCertBundle(pfxPath string, password config.Secret) (certBundle *CertificateBundle, err error) {
	pfxPath = config.ResolveHome(pfxPath)
	pfxData, err := os.ReadFile(pfxPath)
	if err != nil {
		err = fmt.Errorf("failed to read PFX file: %w", err)
		return
	}
	key, c, caCerts, err := pkcs12.DecodeChain(pfxData, string(password))
	if err != nil {
		err = fmt.Errorf("failed to decode PFX file: %w", err)
		return
	}
	pk, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		err = fmt.Errorf("failed to marshal private key: %w", err)
		return
	}

	leaf, intermediates, root, err := ParseCertChain(append([]*x509.Certificate{c}, caCerts...)...)
	if err != nil {
		err = fmt.Errorf("failed to decompose certificate chain: %w", err)
		return
	}
	if leaf == nil || pk == nil || !CheckKeyMatch(PrivateKey(pk), leaf) {
		err = fmt.Errorf("no leaf certificate and/or matching key found in instance bundle")
		return
	}
	certBundle = &CertificateBundle{
		Leaf:      leaf,
		Key:       PrivateKey(pk),
		FullChain: append([]*x509.Certificate{leaf}, intermediates...),
		Root:      root,
		Valid:     true,
	}
	return
}
