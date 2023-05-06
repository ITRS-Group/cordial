/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package config

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/pbkdf2"
)

// KeyValues contains the values required to create a Geneos Gateway AES
// key file and then to encode and decode AES passwords in
// configurations
type KeyValues struct {
	key []byte
	iv  []byte
}

// NewKeyValues returns a new KeyValues structure with a key and iv
// generated using the crypto/rand package.
func NewKeyValues() (kv KeyValues, err error) {
	rp := make([]byte, 20)
	salt := make([]byte, 10)

	// generate the key and IV separately; this could be done in one
	// call, but it seems better practise to do it in two passes

	if _, err = rand.Read(rp); err != nil {
		return
	}
	if _, err = rand.Read(salt); err != nil {
		return
	}
	kv.key = pbkdf2.Key(rp, salt, 10000, 32, sha1.New)

	if _, err = rand.Read(rp); err != nil {
		return
	}
	if _, err = rand.Read(salt); err != nil {
		return
	}
	kv.iv = pbkdf2.Key(rp, salt, 10000, aes.BlockSize, sha1.New)

	return
}

// String method for KeyValues
//
// The output is in the format for suitable for use as a gateway key
// file for secure passwords as described in:
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm
func (kv *KeyValues) String() string {
	if len(kv.key) != 32 || len(kv.iv) != aes.BlockSize {
		return ""
	}
	// space intentional to match native OpenSSL output
	return fmt.Sprintf("key=%X\niv =%X\n", kv.key, kv.iv)
}

// Write writes the KeyValues structure to the io.Writer.
func (kv *KeyValues) Write(w io.Writer) error {
	if len(kv.key) != 32 || len(kv.iv) != aes.BlockSize {
		return fmt.Errorf("invalid AES values")
	}
	s := kv.String()
	if s != "" {
		if _, err := fmt.Fprint(w, kv); err != nil {
			return err
		}
	}

	return nil
}

// Read returns an KeyValues struct populated with the contents
// read from r. The caller must close the Reader on return.
func Read(r io.Reader) (kv KeyValues, err error) {
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
			kv.key, _ = hex.DecodeString(value)
		case "iv":
			kv.iv, _ = hex.DecodeString(value)
		default:
			err = fmt.Errorf("unknown entry in file: %q", key)
			return
		}
	}
	if len(kv.key) != 32 || len(kv.iv) != aes.BlockSize {
		return KeyValues{}, fmt.Errorf("invalid AES values")
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

func (kv *KeyValues) encode(in []byte) (out []byte, err error) {
	block, err := aes.NewCipher(kv.key)
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
		return
	}
	if len(kv.iv) != aes.BlockSize {
		err = fmt.Errorf("IV is not the same length as the block size")
		return
	}

	// always pad at least one byte (the length)
	var pad []byte
	padBytes := aes.BlockSize - len(in)%aes.BlockSize
	if padBytes == 0 {
		padBytes = aes.BlockSize
	}
	pad = bytes.Repeat([]byte{byte(padBytes)}, padBytes)
	in = append(in, pad...)
	mode := cipher.NewCBCEncrypter(block, kv.iv)
	mode.CryptBlocks(in, in)
	out = in
	return
}

func (kv *KeyValues) Encode(in []byte) (out []byte, err error) {
	text := []byte(in)
	cipher, err := kv.encode(text)
	if err == nil {
		out = make([]byte, len(cipher)*2)
		hex.Encode(out, cipher)
		out = bytes.ToUpper(out)
	}
	return
}

func (kv *KeyValues) EncodeString(in string) (out string, err error) {
	text := []byte(in)
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
	block, err := aes.NewCipher(kv.key)
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
		return
	}
	if len(text)%aes.BlockSize != 0 {
		err = fmt.Errorf("input is not a multiple of the block size")
		return
	}
	if len(kv.iv) != aes.BlockSize {
		err = fmt.Errorf("IV is not the same length as the block size")
		return
	}
	mode := cipher.NewCBCDecrypter(block, kv.iv)
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

// Checksum reads from [io.Reader] data until EOF and returns crc as the
// 32-bit IEEE checksum. data should be closed by the caller on return.
// If there is an error reading from r then err is returned with the
// reason.
func Checksum(data io.Reader) (crc uint32, err error) {
	b := bytes.Buffer{}
	if _, err = b.ReadFrom(data); err != nil {
		return
	}
	crc = crc32.ChecksumIEEE(b.Bytes())
	return
}
