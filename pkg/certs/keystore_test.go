package certs

import (
	"bytes"
	"crypto/x509"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// testKeyAndChain returns a leaf certificate and key and its root
func testKeyAndChain(t *testing.T, cn string) (PrivateKey, []*x509.Certificate) {
	t.Helper()
	root, rootKey := createTestRoot(t)
	leaf, leafKey, err := CreateCertificate(Template(cn), root, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	return leafKey, []*x509.Certificate{leaf, root}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAddCertChainToKeyStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keystore.db")
	key1, chain1 := testKeyAndChain(t, "one")
	key2, chain2 := testKeyAndChain(t, "two")

	// new keystore with the default password
	if err := AddCertChainToKeyStore(host.Localhost, path, nil, "one", key1, chain1...); err != nil {
		t.Fatal(err)
	}
	// existing keystore is added to
	if err := AddCertChainToKeyStore(host.Localhost, path, nil, "two", key2, chain2...); err != nil {
		t.Fatal(err)
	}
	ks, err := ReadKeystore(host.Localhost, path, config.Secret("changeit"))
	if err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"one", "two"} {
		if _, err := ks.GetPrivateKeyEntry(alias, []byte("changeit")); err != nil {
			t.Errorf("alias %q: %v", alias, err)
		}
	}
}

func TestAddCertChainToKeyStoreDoesNotWipeExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keystore.db")
	key1, chain1 := testKeyAndChain(t, "one")
	key2, chain2 := testKeyAndChain(t, "two")
	if err := AddCertChainToKeyStore(host.Localhost, path, config.Secret("secret1"), "one", key1, chain1...); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, path)

	// wrong password
	if err := AddCertChainToKeyStore(host.Localhost, path, config.Secret("wrong-password"), "two", key2, chain2...); err == nil {
		t.Fatal("AddCertChainToKeyStore() with the wrong password succeeded")
	}
	if !bytes.Equal(before, readFile(t, path)) {
		t.Fatal("keystore was changed after a failed update with the wrong password")
	}

	// no key (an error adding the key must not result in a keystore
	// being written without it)
	if err := AddCertChainToKeyStore(host.Localhost, path, config.Secret("secret1"), "two", key2); err == nil {
		t.Fatal("AddCertChainToKeyStore() without a certificate chain succeeded")
	}
	if !bytes.Equal(before, readFile(t, path)) {
		t.Fatal("keystore was changed after failing to add a key")
	}

	// corrupt file
	if err := os.WriteFile(path, []byte("not a keystore"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AddCertChainToKeyStore(host.Localhost, path, config.Secret("secret1"), "two", key2, chain2...); err == nil {
		t.Fatal("AddCertChainToKeyStore() on a corrupt file succeeded")
	}
	if got := readFile(t, path); string(got) != "not a keystore" {
		t.Fatal("corrupt keystore was overwritten")
	}
}

func TestAddCertChainToKeyStoreEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keystore.db")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	key, chain := testKeyAndChain(t, "one")
	if err := AddCertChainToKeyStore(host.Localhost, path, nil, "one", key, chain...); err != nil {
		t.Fatalf("AddCertChainToKeyStore() on an empty file error = %v", err)
	}
	if _, err := ReadKeystore(host.Localhost, path, nil); err != nil {
		t.Fatal(err)
	}
}

func TestReadKeystoreErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadKeystore(host.Localhost, filepath.Join(dir, "missing.db"), nil); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadKeystore(missing) error = %v, want os.ErrNotExist", err)
	}

	path := filepath.Join(dir, "keystore.db")
	key, chain := testKeyAndChain(t, "one")
	if err := AddCertChainToKeyStore(host.Localhost, path, config.Secret("secret1"), "one", key, chain...); err != nil {
		t.Fatal(err)
	}
	_, err := ReadKeystore(host.Localhost, path, config.Secret("wrong-password"))
	if err == nil || errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadKeystore(wrong password) error = %v, want an error that is not os.ErrNotExist", err)
	}
}

// failingHost fails writes to files being created
type failingHost struct {
	host.Host
}

type failingWriter struct{ io.Writer }

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }
func (failingWriter) Close() error              { return nil }

func (h failingHost) Create(p string, perms os.FileMode) (io.WriteCloser, error) {
	w, err := h.Host.Create(p, perms)
	if err != nil {
		return nil, err
	}
	w.Close()
	return failingWriter{}, nil
}

func TestWriteKeystoreIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keystore.db")
	key, chain := testKeyAndChain(t, "one")
	if err := AddCertChainToKeyStore(host.Localhost, path, nil, "one", key, chain...); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, path)

	ks, err := ReadKeystore(host.Localhost, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = ks.WriteKeystore(failingHost{host.Localhost}, path, nil, 0o600); err == nil {
		t.Fatal("WriteKeystore() succeeded with a failing writer")
	}
	if !bytes.Equal(before, readFile(t, path)) {
		t.Error("existing keystore was modified by a failed write")
	}

	if err = ks.WriteKeystore(host.Localhost, path, nil, 0); !errors.Is(err, os.ErrInvalid) {
		t.Errorf("WriteKeystore() with no permissions error = %v, want os.ErrInvalid", err)
	}

	// no temporary files are left behind
	if err = ks.WriteKeystore(host.Localhost, path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "keystore.db" {
		t.Errorf("directory contains %v, want only keystore.db", entries)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(path); err != nil || fi.Mode().Perm()&0o077 != 0 {
			t.Errorf("keystore mode = %v, %v, want no group or other access", fi.Mode(), err)
		}
	}
}

func TestChangePassword(t *testing.T) {
	for name, tt := range map[string]struct {
		oldPassword, newPassword config.Secret
		storePassword            config.Secret
		wantNew                  string
	}{
		"explicit": {config.Secret("secret1"), config.Secret("secret2"), config.Secret("secret1"), "secret2"},
		// an empty old password is the default "changeit", which must
		// also be used for the keys
		"default old": {nil, config.Secret("secret2"), nil, "secret2"},
		"default new": {config.Secret("secret1"), nil, config.Secret("secret1"), "changeit"},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "keystore.db")
			key, chain := testKeyAndChain(t, "one")
			if err := AddCertChainToKeyStore(host.Localhost, path, tt.storePassword, "one", key, chain...); err != nil {
				t.Fatal(err)
			}
			if runtime.GOOS != "windows" {
				if err := os.Chmod(path, 0o640); err != nil {
					t.Fatal(err)
				}
			}

			if err := ChangePassword(host.Localhost, path, tt.oldPassword, tt.newPassword); err != nil {
				t.Fatal(err)
			}

			ks, err := ReadKeystore(host.Localhost, path, config.Secret(tt.wantNew))
			if err != nil {
				t.Fatalf("keystore cannot be read with the new password: %v", err)
			}
			entry, err := ks.GetPrivateKeyEntry("one", []byte(tt.wantNew))
			if err != nil {
				t.Fatalf("private key cannot be read with the new password: %v", err)
			}
			if !bytes.Equal(entry.PrivateKey, key) {
				t.Error("private key changed")
			}
			if _, err := ks.GetPrivateKeyEntry("one", []byte("secret1")); err == nil && tt.wantNew != "secret1" {
				t.Error("private key can still be read with the old password")
			}
			if runtime.GOOS != "windows" {
				if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o640 {
					t.Errorf("mode = %v, %v, want 0640 to be preserved", fi.Mode(), err)
				}
			}
		})
	}
}

func TestChangePasswordFailuresLeaveKeystore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keystore.db")
	key, chain := testKeyAndChain(t, "one")
	if err := AddCertChainToKeyStore(host.Localhost, path, config.Secret("secret1"), "one", key, chain...); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, path)

	if err := ChangePassword(host.Localhost, path, config.Secret("wrong-password"), config.Secret("secret2")); err == nil {
		t.Error("ChangePassword() with the wrong old password succeeded")
	}
	if err := ChangePassword(host.Localhost, filepath.Join(dir, "missing.db"), nil, nil); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ChangePassword(missing) error = %v, want os.ErrNotExist", err)
	}

	// a key under a different password to the store cannot be changed
	// and must not be silently skipped
	ks, err := ReadKeystore(host.Localhost, path, config.Secret("secret1"))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := ks.GetPrivateKeyEntry("one", []byte("secret1"))
	if err != nil {
		t.Fatal(err)
	}
	if err = ks.SetPrivateKeyEntry("one", entry, []byte("different")); err != nil {
		t.Fatal(err)
	}
	if err = ks.WriteKeystore(host.Localhost, path, config.Secret("secret1"), 0o600); err != nil {
		t.Fatal(err)
	}
	before = readFile(t, path)
	if err = ChangePassword(host.Localhost, path, config.Secret("secret1"), config.Secret("secret2")); err == nil {
		t.Error("ChangePassword() succeeded with a key that cannot be decrypted")
	}
	if !bytes.Equal(before, readFile(t, path)) {
		t.Error("keystore was changed after a failed ChangePassword()")
	}
}

func trustedAliases(t *testing.T, path string, password config.Secret) map[string]*x509.Certificate {
	t.Helper()
	ks, err := ReadKeystore(host.Localhost, path, password)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]*x509.Certificate{}
	for _, alias := range ks.Aliases() {
		e, err := ks.GetTrustedCertificateEntry(alias)
		if err != nil {
			t.Fatalf("alias %q: %v", alias, err)
		}
		c, err := ParseTrustedCertificate(e)
		if err != nil {
			t.Fatal(err)
		}
		m[alias] = c
	}
	return m
}

func TestTrustStoreAliases(t *testing.T) {
	now := time.Now()
	a, _ := testCA(t, "Same Name", now.Add(-time.Hour), now.Add(time.Hour), x509.KeyUsageCertSign)
	b, _ := testCA(t, "Same Name", now.Add(-time.Hour), now.Add(time.Hour), x509.KeyUsageCertSign)
	noCN1, _ := testCA(t, "", now.Add(-time.Hour), now.Add(time.Hour), x509.KeyUsageCertSign)
	noCN2, _ := testCA(t, "", now.Add(-time.Hour), now.Add(time.Hour), x509.KeyUsageCertSign)

	all := []*x509.Certificate{a, b, noCN1, noCN2, a}

	check := func(t *testing.T, path string) {
		t.Helper()
		got := trustedAliases(t, path, nil)
		if len(got) != 4 {
			t.Fatalf("truststore has %d entries (%v), want 4", len(got), got)
		}
		for _, want := range []*x509.Certificate{a, b, noCN1, noCN2} {
			found := false
			for _, c := range got {
				found = found || c.Equal(want)
			}
			if !found {
				t.Errorf("certificate with serial %v is missing from the truststore", want.SerialNumber)
			}
		}
		if _, ok := got["same name"]; !ok {
			t.Errorf("first certificate does not have the common name as alias: %v", got)
		}
	}

	t.Run("WriteTrustStore", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "truststore.db")
		if err := WriteTrustStore(host.Localhost, path, nil, all...); err != nil {
			t.Fatal(err)
		}
		check(t, path)
	})

	t.Run("AddRootsToTrustStore", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "truststore.db")
		for _, c := range all {
			if err := AddRootsToTrustStore(host.Localhost, path, nil, c); err != nil {
				t.Fatal(err)
			}
		}
		check(t, path)
	})
}

func TestAddRootsToTrustStoreDoesNotWipeExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "truststore.db")
	a, _ := validRoot(t, "a")
	b, _ := validRoot(t, "b")
	if err := AddRootsToTrustStore(host.Localhost, path, config.Secret("secret1"), a); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, path)
	if err := AddRootsToTrustStore(host.Localhost, path, config.Secret("wrong-password"), b); err == nil {
		t.Fatal("AddRootsToTrustStore() with the wrong password succeeded")
	}
	if !bytes.Equal(before, readFile(t, path)) {
		t.Fatal("truststore was overwritten")
	}
}
