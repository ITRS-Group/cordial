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
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/awnumar/memguard"
	"github.com/itrs-group/cordial/pkg/host"
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

// CreateWithBackup will create a new keyfile at path. It will rename
// any existing file with backup appended to the filename before the
// extension, unless backup is an empty string, in which case any
// existing file is overwritten and no backup made.
func (k *KeyFile) CreateWithBackup(h host.Host, backup string) (crc uint32, err error) {
	if _, err = k.ReadCRC(h); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// if is doesn't exist just try to create it, no backup
			crc, _, err = k.ReadOrCreate(h)
		}
		return
	}

	if backup != "" {
		kp := string(*k)
		ext := filepath.Ext(kp)
		basename := strings.TrimSuffix(filepath.Base(kp), ext)
		dir := filepath.Dir(kp)
		bkp := filepath.Join(dir, basename+backup+ext)
		if err = h.Rename(kp, bkp); err != nil {
			return 0, err
		}
	}
	crc, _, err = k.ReadOrCreate(h)
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
	return filepath.Base(k.String())
}

// Dir returns the path to the directory containing keyfile
func (k *KeyFile) Dir() string {
	return filepath.Dir(k.String())
}

// Read returns an KeyValues struct populated with the contents of the
// file passed as path. If the keyfile is not in a valid format and err
// is returned.
func (k *KeyFile) Read(h host.Host) (kv *KeyValues, err error) {
	r, err := h.Open(k.String())
	if err != nil {
		return
	}
	defer r.Close()
	return ReadKeyValues(r)
}

func (k *KeyFile) Write(h host.Host, kv *KeyValues) (err error) {
	w, err := h.Create(k.String(), 0600)
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

// ReadOrCreate will either return the CRC32 checksum of an existing
// keyfile or, if the file does not exist and create is true then a
// keyfile will be created with new contents along with any intermediate
// directories, and the checksum of the new file will be returned. On
// error the checksum is undefined and err will indicate why. If create
// is true then directories and a file may have been created even on
// error.
func (k *KeyFile) ReadOrCreate(h host.Host) (crc32 uint32, created bool, err error) {
	if kv, err2 := k.Read(h); err2 == nil { // ok?
		crc32, err = kv.Checksum()
		return
	}

	// only try to create if the file error is a not exists
	if _, err = h.Stat(k.String()); err != nil && errors.Is(err, fs.ErrNotExist) {
		if err = h.MkdirAll(k.Dir(), 0775); err != nil {
			err = fmt.Errorf("failed to create keyfile directory %q: %w", k.Dir(), err)
			return
		}
		kv := NewRandomKeyValues()
		if err = h.WriteFile(k.String(), []byte(kv.String()), 0600); err != nil {
			err = fmt.Errorf("failed to write keyfile to %q: %w", k, err)
			return
		}

		created = true
		crc32, err = kv.Checksum()
	}
	return
}

// ReadCRC will return the CRC32 checksum of an existing keyfile, or an
// error if the file cannot be read.
func (k *KeyFile) ReadCRC(h host.Host) (crc32 uint32, err error) {
	kv, err := k.Read(h)
	if err == nil {
		crc32, err = kv.Checksum()
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
func (k *KeyFile) EncodeString(h host.Host, plaintext string, expandable bool) (out string, err error) {
	a, err := k.Read(h)
	if err != nil {
		return "", err
	}

	e, err := a.EncodeString(plaintext)
	if err != nil {
		return "", err
	}

	if expandable {
		out = fmt.Sprintf("${enc:%s:+encs+%s}", ExpandHome(string(*k)), e)
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
func (k *KeyFile) Encode(h host.Host, plaintext *Plaintext, expandable bool) (out string, err error) {
	kv, err := k.Read(h)
	if err != nil {
		return
	}

	e, err := kv.Encode(plaintext)
	if err != nil {
		return
	}

	if expandable {
		out = fmt.Sprintf("${enc:%s:+encs+%s}", KeyFile(AbbreviateHome(k.String())), e)
	} else {
		out = fmt.Sprintf("+encs+%s", e)
	}
	return
}

// DecodeString decodes the input as a string using keyfile and return
// plaintext. An error is returned if the keyfile is not readable.
func (k *KeyFile) DecodeString(h host.Host, input string) (plaintext string, err error) {
	a, err := k.Read(h)
	if err != nil {
		return
	}
	return a.DecodeString(input)
}

// Decode input as a byte slice using keyfile and return byte slice
// plaintext. An error is returned if the keyfile is not readable.
func (k *KeyFile) Decode(h host.Host, input []byte) (plaintext []byte, err error) {
	a, err := k.Read(h)
	if err != nil {
		return
	}
	return a.Decode(input)
}

// DecodeEnclave decodes the input using the keyfile k and returns a
// memguard.Enclave
func (k *KeyFile) DecodeEnclave(h host.Host, input []byte) (plaintext *memguard.Enclave, err error) {
	a, err := k.Read(h)
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
func (k *KeyFile) EncodePasswordInput(h host.Host, expandable bool) (out string, err error) {
	plaintext, err := ReadPasswordInput(true, 3)
	if err != nil {
		return
	}
	out, err = k.Encode(h, plaintext, expandable)
	return
}
