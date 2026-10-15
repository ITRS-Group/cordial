package certs

import (
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
// Any existing entries with the same alias as a root certificate are
// deleted before adding the new root certificates. Aliases are derived
// from the Subject Common Name of each certificate.
//
// Any certificates that are not root CAs are ignored.
func AddRootsToTrustStore(h host.Host, path string, password config.Secret, roots ...*x509.Certificate) error {
	k, err := ReadKeystore(h,
		path,
		password,
	)
	if err != nil {
		k = &KeyStore{
			KeyStore: keystore.New(),
		}
	}

	for _, cert := range roots {
		if !IsValidRootCA(cert) {
			continue
		}
		alias := cert.Subject.CommonName
		k.DeleteEntry(alias)
		if err = k.AddTrustedCertificate(alias, cert); err != nil {
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
		alias := cert.Subject.CommonName
		k.DeleteEntry(alias)
		if err := k.AddTrustedCertificate(alias, cert); err != nil {
			return err
		}
	}

	if password == nil {
		password = config.Secret("changeit")
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
