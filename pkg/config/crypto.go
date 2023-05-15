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
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sort"
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

// NewRandomKeyValues returns a new KeyValues structure with a key and iv
// generated using the memguard.
func NewRandomKeyValues() (kv *KeyValues) {
	var k *keyvalues
	kv = &KeyValues{
		memguard.NewEnclaveRandom(int(unsafe.Sizeof(*k))),
	}
	return
}

func NewKeyValues() (kv *KeyValues) {
	var k *keyvalues
	m := memguard.NewBuffer(int(unsafe.Sizeof(*k)))
	kv = &KeyValues{
		m.Seal(),
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
	if _, err := fmt.Fprint(w, kv.String()); err != nil {
		return err
	}

	return nil
}

// Read KeyValues from the io.Reader r and return a locked buffer keyvalues kv. m should be
// destroyed after use.
func Read(r io.Reader) (kv *KeyValues) {
	var k *keyvalues
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
			// err = fmt.Errorf("invalid line (must be key=value) %q", line)
			return
		}
		key, value := strings.TrimSpace(s[0]), strings.TrimSpace(s[1])
		switch key {
		case "salt":
			// ignore
		case "key":
			key, _ := hex.DecodeString(value)
			copy(k.key[:], key)
		case "iv":
			iv, _ := hex.DecodeString(value)
			copy(k.iv[:], iv)
		default:
			// err = fmt.Errorf("unknown entry in file: %q", key)
			return
		}
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

func (kv *KeyValues) encode(plaintext *memguard.Enclave) (out []byte, err error) {
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

func (kv *KeyValues) Encode(in *memguard.Enclave) (out []byte, err error) {
	cipher, err := kv.encode(in)
	if err == nil {
		out = make([]byte, len(cipher)*2)
		hex.Encode(out, cipher)
		out = bytes.ToUpper(out)
	}
	return
}

func (kv *KeyValues) EncodeString(in string) (out string, err error) {
	text := memguard.NewEnclave([]byte(in))
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

// Credentials handling function

// Credentials can carry a number of different credential types. Add
// more as required. Eventually this will go into memguard.
type Credentials struct {
	Domain       string `json:"domain,omitempty"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Token        string `json:"token,omitempty"`
	Renewal      string `json:"renewal,omitempty"`
}

// FindCreds finds a set of credentials in the given config structure
// under the key "credentials" and tries to match the longest one, if
// any. creds is nil if not matching credentials found.
func (cf *Config) FindCreds(path string) (creds *Config) {
	cr := cf.GetStringMap("credentials")
	if cr == nil {
		return
	}
	paths := []string{}
	for k := range cr {
		paths = append(paths, k)
	}
	// sort the paths longest to shortest
	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i]) > len(paths[j])
	})
	creds = New()
	for _, p := range paths {
		if strings.Contains(path, p) {
			creds.MergeConfigMap(cf.GetStringMap(cf.Join("credentials", p)))
			return
		}
	}
	return nil
}

// FindCreds looks for matching credentials in a default "credentials"
// file. options are the same as for [Load] but the default KeyDelimiter
// is set to "::" as credential domains are likely to be hostnames or
// URLs. The longest match wins.
func FindCreds(path string, options ...FileOptions) (cred *Config) {
	options = append(options, KeyDelimiter("::"))
	cf, _ := Load("credentials", options...)
	return cf.FindCreds(path)
}

// Add creds to the "credentials" file identified by the options.
// creds.Domain is used as the key for matching later on. Any existing
// credential with the same Domain is overwritten. If there is an error
// un the underlying routines it is returned without change.
func AddCreds(creds Credentials, options ...FileOptions) (err error) {
	options = append(options, KeyDelimiter("::"))
	cf, err := Load("credentials", options...)
	if err != nil {
		return
	}
	cf.Set(cf.Join("credentials", creds.Domain), creds)
	return cf.Save("credentials", options...)
}

// DeleteCreds removes the entry for domain from the "credentials" file
// using FileOptions options.
func DeleteCreds(domain string, options ...FileOptions) (err error) {
	options = append(options, KeyDelimiter("::"))
	cf, err := Load("credentials", options...)
	if err != nil {
		return
	}
	credmap := cf.GetStringMap("credentials")
	delete(credmap, domain)
	cf.Set("credentials", credmap)
	return cf.Save("credentials", options...)
}

func DeleteAllCreds(options ...FileOptions) (err error) {
	options = append(options, KeyDelimiter("::"))
	cf := New(options...)
	cf.Set("credentials", &Credentials{})
	return cf.Save("credentials", options...)
}
