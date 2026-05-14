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
)

type KeyType string
type PrivateKey []byte

const (
	RSA     KeyType = "rsa"
	ECDSA   KeyType = "ecdsa"
	ED25519 KeyType = "ed25519"
	ECDH    KeyType = "ecdh"
)

// DefaultKeyType is the default key type
const DefaultKeyType = ECDH

func (k *KeyType) String() string {
	return string(*k)
}

func (k *KeyType) Set(s string) error {
	kt := strings.ToLower(s)
	switch kt {
	case "rsa", "ecdsa", "ed25519", "ecdh":
		*k = KeyType(kt)
		return nil
	default:
		return fmt.Errorf("invalid key type %q, must be one of rsa, ecdsa, ed25519 or ecdh", s)
	}
}

func (k *KeyType) Type() string {
	return "KeyType"
}

// GenerateKey returns a PKCS#8 DER encoded private key as an enclave
// using the keyType specified.
func GenerateKey(keyType KeyType) (privateKey PrivateKey, publicKey any, err error) {
	var pKey any
	switch keyType {
	case RSA:
		pKey, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return
		}
	case ECDSA:
		pKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return
		}
	case ED25519:
		_, pKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			return
		}
	case ECDH:
		var ecdsaKey *ecdsa.PrivateKey
		ecdsaKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return
		}
		pKey, err = ecdsaKey.ECDH()
		if err != nil {
			return
		}
	default:
		err = fmt.Errorf("%w unsupported key type %s", os.ErrInvalid, keyType)
		return
	}

	if k, ok := pKey.(crypto.Signer); ok {
		publicKey = k.Public()
	}

	key, err := x509.MarshalPKCS8PrivateKey(pKey)
	if err != nil {
		return
	}
	privateKey = PrivateKey(key)

	return
}

// ParsePrivateKey parses the DER encoded private key enclave, first as
// PKCS#8 and then as a PKCS#1 and finally as SEC1 (EC) if that fails.
// It returns the private key or an error.
func ParsePrivateKey(key PrivateKey) (privatekey any, keytype KeyType, err error) {
	if key == nil {
		err = fmt.Errorf("no key provided")
		return
	}
	if privatekey, err = x509.ParsePKCS8PrivateKey(key); err != nil {
		if privatekey, err = x509.ParsePKCS1PrivateKey(key); err != nil {
			if privatekey, err = x509.ParseECPrivateKey(key); err != nil {
				return
			}
		}
	}

	switch privatekey.(type) {
	case *rsa.PrivateKey:
		keytype = RSA
	case *ecdsa.PrivateKey:
		keytype = ECDSA
	case *ecdh.PrivateKey:
		keytype = ECDH
	case ed25519.PrivateKey: // not a pointer
		keytype = ED25519
	}
	return
}

// PublicKey parses the DER encoded private key enclave and returns the
// public key if successful. It will first try as PKCS#8 and then PKCS#1
// if that fails and finally as SEC1 (EC). Returns an error if parsing
// fails.
func PublicKey(key PrivateKey) (publicKey crypto.PublicKey, err error) {
	if key == nil {
		err = fmt.Errorf("no key provided")
		return
	}
	privateKey, _, err := ParsePrivateKey(key)
	if err != nil {
		return
	}

	if k, ok := privateKey.(crypto.Signer); ok {
		publicKey = k.Public()
	}

	return
}

// IndexPrivateKey tests the slice DER encoded private keys against the x509
// cert and returns the index of the first match, or -1 if none of the
// keys match.
func IndexPrivateKey(keys []PrivateKey, cert *x509.Certificate) int {
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
func CheckKeyMatch(key PrivateKey, cert *x509.Certificate) bool {
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
