/*
Copyright © 2026 ITRS Group

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

package instance

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	"github.com/pavlo-v-chernykh/keystore-go/v4"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

var (
	// ErrKeyStorePassword is returned, wrapped, when a keystore cannot
	// be read because the password is incorrect or the file has been
	// modified, which cannot be told apart.
	ErrKeyStorePassword = errors.New("incorrect password or file has been modified")

	// ErrKeyStoreCorrupt is returned, wrapped, when a keystore file is
	// truncated or is not a valid Java keystore.
	ErrKeyStoreCorrupt = errors.New("file is truncated, corrupt or not a Java keystore")
)

// ReadKeyStore reads the Java keystore or truststore at path on h and
// returns it. All failures, including a missing file, are returned
// with an error that names the file and gives the reason, wrapping
// ErrKeyStorePassword or ErrKeyStoreCorrupt where those can be
// identified.
func ReadKeyStore(h host.Host, path string, password config.Secret) (ks *certs.KeyStore, err error) {
	data, err := h.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading keystore %q: %w", path, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("reading keystore %q: %w (file is empty)", path, ErrKeyStoreCorrupt)
	}

	ks, err = certs.ReadKeystore(h, path, password)
	if err == nil && ks != nil {
		return ks, nil
	}
	if err == nil {
		err = errors.New("no keystore returned")
	}

	// the certs library falls back to JCEKS and so loses the reason the
	// JKS format failed, so diagnose that here
	pw := []byte(password)
	if len(pw) == 0 {
		pw = []byte("changeit")
	}
	jerr := keystore.New().Load(bytes.NewReader(data), pw)
	switch {
	case jerr == nil:
		return nil, fmt.Errorf("reading keystore %q: %w", path, err)
	case strings.Contains(jerr.Error(), "invalid digest"):
		return nil, fmt.Errorf("reading keystore %q: %w", path, ErrKeyStorePassword)
	default:
		return nil, fmt.Errorf("reading keystore %q: %w (%s)", path, ErrKeyStoreCorrupt, jerr)
	}
}

// LoadKeyStore reads the keystore or truststore at path on h for
// update. Only a missing or zero-length file is treated as a new
// keystore, in which case an empty keystore is returned with existing
// set to false. Any other failure, such as a wrong password or a
// corrupt, truncated or unreadable file, is returned as an error with a
// nil keystore so that callers cannot replace a keystore they could not
// read.
func LoadKeyStore(h host.Host, path string, password config.Secret) (ks *certs.KeyStore, existing bool, err error) {
	st, err := h.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return certs.NewKeyStore(), false, nil
		}
		return nil, false, fmt.Errorf("reading keystore %q: %w", path, err)
	}
	if st.Size() == 0 {
		return certs.NewKeyStore(), false, nil
	}

	if ks, err = ReadKeyStore(h, path, password); err != nil {
		return nil, false, err
	}
	return ks, true, nil
}

// MarshalKeyStore returns the encoded form of ks, protected with
// password, suitable for FileUpdate.Data.
func MarshalKeyStore(ks *certs.KeyStore, password config.Secret) (data []byte, err error) {
	if ks == nil {
		return nil, os.ErrInvalid
	}
	if len(password) == 0 {
		password = config.Secret("changeit")
	}
	var b bytes.Buffer
	if err = ks.Store(&b, password); err != nil {
		return nil, fmt.Errorf("encoding keystore: %w", err)
	}
	return b.Bytes(), nil
}

// FileUpdate is a file to be replaced by WriteFilesAtomic.
type FileUpdate struct {
	Path string
	Data []byte
	// Perm is the mode for a new file, the mode of an existing file is
	// preserved
	Perm os.FileMode
}

// WriteFilesAtomic replaces all the files on h, which are normally the
// keystore, truststore and related key file that must stay consistent
// with each other. Either all files are replaced or none are, and on
// any error the original files are left with their original contents
// and mode and no temporary files remain.
//
// All the new contents are first written to temporary files in the same
// directories as the targets, so a failure at that stage, such as a
// read-only directory or a full disk, has not touched any target. The
// targets are then replaced by renaming over them, in order, after
// keeping each existing file under a backup name (a hard link where
// possible, so it is never missing). If a rename fails the files already
// replaced are restored from their backups. If that restore itself
// fails the returned error says so and names the backup files, which
// hold the original contents. New files are removed when rolling back.
func WriteFilesAtomic(h host.Host, files ...FileUpdate) (err error) {
	type step struct {
		FileUpdate
		tmp, backup string
		existed     bool // target existed before
		moved       bool // backup was made by moving the target
		committed   bool
	}

	steps := make([]*step, 0, len(files))
	for _, f := range files {
		steps = append(steps, &step{FileUpdate: f})
	}

	cleanup := func() {
		for _, s := range steps {
			if s.tmp != "" {
				_ = h.Remove(s.tmp)
			}
		}
	}

	for _, s := range steps {
		perm := s.Perm
		if st, serr := h.Stat(s.Path); serr == nil {
			s.existed = true
			perm = st.Mode().Perm()
		} else if !errors.Is(serr, fs.ErrNotExist) {
			cleanup()
			return fmt.Errorf("writing %q: %w", s.Path, serr)
		}
		if perm == 0 {
			perm = 0600
		}

		s.tmp = s.Path + ".tmp-" + randomSuffix()
		if err = writeTemp(h, s.tmp, s.Data, perm); err != nil {
			s.tmp = ""
			cleanup()
			return fmt.Errorf("writing %q (existing files not changed): %w", s.Path, err)
		}
	}

	rollback := func() (rerr error) {
		for _, s := range steps {
			switch {
			case s.committed && s.backup != "":
				if e := h.Rename(s.backup, s.Path); e != nil {
					rerr = errors.Join(rerr, fmt.Errorf("restoring %q, original contents are in %q: %w", s.Path, s.backup, e))
				}
			case s.committed:
				if e := h.Remove(s.Path); e != nil {
					rerr = errors.Join(rerr, fmt.Errorf("removing new file %q: %w", s.Path, e))
				}
			case s.moved:
				// target moved to backup but replacement not committed
				if e := h.Rename(s.backup, s.Path); e != nil {
					rerr = errors.Join(rerr, fmt.Errorf("restoring %q, original contents are in %q: %w", s.Path, s.backup, e))
				}
			case s.backup != "":
				_ = h.Remove(s.backup)
			}
		}
		return
	}

	for _, s := range steps {
		if s.existed {
			s.backup = s.Path + ".bak-" + randomSuffix()
			if lerr := h.Link(s.Path, s.backup); lerr != nil {
				if rerr := h.Rename(s.Path, s.backup); rerr != nil {
					s.backup = ""
					err = fmt.Errorf("cannot keep a backup of %q: %w", s.Path, errors.Join(lerr, rerr))
					break
				}
				s.moved = true
			}
		}
		if err = h.Rename(s.tmp, s.Path); err != nil {
			err = fmt.Errorf("replacing %q: %w", s.Path, err)
			break
		}
		s.tmp = ""
		s.committed = true
		s.moved = false
	}

	if err != nil {
		if rerr := rollback(); rerr != nil {
			err = fmt.Errorf("%w; rollback failed: %w", err, rerr)
		} else {
			err = fmt.Errorf("%w (existing files restored)", err)
		}
		cleanup()
		return
	}

	for _, s := range steps {
		if s.backup != "" {
			if rerr := h.Remove(s.backup); rerr != nil {
				slog.Debug("cannot remove backup file", slog.String("path", s.backup), slog.Any("error", rerr))
			}
		}
	}
	return nil
}

func randomSuffix() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// writeTemp writes data to a new file at path with mode perm.
func writeTemp(h host.Host, path string, data []byte, perm os.FileMode) (err error) {
	w, err := h.Create(path, perm)
	if err != nil {
		return
	}
	// the mode is subject to the umask, so set it explicitly where we
	// can, before any data is written
	if h.IsLocalhost() {
		if err = os.Chmod(path, perm); err != nil {
			_ = w.Close()
			_ = h.Remove(path)
			return
		}
	}
	if _, err = w.Write(data); err != nil {
		_ = w.Close()
		_ = h.Remove(path)
		return
	}
	if err = w.Close(); err != nil {
		_ = h.Remove(path)
	}
	return
}
