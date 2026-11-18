package certs

import (
	"bytes"
	"crypto/sha1"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/pavlo-v-chernykh/keystore-go/v4"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// AddRootsToTrustStore adds the given root certificates to the truststore
// at the specified path on the given host. If the truststore does not
// exist, it is created. If password is nil, "changeit" is used.
//
// Aliases are derived from the Subject Common Name of each certificate.
// If a different certificate already has the same alias then the
// certificate serial number, or failing that the SHA-1 fingerprint, is
// added to the alias so that existing entries are not replaced. A
// certificate that is already in the truststore is not added again.
// Certificates without a Common Name use the SHA-1 fingerprint as the
// alias.
//
// Any certificates that are not root CAs are ignored. If the truststore
// exists but cannot be read, for example because the password is wrong,
// an error is returned and the file is not changed.
func AddRootsToTrustStore(h host.Host, path string, password config.Secret, roots ...*x509.Certificate) error {
	k, err := readOrNewKeystore(h, path, password)
	if err != nil {
		return err
	}

	for _, cert := range roots {
		if !IsValidRootCA(cert) {
			continue
		}
		if err = k.addRoot(cert); err != nil {
			return err
		}
	}

	// write the updated truststore back to the host with appropriate
	// permissions
	if err = k.WriteKeystore(h, path, password, 0644); err != nil {
		return err
	}

	return nil
}

// WriteTrustStore writes the given root certificates to a keystore at
// the specified path on the given host. If password is nil, "changeit"
// is used. Any certificates that are not valid root CAs are ignored.
// Any existing file is overwritten.
func WriteTrustStore(h host.Host, path string, password config.Secret, roots ...*x509.Certificate) error {
	k := &KeyStore{
		keystore.New(),
	}

	for _, cert := range roots {
		if !IsValidRootCA(cert) {
			continue
		}
		if err := k.addRoot(cert); err != nil {
			return err
		}
	}

	// a truststore is just a keystore with trusted certs, permissions
	// are less restrictive then keystores that contain private keys
	return k.WriteKeystore(h, path, password, 0644)
}

// UpdateCACertsFileFromTrustStore updates the CA bundle file at
// caBundlePath with any root CA certificates found in the truststore at
// truststorePath. If password is nil, "changeit" is used. It returns
// updated true if the CA bundle file was updated.
func UpdateCACertsFileFromTrustStore(h host.Host, truststorePath string, truststorePassword config.Secret, caBundlePath string) (updated bool, err error) {
	var roots []*x509.Certificate

	if truststorePath == "" {
		return false, fmt.Errorf("truststore path empty")
	}

	k, err := ReadKeystore(h, truststorePath, truststorePassword)
	if err != nil {
		return
	}

	for _, alias := range k.Aliases() {
		if c, err := k.GetTrustedCertificateEntry(alias); err == nil {
			cert, err := x509.ParseCertificate(c.Certificate.Content)
			if err == nil && IsValidRootCA(cert) {
				roots = append(roots, cert)
			}
		}
	}

	return UpdateCACertsFiles(h, caBundlePath, roots...)
}

// addRoot adds the root certificate to the truststore using the Subject
// Common Name as the alias, unless that is empty or already used by a
// different certificate, in which case a unique alias is chosen.
func (k *KeyStore) addRoot(cert *x509.Certificate) error {
	cn := cert.Subject.CommonName
	fingerprint := fmt.Sprintf("%X", sha1.Sum(cert.Raw))

	candidates := []string{fingerprint}
	if cn != "" {
		candidates = []string{cn, fmt.Sprintf("%s (%X)", cn, cert.SerialNumber.Bytes()), fingerprint}
	}

	for _, alias := range candidates {
		if !k.IsPrivateKeyEntry(alias) && !k.IsTrustedCertificateEntry(alias) {
			return k.AddTrustedCertificate(alias, cert)
		}
		if e, err := k.GetTrustedCertificateEntry(alias); err == nil && bytes.Equal(e.Certificate.Content, cert.Raw) {
			// already present
			return nil
		}
	}
	return fmt.Errorf("cannot find a unique truststore alias for certificate %q", cn)
}

// AddTrustedCertificate adds a trusted certificate to the keystore.
func (k *KeyStore) AddTrustedCertificate(alias string, cert *x509.Certificate) (err error) {
	if k == nil || alias == "" || cert == nil {
		return os.ErrInvalid
	}

	c := keystore.Certificate{
		Type:    "X509",
		Content: cert.Raw,
	}
	k.DeleteEntry(alias)
	k.SetTrustedCertificateEntry(alias,
		keystore.TrustedCertificateEntry{
			CreationTime: cert.NotBefore,
			Certificate:  c,
		},
	)
	return
}
