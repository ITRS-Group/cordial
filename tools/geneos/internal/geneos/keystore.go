package geneos

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/pavlo-v-chernykh/keystore-go/v4"
)

// functions to manage Java keystore files for webserver, collection
// agent and others

type KeyStore struct {
	keystore.KeyStore
}

// OpenKeystore returns a keystore
func ReadKeystore(path string, password *config.Plaintext) (k KeyStore, err error) {
	r, err := os.Open(path)
	if err != nil {
		return
	}
	k = KeyStore{
		keystore.New(),
	}

	err = k.Load(r, password.Bytes())
	return
}

func (k *KeyStore) WriteKeystore(path string, password *config.Plaintext) (err error) {
	if k == nil {
		return ErrInvalidArgs
	}
	w, err := os.Create(path)
	if err != nil {
		return
	}
	defer w.Close()
	return k.Store(w, password.Bytes())
}

func (k *KeyStore) AddCertKeystore(alias string, cert *x509.Certificate) (err error) {
	k.DeleteEntry(alias)
	c := keystore.Certificate{
		Type: "X509",
		Content: pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}),
	}
	k.SetTrustedCertificateEntry(alias, keystore.TrustedCertificateEntry{CreationTime: time.Now(), Certificate: c})
	return
}
