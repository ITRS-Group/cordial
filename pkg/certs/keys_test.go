package certs

import (
	"bytes"
	"crypto"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/host"
)

func TestKeyTypeSet(t *testing.T) {
	var keyType KeyType
	if err := keyType.Set("ED25519"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if keyType != ED25519 {
		t.Fatalf("Set() = %q, want %q", keyType, ED25519)
	}
	if err := keyType.Set("unknown"); err == nil {
		t.Fatal("Set() succeeded for unsupported key type")
	}
}

func TestGenerateAndParsePrivateKey(t *testing.T) {
	for _, keyType := range []KeyType{ECDSA, ED25519, ECDH} {
		t.Run(string(keyType), func(t *testing.T) {
			privateKey, publicKey, err := GenerateKey(keyType)
			if err != nil {
				t.Fatalf("GenerateKey() error = %v", err)
			}
			if len(privateKey) == 0 {
				t.Fatal("GenerateKey() returned an empty private key")
			}
			if publicKey == nil {
				t.Fatalf("GenerateKey() returned a nil public key for %q", keyType)
			}

			// ECDH keys are generated as ECDSA P-256 keys so that they
			// can sign certificates
			want := keyType
			if keyType == ECDH {
				want = ECDSA
			}
			_, parsedType, err := ParsePrivateKey(privateKey)
			if err != nil {
				t.Fatalf("ParsePrivateKey() error = %v", err)
			}
			if parsedType != want {
				t.Fatalf("ParsePrivateKey() type = %q, want %q", parsedType, want)
			}
			if !publicMatches(t, privateKey, publicKey) {
				t.Fatal("public key does not match the private key")
			}
		})
	}
}

// publicMatches reports whether the public key derived from key
// equals pub.
func publicMatches(t *testing.T, key PrivateKey, pub any) bool {
	t.Helper()
	derived, err := PublicKey(key)
	if err != nil {
		t.Fatalf("PublicKey() error = %v", err)
	}
	return derived.(interface{ Equal(crypto.PublicKey) bool }).Equal(pub)
}

func TestX25519KeyCannotSign(t *testing.T) {
	xk, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(xk)
	if err != nil {
		t.Fatal(err)
	}

	_, keyType, err := ParsePrivateKey(der)
	if err != nil || keyType != ECDH {
		t.Fatalf("ParsePrivateKey() = %q, %v, want %q", keyType, err, ECDH)
	}
	pub, err := PublicKey(der)
	if err != nil || pub == nil {
		t.Fatalf("PublicKey() = %v, %v, want a public key", pub, err)
	}

	// the key cannot sign, so this must be an explicit error and not a
	// nil public key or a panic
	root, _ := createTestRoot(t)
	_, _, err = CreateCertificate(Template("x"), root, der)
	if err == nil || !strings.Contains(err.Error(), "cannot sign") {
		t.Fatalf("CreateCertificate() with X25519 signing key error = %v, want a 'cannot sign' error", err)
	}
}

func TestParsePrivateKeyRejectsInvalidInput(t *testing.T) {
	for name, input := range map[string]PrivateKey{
		"nil":       nil,
		"malformed": []byte("not a DER private key"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := ParsePrivateKey(input); err == nil {
				t.Fatal("ParsePrivateKey() succeeded for invalid input")
			}
		})
	}

	if _, _, err := GenerateKey(KeyType("invalid")); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("GenerateKey() error = %v, want os.ErrInvalid", err)
	}
}

func TestParsePEMRejectsMissingCertificates(t *testing.T) {
	for name, input := range map[string][]byte{
		"no input":      nil,
		"non-PEM input": []byte("not a PEM document"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParsePEM(input); err == nil {
				t.Fatal("ParsePEM() succeeded without certificates")
			}
		})
	}
}

func TestKeyTypeChains(t *testing.T) {
	if testing.Short() {
		t.Skip("generating RSA keys is slow")
	}

	for name, keyType := range map[string]KeyType{
		"rsa": RSA, "ecdsa": ECDSA, "ed25519": ED25519, "ecdh": ECDH, "default": DefaultKeyType,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()

			root, err := WriteNewRootCert(filepath.Join(dir, "root"), "test root", keyType)
			if err != nil {
				t.Fatalf("WriteNewRootCert() error = %v", err)
			}
			rootKey, err := ReadPrivateKey(host.Localhost, filepath.Join(dir, "root"+KEYExtension))
			if err != nil {
				t.Fatalf("ReadPrivateKey(root) error = %v", err)
			}
			if err := root.CheckSignatureFrom(root); err != nil {
				t.Errorf("self-signed root does not verify: %v", err)
			}
			if !CheckKeyMatch(rootKey, root) {
				t.Error("root key does not match root certificate")
			}

			signing, err := WriteNewSigningCert(filepath.Join(dir, "signing"), root, rootKey, "test signing")
			if err != nil {
				t.Fatalf("WriteNewSigningCert() error = %v", err)
			}
			signingKey, err := ReadPrivateKey(host.Localhost, filepath.Join(dir, "signing"+KEYExtension))
			if err != nil {
				t.Fatalf("ReadPrivateKey(signing) error = %v", err)
			}
			if err := signing.CheckSignatureFrom(root); err != nil {
				t.Errorf("signing cert does not verify against root: %v", err)
			}

			// the leaf key type is chosen explicitly and by inheritance
			// from the signing key
			for name, opts := range map[string][]CreateCertOption{
				"inherited": nil,
				"explicit":  {WithKeyType(keyType)},
			} {
				leaf, leafKey, err := CreateCertificate(Template("leaf.example", DNSNames("leaf.example")), signing, signingKey, opts...)
				if err != nil {
					t.Errorf("CreateCertificate(%s) error = %v", name, err)
					continue
				}
				if err := leaf.CheckSignatureFrom(signing); err != nil {
					t.Errorf("%s leaf does not verify against signing cert: %v", name, err)
				}
				if !CheckKeyMatch(leafKey, leaf) {
					t.Errorf("%s leaf key does not match leaf certificate", name)
				}
				if !Verify(leaf, signing, root) {
					t.Errorf("%s: Verify() rejected the chain", name)
				}
				if _, _, _, err := ParseCertChain(leaf, signing, root); err != nil {
					t.Errorf("%s: ParseCertChain() error = %v", name, err)
				}

				keyPath := filepath.Join(dir, name+KEYExtension)
				if err := WritePrivateKey(host.Localhost, keyPath, leafKey); err != nil {
					t.Fatalf("WritePrivateKey() error = %v", err)
				}
				rt, err := ReadPrivateKey(host.Localhost, keyPath)
				if err != nil {
					t.Fatalf("ReadPrivateKey() error = %v", err)
				}
				if !bytes.Equal(rt, leafKey) {
					t.Errorf("%s: private key does not round-trip", name)
				}
			}
		})
	}
}
