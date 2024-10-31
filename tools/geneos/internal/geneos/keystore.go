package geneos

import (
	"crypto/x509"
	"time"

	"github.com/awnumar/memguard"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/pavlo-v-chernykh/keystore-go/v4"
)

// functions to manage Java keystore files for webserver, collection
// agent and others

type KeyStore struct {
	keystore.KeyStore
}

// OpenKeystore returns a keystore
func ReadKeystore(h *Host, path string, password *config.Plaintext) (k KeyStore, err error) {
	r, err := h.Open(path)
	if err != nil {
		return
	}
	k = KeyStore{
		keystore.New(),
	}

	pw := password.Bytes()
	defer memguard.WipeBytes(pw)
	return k, k.Load(r, pw)
}

func (k *KeyStore) WriteKeystore(h *Host, path string, password *config.Plaintext) (err error) {
	if k == nil {
		return ErrInvalidArgs
	}
	w, err := h.Create(path, 0644)
	if err != nil {
		return
	}
	defer w.Close()

	pw := password.Bytes()
	defer memguard.WipeBytes(pw)
	return k.Store(w, pw)
}

func (k *KeyStore) AddKeystoreCert(alias string, cert *x509.Certificate) (err error) {
	k.DeleteEntry(alias)
	c := keystore.Certificate{
		Type:    "X509",
		Content: cert.Raw,
	}
	k.SetTrustedCertificateEntry(alias, keystore.TrustedCertificateEntry{CreationTime: time.Now(), Certificate: c})
	return
}

func (k *KeyStore) AddKeystoreKey(alias string, key *memguard.Enclave, password *config.Plaintext, chain []*x509.Certificate) (err error) {
	k.DeleteEntry(alias)
	l, err := key.Open()
	if err != nil {
		return
	}
	var ch []keystore.Certificate
	for _, c := range chain {
		ch = append(ch, keystore.Certificate{
			Type:    "X509",
			Content: c.Raw,
		})
	}
	c := keystore.PrivateKeyEntry{
		CreationTime:     time.Now(),
		PrivateKey:       l.Bytes(),
		CertificateChain: ch,
	}
	pw := password.Bytes()
	defer memguard.WipeBytes(pw)
	return k.SetPrivateKeyEntry(alias, c, pw)
}
