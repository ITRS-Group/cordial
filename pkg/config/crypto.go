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
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// Secret is a type that represents a plaintext string, but a byteslice
// can be clear()-ed after use, whereas a string cannot.
type Secret []byte

// String returns the secret as a string. This is only to satisfy the
// pflags VerP() interface.
func (secret *Secret) String() string {
	return string(*secret)
}

// Set is required to satisfy the pflag Values interface
func (secret *Secret) Set(value string) error {
	*secret = Secret(value)
	return nil
}

// Type is required to satisfy the pflag Values interface
func (secret *Secret) Type() string {
	return "SECRET"
}

// KeyValues contains the values required to create a Geneos Gateway AES key
// file and then to encode and decode AES passwords in configurations.
type KeyValues struct {
	key [aes.BlockSize * 2]byte
	iv  [aes.BlockSize]byte
}

// NewKeyValues returns a new KeyValues structure with a key and iv
// generated using the crypto/rand package.
func NewKeyValues() (kv *KeyValues) {
	kv = &KeyValues{}
	rand.Read(kv.key[:])
	rand.Read(kv.iv[:])
	return
}

// String method for KeyValues
//
// The output is in the format for suitable for use as a gateway key
// file for secure passwords as described in:
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm
func (kv *KeyValues) String() string {
	// leading space intentional to match native OpenSSL output
	return fmt.Sprintf("key=%X\niv =%X\n", kv.key, kv.iv)
}

// Write writes the KeyValues structure to the io.Writer as an OpenSSL
// formatted file.
func (kv *KeyValues) Write(w io.Writer) error {
	_, err := fmt.Fprintf(w, "key=%X\niv =%X\n", kv.key, kv.iv)
	return err
}

// Destroy zeroes the key and iv values in the KeyValues structure. This
// should be called after use to clear the sensitive data from memory.
// Note that this does not guarantee that the data is cleared from
// memory, as the garbage collector may have made copies of the data,
// but it is a best effort to reduce the risk of sensitive data being
// left in memory. After calling Destroy, the KeyValues structure should
// not be used again.
func (kv *KeyValues) Destroy() {
	clear(kv.key[:])
	clear(kv.iv[:])
}

// ReadKeyValues from the io.Reader r
func ReadKeyValues(r io.Reader) (kv *KeyValues, err error) {
	var gotkey, gotiv bool
	kv = &KeyValues{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			err = fmt.Errorf("invalid line (must be key=value) %q", line)
			return
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)

		switch key {
		case "salt":
			// ignore
		case "key":
			key, err := hex.DecodeString(value)
			if err != nil {
				return nil, err
			}
			if len(key) != len(kv.key) {
				err = errors.New("invalid key length")
			}
			copy(kv.key[:], key)
			gotkey = true
		case "iv":
			iv, err := hex.DecodeString(value)
			if err != nil {
				return nil, err
			}
			if len(iv) != len(kv.iv) {
				err = errors.New("invalid iv length")
			}
			copy(kv.iv[:], iv)
			gotiv = true
		default:
			err = fmt.Errorf("unknown entry in file: %q", key)
			return
		}
	}

	if !gotkey || !gotiv {
		return nil, fmt.Errorf("invalid keyfile contents")
	}
	return
}

// Checksum returns the CRC32 checksum of the KeyValues
func (kv *KeyValues) Checksum() (c uint32, err error) {
	if kv == nil {
		err = os.ErrInvalid
		return
	}
	c = crc32.ChecksumIEEE([]byte(kv.String()))
	return
}

// Checksum returns the CRC32 checksum of the KeyValues
func (kv *KeyValues) ChecksumString() (c string, err error) {
	if kv == nil {
		err = os.ErrInvalid
		return
	}
	c = fmt.Sprintf("%08X", crc32.ChecksumIEEE([]byte(kv.String())))
	return
}

func (kv *KeyValues) encode(secret Secret) (out []byte, err error) {
	block, err := aes.NewCipher(kv.key[:])
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
		return
	}

	// always pad at least one byte (the length)
	var pad []byte
	padBytes := aes.BlockSize - len(secret)%aes.BlockSize
	if padBytes == 0 {
		padBytes = aes.BlockSize
	}
	pad = bytes.Repeat([]byte{byte(padBytes)}, padBytes)
	mode := cipher.NewCBCEncrypter(block, kv.iv[:])
	out = make([]byte, len(secret)+len(pad))
	mode.CryptBlocks(out, append(secret, pad...))
	return
}

// Encode the plaintext using kv, return a byte slice
func (kv *KeyValues) Encode(secret Secret) (out []byte, err error) {
	cipher, err := kv.encode(secret)
	if err == nil {
		out = make([]byte, len(cipher)*2)
		hex.Encode(out, cipher)
		out = bytes.ToUpper(out)
	}
	return
}

// EncodeString encodes the plaintext string using kv, return as a string
func (kv *KeyValues) EncodeString(plaintext string) (out string, err error) {
	text := Secret(plaintext)
	cipher, err := kv.encode(text)
	if err == nil {
		out = strings.ToUpper(hex.EncodeToString(cipher))
	}
	return
}

// Decode returns the decoded value of in bytes using the KeyValues
// given as the method receiver. Any prefix of "+encs+" is trimmed
// before decode. If decoding fails then out is returned empty and err
// will contain the reason.
func (kv *KeyValues) Decode(in []byte) (out []byte, err error) {
	in = bytes.TrimPrefix(in, []byte("+encs+"))

	text := make([]byte, hex.DecodedLen(len(in)))
	hex.Decode(text, in)
	block, err := aes.NewCipher(kv.key[:])
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
		return
	}
	if len(text)%aes.BlockSize != 0 {
		err = fmt.Errorf("input is not a multiple of the block size")
		return
	}
	mode := cipher.NewCBCDecrypter(block, kv.iv[:])
	mode.CryptBlocks(text, text)

	if len(text) == 0 {
		err = fmt.Errorf("decode failed")
		return
	}

	// remove padding as per RFC5246
	paddingLength := int(text[len(text)-1])
	if paddingLength == 0 || paddingLength > aes.BlockSize {
		err = fmt.Errorf("invalid padding size")
		return
	}
	text = text[0 : len(text)-paddingLength]
	if !utf8.Valid(text) {
		err = fmt.Errorf("decoded test not valid UTF-8")
		return
	}
	out = text
	return
}

// DecodeString returns plaintext of the input or an error
func (kv *KeyValues) DecodeString(in string) (out string, err error) {
	plain, err := kv.Decode([]byte(in))
	if err == nil {
		out = string(plain)
	}
	return
}

// Checksum reads from [io.Reader] data until EOF (or other error) and
// returns crc as the 32-bit IEEE checksum. data should be closed by the
// caller on return. If there is an error reading from r then err is
// returned with the reason.
func Checksum(data io.Reader) (crc uint32, err error) {
	b := bytes.Buffer{}
	if _, err = b.ReadFrom(data); err != nil {
		return
	}
	crc = crc32.ChecksumIEEE(b.Bytes())
	return
}
