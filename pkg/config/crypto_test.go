package config

import (
	"crypto/aes"
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
