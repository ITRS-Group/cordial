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
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"
	"unicode/utf8"
	"unsafe"

	"github.com/awnumar/memguard"
)

// KeyValues contains the values required to create a Geneos Gateway AES key
// file and then to encode and decode AES passwords in configurations. It is
// handled as a memguard Enclave to protect the plaintext as much as possible.
type KeyValues struct {
	*memguard.Enclave
}

// keyvalues holds an AES key and IV
type keyvalues struct {
	key [aes.BlockSize * 2]byte
	iv  [aes.BlockSize]byte
}

// Plaintext is a type that represents a plaintext string that should be
// protected
type Plaintext struct {
	*memguard.Enclave
}

// String returns the secret as a string. Use Bytes() where possible and
// memguard.WipeBytes() after use. This method retruns a cloned string
// for those cases there a downstream API requires a credential as a
// string value. In Go you can't securely zero strings after use.
func (secret *Plaintext) String() string {
	if secret == nil || secret.Enclave == nil {
		return ""
	}
	l, _ := secret.Open()
	defer l.Destroy()
	return strings.Clone(l.String())
}

// Bytes returns the secret as a byte slice. After use, call
// memguard.WipeBytes() on the buffer
func (secret *Plaintext) Bytes() []byte {
	if secret == nil || secret.Enclave == nil {
		return nil
	}
	l, _ := secret.Open()
	defer l.Destroy()
	return bytes.Clone(l.Bytes())
}

// Set is required to satisfy the pflag Values interface
func (secret *Plaintext) Set(value string) error {
	if secret != nil {
		*secret = Plaintext{memguard.NewEnclave([]byte(value))}
	}
	return nil
}

// Type is required to satisfy the pflag Values interface
func (secret *Plaintext) Type() string {
	return "PLAINTEXT"
}

// NewPlaintext returns a memguard Enclave initialised with buf
func NewPlaintext(buf []byte) *Plaintext {
	return &Plaintext{memguard.NewEnclave(buf)}
}

// IsNil returns true if the secret or the underlying memguard Enclave
// is nil
func (secret *Plaintext) IsNil() bool {
	if secret == nil {
		return true
	}
	return secret.Enclave == nil
}

// NewRandomKeyValues returns a new KeyValues structure with a key and iv
// generated using the memguard.
func NewRandomKeyValues() (kv *KeyValues) {
	var k *keyvalues
	kv = &KeyValues{
		memguard.NewEnclaveRandom(int(unsafe.Sizeof(*k))),
	}
	return
}

func lockedBufferTo[T any](m *memguard.LockedBuffer) (v *T) {
	return (*T)(unsafe.Pointer(&m.Bytes()[0]))
}

// String method for KeyValues
//
// The output is in the format for suitable for use as a gateway key
// file for secure passwords as described in:
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm
func (kv *KeyValues) String() string {
	kl, _ := kv.Open()
	defer kl.Destroy()
	k := lockedBufferTo[keyvalues](kl)

	// leading space intentional to match native OpenSSL output
	return fmt.Sprintf("key=%X\niv =%X\n", k.key, k.iv)
}

// Write writes the KeyValues structure to the io.Writer.
func (kv *KeyValues) Write(w io.Writer) error {
	if _, err := io.WriteString(w, kv.String()); err != nil {
		return err
	}

	return nil
}

// ReadKeyValues from the io.Reader r and return a locked buffer
// key values kv.
func ReadKeyValues(r io.Reader) (kv *KeyValues, err error) {
	var k *keyvalues
	var gotkey, gotiv bool
	m := memguard.NewBuffer(int(unsafe.Sizeof(*k)))
	k = lockedBufferTo[keyvalues](m)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		s := strings.SplitN(line, "=", 2)
		if len(s) != 2 {
			err = fmt.Errorf("invalid line (must be key=value) %q", line)
			return
		}
		key, value := strings.TrimSpace(s[0]), strings.TrimSpace(s[1])
		switch key {
		case "salt":
			// ignore
		case "key":
			key, err := hex.DecodeString(value)
			if err != nil {
				return nil, err
			}
			copy(k.key[:], key)
			gotkey = true
		case "iv":
			iv, err := hex.DecodeString(value)
			if err != nil {
				return nil, err
			}
			copy(k.iv[:], iv)
			gotiv = true
		default:
			err = fmt.Errorf("unknown entry in file: %q", key)
			return
		}
	}

	if !gotkey || !gotiv {
		return nil, fmt.Errorf("invalid keyfile contents")
	}
	kv = &KeyValues{
		m.Seal(),
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

func (kv *KeyValues) encode(plaintext *Plaintext) (out []byte, err error) {
	kl, _ := kv.Open()
	defer kl.Destroy()
	k := lockedBufferTo[keyvalues](kl)

	block, err := aes.NewCipher(k.key[:])
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
		return
	}

	in, _ := plaintext.Open()

	// always pad at least one byte (the length)
	var pad []byte
	padBytes := aes.BlockSize - in.Size()%aes.BlockSize
	if padBytes == 0 {
		padBytes = aes.BlockSize
	}
	pad = bytes.Repeat([]byte{byte(padBytes)}, padBytes)
	mode := cipher.NewCBCEncrypter(block, k.iv[:])
	out = make([]byte, in.Size()+len(pad))
	mode.CryptBlocks(out, append(in.Bytes(), pad...))
	in.Destroy()
	return
}

// Encode the plaintext using kv, return a byte slice
func (kv *KeyValues) Encode(plaintext *Plaintext) (out []byte, err error) {
	cipher, err := kv.encode(plaintext)
	if err == nil {
		out = make([]byte, len(cipher)*2)
		hex.Encode(out, cipher)
		out = bytes.ToUpper(out)
	}
	return
}

// EncodeString encodes the plaintext string using kv, return as a string
func (kv *KeyValues) EncodeString(plaintext string) (out string, err error) {
	text := NewPlaintext([]byte(plaintext))
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
	kl, _ := kv.Open()
	defer kl.Destroy()
	k := lockedBufferTo[keyvalues](kl)

	in = bytes.TrimPrefix(in, []byte("+encs+"))

	text := make([]byte, hex.DecodedLen(len(in)))
	hex.Decode(text, in)
	block, err := aes.NewCipher(k.key[:])
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
		return
	}
	if len(text)%aes.BlockSize != 0 {
		err = fmt.Errorf("input is not a multiple of the block size")
		return
	}
	mode := cipher.NewCBCDecrypter(block, k.iv[:])
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

// DecodeEnclave decodes the input using kv and returns a *memguard.Enclave
func (kv *KeyValues) DecodeEnclave(in []byte) (out *memguard.Enclave, err error) {
	kl, _ := kv.Open()
	defer kl.Destroy()
	k := lockedBufferTo[keyvalues](kl)

	in = bytes.TrimPrefix(in, []byte("+encs+"))

	ciphertext := make([]byte, hex.DecodedLen(len(in)))

	hex.Decode(ciphertext, in)
	block, err := aes.NewCipher(k.key[:])
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
		return
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		err = fmt.Errorf("input is not a multiple of the block size")
		return
	}
	mode := cipher.NewCBCDecrypter(block, k.iv[:])
	l := memguard.NewBuffer(len(ciphertext))
	plaintext := l.Bytes()
	mode.CryptBlocks(plaintext, ciphertext)

	if len(plaintext) == 0 {
		err = fmt.Errorf("decode failed")
		return
	}

	// remove padding as per RFC5246
	paddingLength := int((plaintext)[len(plaintext)-1])
	if paddingLength == 0 || paddingLength > aes.BlockSize {
		err = fmt.Errorf("invalid padding size")
		return
	}
	plaintext = (plaintext)[0 : len(plaintext)-paddingLength]
	if !utf8.Valid(plaintext) {
		err = fmt.Errorf("decoded test not valid UTF-8")
		return
	}
	out = memguard.NewEnclave(plaintext)
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
