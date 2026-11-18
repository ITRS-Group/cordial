package config

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func TestReadKeyValuesValidation(t *testing.T) {
	validKey := strings.Repeat("00", aes.BlockSize*2)
	validIV := strings.Repeat("00", aes.BlockSize)
	tests := []struct {
		name string
		data string
	}{
		{
			name: "short key",
			data: fmt.Sprintf("key=%s\niv=%s\n", strings.Repeat("00", aes.BlockSize*2-1), validIV),
		},
		{
			name: "short IV",
			data: fmt.Sprintf("key=%s\niv=%s\n", validKey, strings.Repeat("00", aes.BlockSize-1)),
		},
		{
			name: "invalid hex",
			data: fmt.Sprintf("key=%s\niv=%s\n", strings.Repeat("zz", aes.BlockSize*2), validIV),
		},
		{
			name: "missing IV",
			data: fmt.Sprintf("key=%s\n", validKey),
		},
	}

	// TODO: confirm whether duplicate key and IV entries should be rejected or use the last value.
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ReadKeyValues(strings.NewReader(test.data)); err == nil {
				t.Fatal("ReadKeyValues() succeeded for invalid keyfile")
			}
		})
	}
}

func TestKeyValuesEncodeDecode(t *testing.T) {
	key := strings.Repeat("01", aes.BlockSize*2)
	iv := strings.Repeat("02", aes.BlockSize)
	values, err := ReadKeyValues(strings.NewReader(fmt.Sprintf("key=%s\niv=%s\n", key, iv)))
	if err != nil {
		t.Fatalf("ReadKeyValues() error = %v", err)
	}

	ciphertext, err := values.EncodeString("sensitive value")
	if err != nil {
		t.Fatalf("EncodeString() error = %v", err)
	}
	for _, input := range []string{ciphertext, "+encs+" + ciphertext} {
		plaintext, err := values.DecodeString(input)
		if err != nil {
			t.Fatalf("DecodeString(%q) error = %v", input, err)
		}
		if plaintext != "sensitive value" {
			t.Fatalf("DecodeString() = %q, want %q", plaintext, "sensitive value")
		}
	}

	if _, err := values.DecodeString("not-hex"); err == nil {
		t.Fatal("DecodeString() succeeded for malformed ciphertext")
	}
}

// encryptBlocks encrypts plaintext, which must be a multiple of the
// block size, using kv without adding any padding
func encryptBlocks(t *testing.T, kv *KeyValues, plaintext []byte) string {
	t.Helper()
	block, err := aes.NewCipher(kv.key[:])
	if err != nil {
		t.Fatal(err)
	}
	out := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, kv.iv[:]).CryptBlocks(out, plaintext)
	return strings.ToUpper(hex.EncodeToString(out))
}

func TestKeyValuesDecodeChecksAllPaddingBytes(t *testing.T) {
	kv := NewKeyValues()
	defer kv.Destroy()

	tests := []struct {
		name      string
		plaintext []byte
		want      string
		wantErr   bool
	}{
		{name: "valid padding", plaintext: append([]byte("0123456789"), bytes.Repeat([]byte{6}, 6)...), want: "0123456789"},
		{name: "full block of padding", plaintext: bytes.Repeat([]byte{16}, 16), want: ""},
		{name: "only the last pad byte is right", plaintext: append([]byte("0123456789ab"), 1, 2, 3, 4), wantErr: true},
		{name: "zero padding", plaintext: append([]byte("0123456789abcde"), 0), wantErr: true},
		{name: "padding longer than a block", plaintext: append([]byte("0123456789abcde"), 17), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := kv.DecodeString(encryptBlocks(t, kv, test.plaintext))
			if (err != nil) != test.wantErr {
				t.Fatalf("DecodeString() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("DecodeString() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestKeyValuesDecodeRejectsInvalidHex(t *testing.T) {
	kv := NewKeyValues()
	defer kv.Destroy()

	good := encryptBlocks(t, kv, bytes.Repeat([]byte{16}, 16))
	for name, input := range map[string]string{
		"bad character": good[:30] + "zz",
		"odd length":    good + "0",
	} {
		if out, err := kv.Decode([]byte(input)); err == nil || out != nil {
			t.Errorf("%s: Decode() = %q, %v; want an error", name, out, err)
		}
	}
}

func TestKeyValuesEncodeDoesNotModifyCallersSecret(t *testing.T) {
	kv := NewKeyValues()
	defer kv.Destroy()

	backing := []byte("secret-and-spare-capacity-here")
	secret := Secret(backing[:6])
	if _, err := kv.Encode(secret); err != nil {
		t.Fatal(err)
	}
	if string(backing) != "secret-and-spare-capacity-here" {
		t.Errorf("Encode() modified memory beyond the secret: %q", backing)
	}
}
