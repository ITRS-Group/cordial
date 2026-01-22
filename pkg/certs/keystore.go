package certs

import (
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/awnumar/memguard"
	"github.com/pavlo-v-chernykh/keystore-go/v4"
	"github.com/rs/zerolog/log"
	"github.com/square/certigo/jceks"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// functions to manage Java keystore files for webserver, collection
// agent and others

type KeyStore struct {
	keystore.KeyStore
}

// AddCertChainToKeyStore adds a private key and certificate chain to
// the keystore at the specified path on the given host. If the keystore
// does not exist, it is created. If password is nil, "changeit" is used.
//
// The alias parameter specifies the alias under which to store the key
// and certificate chain.
//
// Any existing entry with the same alias is deleted before adding the
// new key and certificate chain.
//
// The certChain is not validated; the caller is responsible for ensuring
// that it is correct.
func AddCertChainToKeyStore(h host.Host, path string, password *config.Plaintext, alias string, key *memguard.Enclave, certChain ...*x509.Certificate) error {
	if password == nil {
		password = config.NewPlaintext([]byte("changeit"))
	}
	k, err := ReadKeystore(h, path, password)
	if err != nil {
		// new, empty keystore
		k = &KeyStore{
			KeyStore: keystore.New(),
		}
	}
	k.DeleteEntry(alias)
	k.AddKeystoreKey(alias, key, password, certChain...)
	return k.WriteKeystore(h, path, password)
}

// AddRootsToTrustStore adds the given root certificates to the truststore
// at the specified path on the given host. If the truststore does not
// exist, it is created. If password is nil, "changeit" is used.
//
// Any existing entries with the same alias as a root certificate are
// deleted before adding the new root certificates. Aliases are derived
// from the Subject Common Name of each certificate.
//
// Any certificates that are not root CAs are ignored.
func AddRootsToTrustStore(h host.Host, path string, password *config.Plaintext, roots ...*x509.Certificate) error {
	k, err := ReadKeystore(h,
		path,
		password,
	)
	if err != nil {
		log.Debug().Err(err).Msg("")
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

	if err = k.WriteKeystore(h, path, password); err != nil {
		return err
	}

	return nil
}

// WriteTrustStore writes the given root certificates to a keystore at
// the specified path on the given host. If password is nil, "changeit"
// is used. Any certificates that are not valid root CAs are ignored.
// Any existing file is overwritten.
func WriteTrustStore(h host.Host, path string, password *config.Plaintext, roots ...*x509.Certificate) error {
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
		password = config.NewPlaintext([]byte("changeit"))
	}

	// a truststore is just a keystore with trusted certs
	return k.WriteKeystore(h, path, password)
}

// ReadKeystore returns a keystore.
//
// If password is nil, "changeit" is used.
//
// The file is first attempted to be read as a modern keystore, if that
// fails it is tried as a JCEKS formatted keystore.
func ReadKeystore(h host.Host, path string, password *config.Plaintext) (k *KeyStore, err error) {
	var pw []byte
	r, err := h.Open(path)
	if err != nil {
		return
	}
	defer r.Close()

	k = &KeyStore{
		keystore.New(),
	}

	if password.IsNil() {
		pw = []byte("changeit")
	} else {
		pw = password.Bytes()
		defer memguard.WipeBytes(pw)
	}
	if err = k.Load(r, pw); err != nil && !errors.Is(err, os.ErrNotExist) {
		// If file exists but cannot be read as a modern keystore, try
		// to convert a JCEKS formatted keystore
		return readJCEKS(h, path, password)
	}

	return
}

// UpdateCACertsFileFromTrustStore updates the CA bundle file at
// caBundlePath with any root CA certificates found in the truststore at
// truststorePath. If password is nil, "changeit" is used. It returns
// updated true if the CA bundle file was updated.
func UpdateCACertsFileFromTrustStore(h host.Host, truststorePath string, truststorePassword *config.Plaintext, caBundlePath string) (updated bool, err error) {
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

// WriteKeystore writes the keystore to the given path. If password is
// nil, "changeit" is used.
func (k *KeyStore) WriteKeystore(h host.Host, path string, password *config.Plaintext) (err error) {
	if k == nil {
		return os.ErrInvalid
	}

	pw := []byte("changeit")
	if !password.IsNil() {
		pw = password.Bytes()
		defer memguard.WipeBytes(pw)
	}

	w, err := h.Create(path, 0644)
	if err != nil {
		return
	}
	defer w.Close()

	return k.Store(w, pw)
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
	k.SetTrustedCertificateEntry(alias, keystore.TrustedCertificateEntry{CreationTime: time.Now(), Certificate: c})
	return
}

// AddKeystoreKey adds a private key and certificates to the keystore.
//
// If password is nil it uses "changeit" as the keystore password.
func (k *KeyStore) AddKeystoreKey(alias string, key *memguard.Enclave, password *config.Plaintext, certs ...*x509.Certificate) (err error) {
	var pw []byte
	var ch []keystore.Certificate

	if k == nil || alias == "" || key == nil || len(certs) == 0 {
		return os.ErrInvalid
	}

	for _, cert := range certs {
		ch = append(ch, keystore.Certificate{
			Type:    "X509",
			Content: cert.Raw,
		})
	}

	l, err := key.Open()
	if err != nil {
		return
	}
	defer l.Destroy()

	c := keystore.PrivateKeyEntry{
		CreationTime:     time.Now(),
		PrivateKey:       l.Bytes(),
		CertificateChain: ch,
	}

	if password.IsNil() {
		pw = []byte("changeit")
	} else {
		pw = password.Bytes()
		defer memguard.WipeBytes(pw)
	}

	k.DeleteEntry(alias)
	return k.SetPrivateKeyEntry(alias, c, pw)
}

// readJCEKS attempts to read a JCEKS formatted keystore and convert it
// to a standard keystore.KeyStore. If password is nil, "changeit" is used.
func readJCEKS(h host.Host, path string, password *config.Plaintext) (k *KeyStore, err error) {
	var pw []byte
	r, err := h.Open(path)
	if err != nil {
		return
	}
	defer r.Close()

	if password.IsNil() {
		pw = []byte("changeit")
	} else {
		pw = password.Bytes()
		defer memguard.WipeBytes(pw)
	}

	jk, err := jceks.LoadFromReader(r, pw)
	if err != nil {
		return
	}
	k = &KeyStore{
		keystore.New(),
	}

	for _, p := range jk.ListPrivateKeys() {
		key, certs, err := jk.GetPrivateKeyAndCerts(p, pw)
		if err != nil {
			panic(err)
		}
		pkcs8key, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			panic(err)
		}

		if err = k.AddKeystoreKey(p, memguard.NewEnclave(pkcs8key), password, certs...); err != nil {
			panic(err)
		}
	}

	for _, c := range jk.ListCerts() {
		cert, err := jk.GetCert(c)
		if err != nil {
			panic(err)
		}
		if err = k.AddTrustedCertificate(c, cert); err != nil {
			panic(err)
		}
	}

	return
}
