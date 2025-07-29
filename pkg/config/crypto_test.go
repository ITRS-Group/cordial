/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRandomKeyValues(t *testing.T) {
	kv := NewRandomKeyValues()
	if kv == nil {
		t.Fatal("NewRandomKeyValues() returned nil")
	}

	if kv.Enclave == nil {
		t.Error("NewRandomKeyValues() returned KeyValues with nil Enclave")
	}

	// Test that we can get string representation
	s := kv.String()
	if s == "" {
		t.Error("KeyValues should have string representation")
	}
}

func TestNewPlaintext(t *testing.T) {
	testData := []byte("test password")
	plaintext := NewPlaintext(testData)

	if plaintext == nil {
		t.Fatal("NewPlaintext() returned nil")
	}

	if plaintext.IsNil() {
		t.Error("NewPlaintext() returned nil plaintext")
	}

	// Verify content using public methods
	if plaintext.String() != string(testData) {
		t.Error("Plaintext content doesn't match input")
	}

	if !bytes.Equal(plaintext.Bytes(), testData) {
		t.Error("Plaintext bytes don't match input")
	}
}

func TestKeyValuesEncodeDecodeString(t *testing.T) {
	kv := NewRandomKeyValues()

	testString := "Hello, World!"

	// Test encoding
	encoded, err := kv.EncodeString(testString)
	if err != nil {
		t.Fatalf("EncodeString() failed: %v", err)
	}

	if encoded == "" {
		t.Error("EncodeString() returned empty string")
	}

	// Test decoding
	decoded, err := kv.DecodeString("+encs+"+encoded)
	if err != nil {
		t.Fatalf("DecodeString() failed: %v", err)
	}

	if decoded != testString {
		t.Errorf("DecodeString() = %q, want %q", decoded, testString)
	}
}

func TestKeyValuesEncodeDecodeBytes(t *testing.T) {
	kv := NewRandomKeyValues()

	testData := []byte("Binary data test \x00\x01\x02")
	plaintext := NewPlaintext(testData)

	// Test encoding
	encoded, err := kv.Encode(plaintext)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Encode() returned empty slice")
	}

	// Test decoding
	decoded, err := kv.Decode(append([]byte("+encs+"), encoded...))
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if !bytes.Equal(decoded, testData) {
		t.Error("Decoded data doesn't match original")
	}
}

func TestKeyValuesEncodePassword(t *testing.T) {
	kv := NewRandomKeyValues()

	password := NewPlaintext([]byte("secret123"))

	encoded, err := kv.Encode(password)
	if err != nil {
		t.Fatalf("EncodePassword() failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("EncodePassword() returned empty data")
	}

	// Test decoding back
	decoded, err := kv.DecodeString("+encs+"+string(encoded))
	if err != nil {
		t.Fatalf("DecodeString() failed: %v", err)
	}

	if decoded != "secret123" {
		t.Error("Decoded password doesn't match original")
	}
}

func TestReadKeyValues(t *testing.T) {
	// Create a key and get its string representation
	originalKV := NewRandomKeyValues()

	keyData := originalKV.String()

	// Test reading from string reader
	reader := strings.NewReader(keyData)
	readKV, err := ReadKeyValues(reader)
	if err != nil {
		t.Fatalf("ReadKeyValues() failed: %v", err)
	}

	if readKV.Enclave == nil {
		t.Error("ReadKeyValues() returned KeyValues with nil Enclave")
	}

	// Test that both keys can encode/decode the same data
	testData := "test encryption"

	encoded1, err := originalKV.EncodeString(testData)
	if err != nil {
		t.Fatalf("Original key encoding failed: %v", err)
	}

	decoded2, err := readKV.DecodeString("+encs+"+encoded1)
	if err != nil {
		t.Fatalf("Read key decoding failed: %v", err)
	}

	if decoded2 != testData {
		t.Error("Keys don't match - decoding failed")
	}
}

func TestKeyValuesWriteRead(t *testing.T) {
	kv := NewRandomKeyValues()

	// Write to buffer
	var buf bytes.Buffer
	err := kv.Write(&buf)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Write() produced no output")
	}

	// Read back
	readKV, err := ReadKeyValues(&buf)
	if err != nil {
		t.Fatalf("ReadKeyValues() failed: %v", err)
	}

	// Verify they work the same
	testStr := "test round trip"
	encoded, err := kv.EncodeString(testStr)
	if err != nil {
		t.Fatalf("Encoding failed: %v", err)
	}

	decoded, err := readKV.DecodeString("+encs+"+encoded)
	if err != nil {
		t.Fatalf("Decoding with read key failed: %v", err)
	}

	if decoded != testStr {
		t.Error("Round trip encoding/decoding failed")
	}
}

func TestKeyValuesInvalidData(t *testing.T) {
	kv := NewRandomKeyValues()

	// Test decoding invalid data
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"invalid prefix", "invalid+data"},
		{"short data", "+encs+abc"},
		{"invalid hex", "+encs+invalid hex!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := kv.DecodeString(tt.input)
			if err == nil {
				t.Error("DecodeString() should fail with invalid input")
			}
		})
	}
}

func TestChecksum(t *testing.T) {
	testData := "test data for checksum"
	reader := strings.NewReader(testData)

	crc, err := Checksum(reader)
	if err != nil {
		t.Fatalf("Checksum() failed: %v", err)
	}

	if crc == 0 {
		t.Error("Checksum() returned 0, expected non-zero value")
	}

	// Test that same data produces same checksum
	reader2 := strings.NewReader(testData)
	crc2, err := Checksum(reader2)
	if err != nil {
		t.Fatalf("Second Checksum() failed: %v", err)
	}

	if crc != crc2 {
		t.Errorf("Checksums don't match: %d != %d", crc, crc2)
	}

	// Test different data produces different checksum
	reader3 := strings.NewReader("different data")
	crc3, err := Checksum(reader3)
	if err != nil {
		t.Fatalf("Third Checksum() failed: %v", err)
	}

	if crc == crc3 {
		t.Error("Different data should produce different checksum")
	}
}

func TestPlaintextString(t *testing.T) {
	testStr := "test password"
	plaintext := NewPlaintext([]byte(testStr))

	if plaintext.String() != testStr {
		t.Errorf("String() = %q, want %q", plaintext.String(), testStr)
	}
}

func TestPlaintextBytes(t *testing.T) {
	testData := []byte("test data")
	plaintext := NewPlaintext(testData)

	result := plaintext.Bytes()
	if !bytes.Equal(result, testData) {
		t.Error("Bytes() doesn't match original data")
	}
}

func TestPlaintextNil(t *testing.T) {
	plaintext := NewPlaintext([]byte("test"))

	if plaintext.IsNil() {
		t.Error("Plaintext should not be nil initially")
	}

	// Test nil plaintext
	var nilPlaintext *Plaintext = nil
	if !nilPlaintext.IsNil() {
		t.Error("Nil plaintext should return true for IsNil()")
	}
}

func TestKeyValuesWithDifferentData(t *testing.T) {
	kv := NewRandomKeyValues()

	// Test various data types
	testCases := []struct {
		name string
		data string
	}{
		{"simple", "hello"},
		{"unicode", "Hello 世界 🌍"},
		{"special chars", "!@#$%^&*()"},
		{"newlines", "line1\nline2\r\nline3"},
		// Note: binary data with null bytes may not work well with string encoding
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := kv.EncodeString(tc.data)
			if err != nil {
				t.Fatalf("EncodeString() failed: %v", err)
			}

			decoded, err := kv.DecodeString("+encs+"+encoded)
			if err != nil {
				t.Fatalf("DecodeString() failed: %v", err)
			}

			if decoded != tc.data {
				t.Errorf("Round trip failed: got %q, want %q", decoded, tc.data)
			}
		})
	}
}

func TestMultipleKeyValues(t *testing.T) {
	// Test that different KeyValues instances produce different encrypted output
	kv1 := NewRandomKeyValues()
	kv2 := NewRandomKeyValues()

	testData := "same input"

	encoded1, err := kv1.EncodeString(testData)
	if err != nil {
		t.Fatalf("First encoding failed: %v", err)
	}

	encoded2, err := kv2.EncodeString(testData)
	if err != nil {
		t.Fatalf("Second encoding failed: %v", err)
	}

	// Different keys should produce different ciphertext
	if encoded1 == encoded2 {
		t.Error("Different keys should produce different ciphertext")
	}

	// Each key should only be able to decode its own data
	_, err = kv1.DecodeString("+encs+"+encoded2)
	if err == nil {
		t.Error("Key1 should not be able to decode data encrypted with Key2")
	}

	_, err = kv2.DecodeString("+encs+"+encoded1)
	if err == nil {
		t.Error("Key2 should not be able to decode data encrypted with Key1")
	}
}