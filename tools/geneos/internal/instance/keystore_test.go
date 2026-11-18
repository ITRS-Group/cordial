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
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pavlo-v-chernykh/keystore-go/v4"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

func localHost() host.Host { return host.Localhost }

// writeTestKeyStore writes a JKS truststore with the named trusted
// entries, protected by password.
func writeTestKeyStore(t *testing.T, path, password string, mode os.FileMode, aliases ...string) []byte {
	t.Helper()
	ks := certs.NewKeyStore()
	for _, a := range aliases {
		if err := ks.SetTrustedCertificateEntry(a, keystore.TrustedCertificateEntry{
			Certificate: keystore.Certificate{Type: "X509", Content: []byte(a)},
		}); err != nil {
			t.Fatal(err)
		}
	}
	data, err := MarshalKeyStore(ks, config.Secret(password))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return data
}

func skipIfCannotDeny(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file permissions are not enforced")
	}
}

func assertNoStrays(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("directory %s contains %v, want %v (temporary or backup files left behind?)", dir, got, want)
	}
}

func TestLoadKeyStoreFailures(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.db")
	data := writeTestKeyStore(t, good, "secret", 0o600, "one")

	tests := []struct {
		name    string
		content []byte
		unread  bool
		pass    string
		is      error
		text    string
	}{
		{"wrong password", data, false, "wrong-password", ErrKeyStorePassword, "incorrect password"},
		{"corrupt", bytes.Repeat([]byte("not a keystore "), 20), false, "secret", ErrKeyStoreCorrupt, "not a Java keystore"},
		{"truncated", data[:len(data)/2], false, "secret", ErrKeyStoreCorrupt, "truncated"},
		{"permission denied", data, true, "secret", os.ErrPermission, "permission denied"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unread {
				skipIfCannotDeny(t)
			}
			p := filepath.Join(t.TempDir(), "ks.db")
			if err := os.WriteFile(p, tt.content, 0o600); err != nil {
				t.Fatal(err)
			}
			if tt.unread {
				if err := os.Chmod(p, 0); err != nil {
					t.Fatal(err)
				}
			}

			ks, existing, err := LoadKeyStore(localHost(), p, config.Secret(tt.pass))
			if err == nil || ks != nil || existing {
				t.Fatalf("LoadKeyStore() = %v, %v, %v, want a nil keystore and an error", ks, existing, err)
			}
			if !errors.Is(err, tt.is) {
				t.Errorf("error %q is not %v", err, tt.is)
			}
			if !strings.Contains(err.Error(), p) || !strings.Contains(err.Error(), tt.text) {
				t.Errorf("error %q does not name the file %q and say %q", err, p, tt.text)
			}
		})
	}
}

func TestLoadKeyStoreNew(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.db")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]string{"missing": filepath.Join(dir, "missing.db"), "zero length": empty} {
		t.Run(name, func(t *testing.T) {
			ks, existing, err := LoadKeyStore(localHost(), p, config.Secret("secret"))
			if err != nil || ks == nil || existing || len(ks.Aliases()) != 0 {
				t.Errorf("LoadKeyStore() = %v, %v, %v, want a new empty keystore", ks, existing, err)
			}
		})
	}
}

func TestWriteFilesAtomic(t *testing.T) {
	dir := t.TempDir()
	a, b, c := filepath.Join(dir, "a"), filepath.Join(dir, "b"), filepath.Join(dir, "c")
	if err := os.WriteFile(a, []byte("old a"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(a, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("old b"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := WriteFilesAtomic(localHost(),
		FileUpdate{Path: a, Data: []byte("new a"), Perm: 0o600},
		FileUpdate{Path: b, Data: []byte("new b"), Perm: 0o644},
		FileUpdate{Path: c, Data: []byte("new c"), Perm: 0o600},
	)
	if err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]string{a: "new a", b: "new b", c: "new c"} {
		if got, _ := os.ReadFile(p); string(got) != want {
			t.Errorf("%s = %q, want %q", p, got, want)
		}
	}
	if runtime.GOOS != "windows" {
		// the mode of an existing file is kept, new files use the given mode
		for p, want := range map[string]os.FileMode{a: 0o640, b: 0o600, c: 0o600} {
			if st, _ := os.Stat(p); st.Mode().Perm() != want {
				t.Errorf("%s mode = %v, want %v", p, st.Mode().Perm(), want)
			}
		}
	}
	assertNoStrays(t, dir, "a", "b", "c")
}

// failHost fails selected operations to simulate an interrupted update
type failHost struct {
	host.Host
	failRename string // fail renaming a temporary file to this path
}

func (h failHost) Rename(oldpath, newpath string) error {
	if newpath == h.failRename && strings.Contains(oldpath, ".tmp-") {
		return errors.New("simulated rename failure")
	}
	return h.Host.Rename(oldpath, newpath)
}

func TestWriteFilesAtomicRollback(t *testing.T) {
	dir := t.TempDir()
	a, b, c := filepath.Join(dir, "a"), filepath.Join(dir, "b"), filepath.Join(dir, "c")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("old "+filepath.Base(p)), 0o640); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, 0o640); err != nil {
			t.Fatal(err)
		}
	}

	// a is replaced, c is created and then b fails: all must be undone
	h := failHost{Host: localHost(), failRename: b}
	err := WriteFilesAtomic(h,
		FileUpdate{Path: a, Data: []byte("new a"), Perm: 0o600},
		FileUpdate{Path: c, Data: []byte("new c"), Perm: 0o600},
		FileUpdate{Path: b, Data: []byte("new b"), Perm: 0o600},
	)
	if err == nil {
		t.Fatal("WriteFilesAtomic() succeeded, want an error")
	}
	for _, want := range []string{b, "simulated rename failure", "existing files restored"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
	for p, want := range map[string]string{a: "old a", b: "old b"} {
		if got, _ := os.ReadFile(p); string(got) != want {
			t.Errorf("%s = %q after rollback, want %q", p, got, want)
		}
		if runtime.GOOS != "windows" {
			if st, _ := os.Stat(p); st.Mode().Perm() != 0o640 {
				t.Errorf("%s mode = %v after rollback, want 0640", p, st.Mode().Perm())
			}
		}
	}
	assertNoStrays(t, dir, "a", "b")
}

func TestWriteFilesAtomicReadOnlyDirectory(t *testing.T) {
	skipIfCannotDeny(t)

	dir := t.TempDir()
	ro := filepath.Join(dir, "ro")
	if err := os.Mkdir(ro, 0o755); err != nil {
		t.Fatal(err)
	}
	a, b := filepath.Join(dir, "a"), filepath.Join(ro, "b")
	if err := os.WriteFile(a, []byte("old a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("old b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })

	// the second file cannot be written, so the first must not be replaced
	err := WriteFilesAtomic(localHost(),
		FileUpdate{Path: a, Data: []byte("new a"), Perm: 0o600},
		FileUpdate{Path: b, Data: []byte("new b"), Perm: 0o600},
	)
	if err == nil {
		t.Fatal("WriteFilesAtomic() succeeded, want an error")
	}
	if !strings.Contains(err.Error(), b) || !strings.Contains(err.Error(), "permission denied") || !strings.Contains(err.Error(), "not changed") {
		t.Errorf("error %q does not name %q, give the reason and say nothing changed", err, b)
	}
	for p, want := range map[string]string{a: "old a", b: "old b"} {
		if got, _ := os.ReadFile(p); string(got) != want {
			t.Errorf("%s = %q, want %q", p, got, want)
		}
	}
	assertNoStrays(t, dir, "a", "ro")
	assertNoStrays(t, ro, "b")
}
