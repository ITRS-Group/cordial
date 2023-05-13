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

// KeyValues contains the values required to create a Geneos Gateway AES
// key file and then to encode and decode AES passwords in
// configurations
type KeyValues struct {
	key [aes.BlockSize * 2]byte
	iv  [aes.BlockSize]byte
}

// NewRandomKeyValues returns a new KeyValues structure with a key and iv
// generated using the memguard. The KeyValue should be destoryed after use.
func NewRandomKeyValues() (m *memguard.LockedBuffer, kv *KeyValues) {
	var k *KeyValues
	m = memguard.NewBufferRandom(int(unsafe.Sizeof(*k)))
	kv = (*KeyValues)(unsafe.Pointer(&m.Bytes()[0]))
	return
}

func NewKeyValues() (m *memguard.LockedBuffer, kv *KeyValues) {
	var k *KeyValues
	m = memguard.NewBuffer(int(unsafe.Sizeof(*k)))
	kv = (*KeyValues)(unsafe.Pointer(&m.Bytes()[0]))
	return
}

// String method for KeyValues
//
// The output is in the format for suitable for use as a gateway key
// file for secure passwords as described in:
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm
func (kv *KeyValues) String() string {
	// space intentional to match native OpenSSL output
	return fmt.Sprintf("key=%X\niv =%X\n", kv.key, kv.iv)
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
func Read(r io.Reader) (m *memguard.LockedBuffer, kv *KeyValues) {
	m, kv = NewKeyValues()
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
			k, _ := hex.DecodeString(value)
			m.Move(k)
		case "iv":
			i, _ := hex.DecodeString(value)
			m.Move(i)
		default:
			// err = fmt.Errorf("unknown entry in file: %q", key)
			return
		}
	}
	m.Freeze()
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
	block, err := aes.NewCipher(kv.key[:])
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
	mode := cipher.NewCBCEncrypter(block, kv.iv[:])
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
	if len(kv.iv) != aes.BlockSize {
		err = fmt.Errorf("IV is not the same length as the block size")
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
	Domain       string `json:"domain"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Token        string `json:"token,omitempty"`
	Renewal      string `json:"renewal,omitempty"`
}

// FindCreds finds a set of credentials in the given config structure
// under the key "credentials" and tries to match the longest one, if
// any.
func (cf *Config) FindCreds(match string) (creds *Credentials) {
	creds = &Credentials{}
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
	for _, p := range paths {
		if strings.Contains(p, match) {
			cf.UnmarshalKey("credentials::"+p, &creds)
			return
		}
	}
	return
}

// FindCreds looks for matching credentials in a default "credentials"
// file. options are the same as for [Load] but the default KeyDelimiter
// is set to "::" as credential domains are likely to be hostnames or
// URLs. The longest match wins.
func FindCreds(search string, options ...FileOptions) (cred *Credentials) {
	options = append(options, KeyDelimiter("::"))
	cf, _ := Load("credentials", options...)
	return cf.FindCreds(search)
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
	cf.Set("credentials::"+creds.Domain, creds)
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
	cr := cf.GetStringMap("credentials")
	delete(cr, domain)
	cf.Set("credentials", cr)
	return cf.Save("credentials", options...)
}
