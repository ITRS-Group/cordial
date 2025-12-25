package certs

import (
	"crypto/x509"
	"errors"
	"os"
	"time"

	"github.com/awnumar/memguard"
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

// AddKeystoreCertificate adds a certificate to the keystore.
func (k *KeyStore) AddKeystoreCertificate(alias string, cert *x509.Certificate) (err error) {
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

	for _, c := range certs {
		ch = append(ch, keystore.Certificate{
			Type:    "X509",
			Content: c.Raw,
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
		if err = k.AddKeystoreCertificate(c, cert); err != nil {
			panic(err)
		}
	}

	return
}
