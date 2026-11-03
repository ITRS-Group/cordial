package certs

import (
	"errors"
	"os"
	"testing"
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

			_, parsedType, err := ParsePrivateKey(privateKey)
			if err != nil {
				t.Fatalf("ParsePrivateKey() error = %v", err)
			}
			if keyType != ECDH && parsedType != keyType {
				t.Fatalf("ParsePrivateKey() type = %q, want %q", parsedType, keyType)
			}
			// TODO: define whether P-256 ECDH keys should retain the ECDH type after PKCS#8 parsing.
			if (keyType == ECDH) != (publicKey == nil) {
				t.Fatalf("GenerateKey() public key nil = %t for %q", publicKey == nil, keyType)
			}
		})
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
