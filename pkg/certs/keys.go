package certs

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/awnumar/memguard"
)

type KeyType string

const (
	RSA     KeyType = "rsa"
	ECDSA   KeyType = "ecdsa"
	ED25519 KeyType = "ed25519"
	ECDH    KeyType = "ecdh"
)

// DefaultKeyType is the default key type
const DefaultKeyType = ECDH

func (k KeyType) String() string {
	return string(k)
}

func (k *KeyType) Set(s string) error {
	switch strings.ToLower(s) {
	case "rsa", "ecdsa", "ed25519", "ecdh":
		*k = KeyType(strings.ToLower(s))
		return nil
	default:
		return fmt.Errorf("invalid key type %q, must be one of rsa, ecdsa, ed25519 or ecdh", s)
	}
}

func (k *KeyType) Type() string {
	return "KeyType"
}

// PrivateKeyType returns the type of the DER encoded private key,
// suitable for use to NewPrivateKey
func PrivateKeyType(der *memguard.Enclave) (keytype KeyType) {
	if der == nil {
		return
	}
	key, err := PrivateKey(der)
	if err != nil {
		return
	}

	switch key.(type) {
	case *rsa.PrivateKey:
		return RSA
	case *ecdsa.PrivateKey:
		return ECDSA
	case *ecdh.PrivateKey:
		return ECDH
	case ed25519.PrivateKey: // not a pointer
		return ED25519
	default:
		return ""
	}
}

// NewPrivateKey returns a PKCS#8 DER encoded private key as an enclave.
func NewPrivateKey(keytype KeyType) (der *memguard.Enclave, publickey any, err error) {
	var privateKey any
	switch keytype {
	case RSA:
		privateKey, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return
		}
	case ECDSA:
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return
		}
	case ED25519:
		_, privateKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			return
		}
	case ECDH:
		var ecdsaKey *ecdsa.PrivateKey
		ecdsaKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return
		}
		privateKey, err = ecdsaKey.ECDH()
		if err != nil {
			return
		}
	default:
		err = fmt.Errorf("%w unsupported key type %s", os.ErrInvalid, keytype)
		return
	}

	key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return
	}
	der = memguard.NewEnclave(key)

	if k, ok := privateKey.(crypto.Signer); ok {
		publickey = k.Public()
	}
	return
}

// PrivateKey parses the DER encoded private key enclave, first as
// PKCS#8 and then as a PKCS#1 and finally as SEC1 (EC) if that fails.
// It returns the private key or an error.
func PrivateKey(key *memguard.Enclave) (privatekey any, err error) {
	k, err := key.Open()
	if err != nil {
		return
	}
	defer k.Destroy()

	der := k.Bytes()
	if privatekey, err = x509.ParsePKCS8PrivateKey(der); err != nil {
		if privatekey, err = x509.ParsePKCS1PrivateKey(der); err != nil {
			if privatekey, err = x509.ParseECPrivateKey(der); err != nil {
				return
			}
		}
	}
	return
}

// PublicKey parses the DER encoded private key enclave and returns the
// public key if successful. It will first try as PKCS#8 and then PKCS#1
// if that fails and finally as SEC1 (EC). Returns an error if parsing
// fails.
func PublicKey(key *memguard.Enclave) (publickey crypto.PublicKey, err error) {
	privatekey, err := PrivateKey(key)
	if err != nil {
		return
	}

	if k, ok := privatekey.(crypto.Signer); ok {
		publickey = k.Public()
	}

	return
}

// IndexPrivateKey tests the slice DER encoded private keys against the x509
// cert and returns the index of the first match, or -1 if none of the
// keys match.
func IndexPrivateKey(keys []*memguard.Enclave, cert *x509.Certificate) int {
	if cert == nil {
		return -1
	}

	for i, key := range keys {
		if pubkey, err := PublicKey(key); err == nil { // if ok then compare
			// ensure we have an Equal() method on the opaque key
			if k, ok := pubkey.(interface{ Equal(crypto.PublicKey) bool }); ok {
				if k.Equal(cert.PublicKey) {
					return i
				}
			}
		}
	}
	return -1
}

// CheckKeyMatch returns true if the DER encoded private key matches
// the public key in the provided x509 certificate.
func CheckKeyMatch(key *memguard.Enclave, cert *x509.Certificate) bool {
	pubkey, err := PublicKey(key)
	if err != nil {
		return false
	}
	// ensure we have an Equal() method on the opaque key
	if k, ok := pubkey.(interface{ Equal(crypto.PublicKey) bool }); ok {
		return k.Equal(cert.PublicKey)
	}
	return false
}
