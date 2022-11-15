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

// An AESValues structure contains the values required to create a
// Geneos Gateway AES key file and then to encode and decode AES
// passwords in configurations
type AESValues struct {
	Key []byte
	IV  []byte
}

// NewAESValues returns a new AESValues structure or an error
func NewAESValues() (a AESValues, err error) {
	rp := make([]byte, 20)
	salt := make([]byte, 10)

	// generate the key and IV separately

	if _, err = rand.Read(rp); err != nil {
		return
	}
	if _, err = rand.Read(salt); err != nil {
		return
	}
	a.Key = pbkdf2.Key(rp, salt, 10000, 32, sha1.New)

	if _, err = rand.Read(rp); err != nil {
		return
	}
	if _, err = rand.Read(salt); err != nil {
		return
	}
	a.IV = pbkdf2.Key(rp, salt, 10000, aes.BlockSize, sha1.New)

	return
}

// String method for AESValues
//
// The output is in the format for suitable for use as a gateway key
// file for secure passwords as described in:
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm
func (a AESValues) String() string {
	if len(a.Key) != 32 || len(a.IV) != aes.BlockSize {
		return ""
	}
	// space intentional to match native OpenSSL output
	return fmt.Sprintf("key=%X\niv =%X\n", a.Key, a.IV)
}

// WriteAESValues writes the AESValues structure to the io.Writer. Each
// fields acts as if it were being marshalled with an ",omitempty" tag.
func (a AESValues) WriteAESValues(w io.Writer) error {
	if len(a.Key) != 32 || len(a.IV) != aes.BlockSize {
		return fmt.Errorf("invalid AES values")
	}
	s := a.String()
	if s != "" {
		if _, err := fmt.Fprint(w, a); err != nil {
			return err
		}
	}

	return nil
}

// ReadAESValuesFile returns an AESValues struct populated with the
// contents of the file passed as path.
func ReadAESValuesFile(path string) (a AESValues, err error) {
	r, err := os.Open(path)
	if err != nil {
		return
	}
	defer r.Close()
	return ReadAESValues(r)
}

// ReadAESValues returns an AESValues struct populated with the contents
// read from r. The caller must close the Reader on return.
func ReadAESValues(r io.Reader) (a AESValues, err error) {
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
			a.Key, _ = hex.DecodeString(value)
		case "iv":
			a.IV, _ = hex.DecodeString(value)
		default:
			err = fmt.Errorf("unknown entry in file: %q", key)
			return
		}
	}
	if len(a.Key) != 32 || len(a.IV) != aes.BlockSize {
		return AESValues{}, fmt.Errorf("invalid AES values")
	}
	return
}

func ChecksumFile(path string) (c uint32, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	c = crc32.ChecksumIEEE(b)
	return
}

// Checksum returns the CRC32 checksum of the AESValue it is called on.
func (a *AESValues) Checksum() (c uint32, err error) {
	if a == nil {
		err = os.ErrInvalid
		return
	}
	c = crc32.ChecksumIEEE([]byte(a.String()))
	return
}

func Checksum(r io.Reader) (c uint32, err error) {
	b := bytes.Buffer{}
	_, err = b.ReadFrom(r)
	if err != nil {
		return
	}
	c = crc32.ChecksumIEEE(b.Bytes())
	return
}

func ChecksumString(in string) (c uint32, err error) {
	c = crc32.ChecksumIEEE([]byte(in))
	return
}

func (a AESValues) EncodeAES(in []byte) (out []byte, err error) {
	block, err := aes.NewCipher(a.Key)
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
		return
	}
	if len(a.IV) != aes.BlockSize {
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
	mode := cipher.NewCBCEncrypter(block, a.IV)
	mode.CryptBlocks(in, in)
	out = in
	return
}

func (a AESValues) EncodeAESBytes(in []byte) (out []byte, err error) {
	text := []byte(in)
	cipher, err := a.EncodeAES(text)
	if err == nil {
		out = make([]byte, len(cipher)*2)
		hex.Encode(out, cipher)
		out = bytes.ToUpper(out)
	}
	return
}

func (a AESValues) EncodeAESString(in string) (out string, err error) {
	text := []byte(in)
	cipher, err := a.EncodeAES(text)
	if err == nil {
		out = strings.ToUpper(hex.EncodeToString(cipher))
	}
	return
}

func (a AESValues) DecodeAES(in []byte) (out []byte, err error) {
	text := make([]byte, hex.DecodedLen(len(in)))
	hex.Decode(text, in)
	block, err := aes.NewCipher(a.Key)
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
		return
	}
	if len(text)%aes.BlockSize != 0 {
		err = fmt.Errorf("input is not a multiple of the block size")
		return
	}
	if len(a.IV) != aes.BlockSize {
		err = fmt.Errorf("IV is not the same length as the block size")
		return
	}
	mode := cipher.NewCBCDecrypter(block, a.IV)
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

// DecodeAESString returns a plain text of the input or an error
func (a AESValues) DecodeAESString(in string) (out string, err error) {
	plain, err := a.DecodeAES([]byte(in))
	if err == nil {
		out = string(plain)
	}
	return
}
