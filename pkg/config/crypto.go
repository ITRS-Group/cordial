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
	"io"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// An AESValues structure contains the values required to create a
// Geneos Gateway AES key files and also the encode and decode AES
// passwords in the configuration
type AESValues struct {
	Key []byte
	IV  []byte
}

// create a gateway key file for secure passwords as per
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm

// NewAESValues returns a new AESValues structure or an error
func NewAESValues() (aes AESValues, err error) {
	rp := make([]byte, 20)
	salt := make([]byte, 10)
	if _, err = rand.Read(rp); err != nil {
		return
	}
	if _, err = rand.Read(salt); err != nil {
		return
	}

	md := pbkdf2.Key(rp, salt, 10000, 48, sha1.New)
	aes.Key = md[:32]
	aes.IV = md[32:]

	return
}

// WriteAESValues writes the AESValues structure to the io.Writer. Each
// fields acts as if it were being marshalled with an ",omitempty" tag.
func (aes AESValues) WriteAESValues(w io.Writer) (err error) {
	if len(aes.Key) > 0 {
		_, err = fmt.Fprintf(w, "key=%X\n", aes.Key)
		if err != nil {
			return
		}
	}
	if len(aes.IV) > 0 {
		// space intentional to match native output
		_, err = fmt.Fprintf(w, "iv =%X\n", aes.IV)
		if err != nil {
			return
		}
	}

	return
}

// ReadAESValues consumes the io.Reader passed and extracts the salt, key and IV
func ReadAESValues(r io.Reader) (aes AESValues, err error) {
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
			// aes.Salt, _ = hex.DecodeString(value)
			// ignore
		case "key":
			aes.Key, _ = hex.DecodeString(value)
		case "iv":
			aes.IV, _ = hex.DecodeString(value)
		default:
			err = fmt.Errorf("unknown entry in file: %q", key)
			return
		}
	}
	return
}

func (a AESValues) EncodeAES(in []byte) (out []byte, err error) {
	block, err := aes.NewCipher(a.Key)
	if err != nil {
		err = fmt.Errorf("invalid key: %w", err)
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
	mode := cipher.NewCBCDecrypter(block, a.IV)
	mode.CryptBlocks(text, text)

	// remove padding as per RFC5246
	paddingLength := int(text[len(text)-1])
	text = text[0 : len(text)-paddingLength]
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
