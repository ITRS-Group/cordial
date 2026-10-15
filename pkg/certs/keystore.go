package certs

import (
	"crypto/x509"
	"errors"
	"os"
	"time"

	"github.com/pavlo-v-chernykh/keystore-go/v4"
	"github.com/square/certigo/jceks"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// functions to manage Java keystore files for webserver, collection
// agent and others

type KeyStore struct {
	keystore.KeyStore
}

// NewKeyStore creates and returns a new KeyStore instance.
func NewKeyStore() *KeyStore {
	return &KeyStore{
		keystore.New(),
	}
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
func AddCertChainToKeyStore(h host.Host, path string, password config.Secret, alias string, key PrivateKey, certChain ...*x509.Certificate) error {
	if len(password) == 0 {
		password = config.Secret("changeit")
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
	return k.WriteKeystore(h, path, password, 0600)
}

// ReadKeystore returns a keystore.
//
// If password is nil, "changeit" is used.
//
// The file is first attempted to be read as a modern keystore, if that
// fails it is tried as a JCEKS formatted keystore.
func ReadKeystore(h host.Host, path string, password config.Secret) (k *KeyStore, err error) {
	var pw []byte
	r, err := h.Open(path)
	if err != nil {
		return
	}
	defer r.Close()

	k = &KeyStore{
		keystore.New(),
	}

	if len(password) == 0 {
		pw = []byte("changeit")
	} else {
		pw = password
	}
	if err := k.Load(r, pw); err != nil && !errors.Is(err, os.ErrNotExist) {
		// If file exists but cannot be read as a modern keystore, try
		// to convert a JCEKS formatted keystore
		return readJCEKS(h, path, password)
	}

	return
}

// ParseTrustedCertificate parses a trusted certificate from the
// keystore entry tk.
//
// It returns an x509.Certificate instance or an error if parsing fails.
func ParseTrustedCertificate(tk keystore.TrustedCertificateEntry) (*x509.Certificate, error) {
	return x509.ParseCertificate(tk.Certificate.Content)
}

// ParsePrivateKeyCertificates parses the certificate chain from a
// private key entry in the keystore.
//
// It returns a slice of x509.Certificate instances or an error if
// parsing any certificate fails.
func ParsePrivateKeyCertificates(pk keystore.PrivateKeyEntry) (certs []*x509.Certificate, err error) {
	for _, cert := range pk.CertificateChain {
		parsed, err := x509.ParseCertificate(cert.Content)
		if err != nil {
			return nil, err
		}
		certs = append(certs, parsed)
	}
	return certs, nil
}

// WriteKeystore writes the keystore to the given path with the given
// permissions. If password is nil, "changeit" is used. If perms is 0 an
// error is returned.
func (k *KeyStore) WriteKeystore(h host.Host, path string, password config.Secret, perms os.FileMode) (err error) {
	if k == nil || perms == 0 {
		return os.ErrInvalid
	}

	pw := []byte("changeit")
	if len(password) > 0 {
		pw = password
	}

	w, err := h.Create(path, perms)
	if err != nil {
		return
	}
	defer w.Close()

	return k.Store(w, pw)
}

// WritePrivateKeyEntryAsPEM writes the private key entry as a PEM file
// to the specified path on the host. The file is created with the given
// permissions. Returns an error if writing fails.
func WritePrivateKeyEntryAsPEM(h host.Host, path string, entry keystore.PrivateKeyEntry, perms os.FileMode) error {
	w, err := h.Create(path, perms)
	if err != nil {
		return err
	}
	defer w.Close()

	cs, err := ParsePrivateKeyCertificates(entry)
	if err != nil {
		return err
	}
	_, err = WriteCertificatesAndKeyTo(w, entry.PrivateKey, cs...)
	if err != nil {
		return err
	}
	return nil
}

// AddKeystoreKey adds a private key and certificates to the keystore.
//
// If password is nil it uses "changeit" as the keystore password.
func (k *KeyStore) AddKeystoreKey(alias string, key PrivateKey, password config.Secret, certs ...*x509.Certificate) (err error) {
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

	c := keystore.PrivateKeyEntry{
		CreationTime:     time.Now(),
		PrivateKey:       []byte(key),
		CertificateChain: ch,
	}

	if len(password) == 0 {
		pw = []byte("changeit")
	} else {
		pw = password
	}

	k.DeleteEntry(alias)
	return k.SetPrivateKeyEntry(alias, c, pw)
}

// ChangePassword changes the password of all private keys in the
// keystore at the given path. It creates a new keystore file with the
// updated password and replaces the old one.
func ChangePassword(h host.Host, ks string, oldPassword, newPassword config.Secret) (err error) {
	if h == nil || ks == "" {
		return os.ErrInvalid
	}

	k, err := ReadKeystore(h, ks, oldPassword)
	if err != nil {
		return
	}

	// update all private keys
	for _, alias := range k.Aliases() {
		if entry, err := k.GetPrivateKeyEntry(alias, oldPassword); err == nil {
			if err = k.SetPrivateKeyEntry(alias, entry, newPassword); err != nil {
				return err
			}
		}
	}

	nks := ks + ".new"
	err = k.WriteKeystore(h, nks, newPassword, 0600)
	if err == nil {
		err = h.Rename(nks, ks)
	}
	return
}

// readJCEKS attempts to read a JCEKS formatted keystore and convert it
// to a standard keystore.KeyStore. If password is nil, "changeit" is used.
func readJCEKS(h host.Host, path string, password config.Secret) (k *KeyStore, err error) {
	var pw []byte
	r, err := h.Open(path)
	if err != nil {
		return
	}
	defer r.Close()

	if len(password) == 0 {
		pw = []byte("changeit")
	} else {
		pw = password
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

		if err = k.AddKeystoreKey(p, PrivateKey(pkcs8key), password, certs...); err != nil {
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
