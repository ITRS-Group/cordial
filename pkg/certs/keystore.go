package certs

import (
	"crypto/x509"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	"github.com/pavlo-v-chernykh/keystore-go/v4"
	"github.com/square/certigo/jceks"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// functions to manage Java keystore files for webserver, collection
// agent and others

// KeyStore TODO: Describe its purpose and intended use.
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
// If the keystore exists but cannot be read, for example because the
// password is wrong, an error is returned and the file is not changed.
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
	k, err := readOrNewKeystore(h, path, password)
	if err != nil {
		return err
	}
	if err = k.AddKeystoreKey(alias, key, password, certChain...); err != nil {
		return fmt.Errorf("adding private key %q to keystore: %w", alias, err)
	}
	return k.WriteKeystore(h, path, password, 0600)
}

// readOrNewKeystore reads the keystore at path or returns a new, empty
// keystore if, and only if, the file does not exist or is empty. Any other error,
// such as a wrong password or a corrupt file, is returned so that the
// existing file is never overwritten.
func readOrNewKeystore(h host.Host, path string, password config.Secret) (k *KeyStore, err error) {
	k, err = ReadKeystore(h, path, password)
	if err == nil {
		return
	}
	if errors.Is(err, os.ErrNotExist) {
		return NewKeyStore(), nil
	}
	// an empty file has nothing to lose
	if fi, serr := h.Stat(path); serr == nil && fi.Mode().IsRegular() && fi.Size() == 0 {
		return NewKeyStore(), nil
	}
	return nil, err
}

// ReadKeystore returns a keystore.
//
// If password is nil, "changeit" is used.
//
// The file is first attempted to be read as a modern keystore, if that
// fails it is tried as a JCEKS formatted keystore. If the file does not
// exist then the returned error satisfies errors.Is(err, os.ErrNotExist).
func ReadKeystore(h host.Host, path string, password config.Secret) (k *KeyStore, err error) {
	r, err := h.Open(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	pw := defaultKeystorePassword(password)

	k = NewKeyStore()
	if err = k.Load(r, pw); err != nil {
		// If file exists but cannot be read as a modern keystore, try
		// to convert a JCEKS formatted keystore
		jk, jerr := readJCEKS(h, path, password)
		if jerr != nil {
			return nil, fmt.Errorf("cannot read keystore %q: %w (as JCEKS: %v)", path, err, jerr)
		}
		return jk, nil
	}

	return
}

// defaultKeystorePassword returns the password as bytes, or "changeit"
// if the password is empty.
func defaultKeystorePassword(password config.Secret) []byte {
	if len(password) == 0 {
		return []byte("changeit")
	}
	return password
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
//
// The keystore is written to a temporary file in the same directory
// which is then renamed over path, so that an existing file is never
// left truncated or partially written.
func (k *KeyStore) WriteKeystore(h host.Host, path string, password config.Secret, perms os.FileMode) (err error) {
	if k == nil || perms == 0 {
		return os.ErrInvalid
	}

	pw := defaultKeystorePassword(password)

	tmp := path + ".tmp" + strconv.FormatUint(rand.Uint64(), 36)
	w, err := h.Create(tmp, perms)
	if err != nil {
		return
	}

	err = k.Store(w, pw)
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = h.Rename(tmp, path)
	}
	if err != nil {
		_ = h.Remove(tmp)
	}
	return
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

	k.DeleteEntry(alias)
	return k.SetPrivateKeyEntry(alias, c, defaultKeystorePassword(password))
}

// ChangePassword changes the password of the keystore and all private
// keys in the keystore at the given path. It writes a new keystore file
// with the updated password, keeping the existing file permissions, and
// atomically replaces the old one. If either password is empty then
// "changeit" is used. If the keystore cannot be read or any private key
// cannot be decrypted with oldPassword then an error is returned and
// the existing file is not changed.
func ChangePassword(h host.Host, ks string, oldPassword, newPassword config.Secret) (err error) {
	if h == nil || ks == "" {
		return os.ErrInvalid
	}

	oldPassword = config.Secret(defaultKeystorePassword(oldPassword))
	newPassword = config.Secret(defaultKeystorePassword(newPassword))

	k, err := ReadKeystore(h, ks, oldPassword)
	if err != nil {
		return
	}

	// update all private keys
	for _, alias := range k.Aliases() {
		if !k.IsPrivateKeyEntry(alias) {
			continue
		}
		entry, err := k.GetPrivateKeyEntry(alias, oldPassword)
		if err != nil {
			return fmt.Errorf("cannot read private key %q with the old password: %w", alias, err)
		}
		if err = k.SetPrivateKeyEntry(alias, entry, newPassword); err != nil {
			return fmt.Errorf("cannot set password for private key %q: %w", alias, err)
		}
	}

	perms := os.FileMode(0600)
	if fi, err := h.Stat(ks); err == nil && fi.Mode().Perm() != 0 {
		perms = fi.Mode().Perm()
	}

	return k.WriteKeystore(h, ks, newPassword, perms)
}

// readJCEKS attempts to read a JCEKS formatted keystore and convert it
// to a standard keystore.KeyStore. If password is nil, "changeit" is used.
func readJCEKS(h host.Host, path string, password config.Secret) (k *KeyStore, err error) {
	r, err := h.Open(path)
	if err != nil {
		return
	}
	defer r.Close()

	pw := defaultKeystorePassword(password)

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
			return nil, fmt.Errorf("read private key %q: %w", p, err)
		}
		pkcs8key, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("marshal private key %q: %w", p, err)
		}

		if err = k.AddKeystoreKey(p, PrivateKey(pkcs8key), password, certs...); err != nil {
			return nil, fmt.Errorf("add private key %q: %w", p, err)
		}
	}

	for _, c := range jk.ListCerts() {
		cert, err := jk.GetCert(c)
		if err != nil {
			return nil, fmt.Errorf("read certificate %q: %w", c, err)
		}
		if err = k.AddTrustedCertificate(c, cert); err != nil {
			return nil, fmt.Errorf("add trusted certificate %q: %w", c, err)
		}
	}

	return
}
