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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/awnumar/memguard"
)

// KeyFile is a type that represents the path to a keyfile
type KeyFile string

// String returns the path to the keyfile as a string
func (k *KeyFile) String() string {
	return string(*k)
}

// Set is required to satisfy the pflag Values interface
func (k *KeyFile) Set(value string) error {
	*k = KeyFile(value)
	return nil
}

// Type is required to satisfy the pflag Values interface
func (k *KeyFile) Type() string {
	return "KEYFILE"
}

// RollKeyfile will create a new keyfile at path. It will backup any
// existing file with the suffix backup unless the argument is an empty
// string, in which case any existing file is overwritten and no backup
// made.
func (k *KeyFile) RollKeyfile(backup string) (crc uint32, err error) {
	if _, _, err = k.Check(false); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// if is doesn't exist just try to create it
			crc, _, err = k.Check(true)
		}
		return
	}

	if backup != "" {
		if err = os.Rename(k.String(), k.Concat(backup)); err != nil {
			err = fmt.Errorf("keyfile backup failed: %w", err)
			return
		}
	}
	crc, _, err = k.Check(true)
	return
}

// Concat returns a path made up of the path to the keyfile concatenated
// with extras. No separators are added. Typical use is to construct a
// backup file path for an existing keyfile.
func (k *KeyFile) Concat(extras ...string) string {
	elems := append([]string{k.String()}, extras...)
	return strings.Join(elems, "")
}

// Base returns the last component of the file path to keyfile
func (k *KeyFile) Base() string {
	return path.Base(k.String())
}

// Dir returns the path to the directory containing keyfile
func (k *KeyFile) Dir() string {
	return path.Dir(k.String())
}

// Read returns an KeyValues struct populated with the contents of the
// file passed as path. If the keyfile is not in a valid format and err
// is returned.
func (k *KeyFile) Read() (kv *KeyValues, err error) {
	r, err := os.Open(k.String())
	if err != nil {
		return
	}
	defer r.Close()
	kv = Read(r)
	return
}

func (k *KeyFile) Write(kv KeyValues) (err error) {
	w, err := os.Create(k.String())
	if err != nil {
		return
	}
	defer w.Close()

	s := kv.String()
	if s != "" {
		if _, err := fmt.Fprint(w, kv); err != nil {
			return err
		}
	}

	return nil
}

// Check will return the CRC32 checksum of the keyfile at path. If the
// file does not exist and create is true then a new keyfile will be
// created along with any intermediate directories and the checksum of
// the new file will be returned. On error the checksum is undefined and
// err will be set appropriately. If create is true then directories and
// a file may have been created even on error.
func (k *KeyFile) Check(create bool) (crc32 uint32, created bool, err error) {
	if kv, err := k.Read(); err == nil { // ok?
		crc32, err = kv.Checksum()
		return crc32, false, err
	}

	// only try to create if the file error is a not exists
	if _, err = os.Stat(k.String()); err != nil && errors.Is(err, fs.ErrNotExist) {
		if !create {
			return
		}
		if err = os.MkdirAll(k.Dir(), 0775); err != nil {
			err = fmt.Errorf("failed to create keyfile directory %q: %w", k.Dir(), err)
			return
		}
		kv := NewRandomKeyValues()
		if err = os.WriteFile(k.String(), []byte(kv.String()), 0600); err != nil {
			err = fmt.Errorf("failed to write keyfile to %q: %w", k, err)
			return
		}
		created = true

		crc32, err = kv.Checksum()
		if err != nil {
			return
		}
	}
	return
}

// EncodeString encodes the plaintext using the keyfile. The encoded
// password is returned in `Geneos AES256` format, with the `+encs+`
// prefix, unless expandable is set to true in which case it is returned
// in a format that can be used with the Expand function and includes a
// reference to the keyfile.
//
// If the keyfile is located under the user's configuration directory,
// as defined by UserConfigDir, then the function will replace any home
// directory prefix with `~/' to shorten the keyfile path.
func (k *KeyFile) EncodeString(plaintext string, expandable bool) (out string, err error) {
	a, err := k.Read()
	if err != nil {
		return "", err
	}

	e, err := a.EncodeString(plaintext)
	if err != nil {
		return "", err
	}

	if expandable {
		home, _ := UserHomeDir()
		cfdir, _ := UserConfigDir()
		if strings.HasPrefix(k.String(), cfdir) {
			*k = KeyFile("~" + strings.TrimPrefix(k.String(), home))
		}
		out = fmt.Sprintf("${enc:%s:+encs+%s}", k, e)
	} else {
		out = fmt.Sprintf("+encs+%s", e)
	}
	return
}

// Encode encodes the plaintext using the keyfile. The encoded password
// is returned in `Geneos AES256` format, with the `+encs+` prefix,
// unless expandable is set to true in which case it is returned in a
// format that can be used with the Expand function and includes a
// reference to the keyfile.
//
// If the keyfile is located under the user's configuration directory,
// as defined by UserConfigDir, then the function will replace any home
// directory prefix with `~/' to shorten the keyfile path.
func (k *KeyFile) Encode(plaintext *Plaintext, expandable bool) (out string, err error) {
	a, err := k.Read()
	if err != nil {
		return
	}

	e, err := a.Encode(plaintext)
	if err != nil {
		return
	}

	if expandable {
		home, _ := UserHomeDir()
		cfdir, _ := UserConfigDir()
		if strings.HasPrefix(k.String(), cfdir) {
			*k = KeyFile("~" + strings.TrimPrefix(k.String(), home))
		}
		out = fmt.Sprintf("${enc:%s:+encs+%s}", k, e)
	} else {
		out = fmt.Sprintf("+encs+%s", e)
	}
	return
}

// DecodeString decodes the input as a string using keyfile and return
// plaintext. An error is returned if the keyfile is not readable.
func (k *KeyFile) DecodeString(input string) (plaintext string, err error) {
	a, err := k.Read()
	if err != nil {
		return
	}
	return a.DecodeString(input)
}

// Decode input as a byte slice using keyfile and return byte slice
// plaintext. An error is returned if the keyfile is not readable.
func (k *KeyFile) Decode(input []byte) (plaintext []byte, err error) {
	a, err := k.Read()
	if err != nil {
		return
	}
	return a.Decode(input)
}

// DecodeEnclave decodes the input using the keyfile k and returns a
// memguard.Enclave
func (k *KeyFile) DecodeEnclave(input []byte) (plaintext *memguard.Enclave, err error) {
	a, err := k.Read()
	if err != nil {
		return
	}
	return a.DecodeEnclave(input)
}

// EncodePasswordInput prompts the user for a password and again to
// verify, offering up to three attempts until the password match. When
// the two match the plaintext is encoded using the keyfile. If
// expandable is true then the encoded password is returned in a format
// useable by the Expand function which includes a path to the keyfile
// used at the time.
func (k *KeyFile) EncodePasswordInput(expandable bool) (out string, err error) {
	plaintext, err := ReadPasswordInput(true, 3)
	if err != nil {
		return
	}
	out, err = k.Encode(plaintext, expandable)
	return
}
