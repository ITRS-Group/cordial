package config

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/host"
)

func TestKeyFileReadOrCreate(t *testing.T) {
	h := host.NewLocal()

	t.Run("creates with restrictive modes", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "new", "keys")
		k := KeyFile(filepath.Join(dir, "k.aes"))
		crc, created, err := k.ReadOrCreate(h)
		if err != nil || !created || crc == 0 {
			t.Fatalf("ReadOrCreate() = %v, %v, %v", crc, created, err)
		}
		if again, created, err := k.ReadOrCreate(h); err != nil || created || again != crc {
			t.Errorf("second ReadOrCreate() = %v, %v, %v; want %v, false, nil", again, created, err, crc)
		}
		if runtime.GOOS == "windows" {
			return
		}
		if st, err := os.Stat(dir); err != nil || st.Mode().Perm() != 0700 {
			t.Errorf("directory mode = %v, %v; want 0700", st, err)
		}
		if st, err := os.Stat(string(k)); err != nil || st.Mode().Perm() != 0600 {
			t.Errorf("file mode = %v, %v; want 0600", st, err)
		}
	})

	t.Run("corrupt keyfile is an error and is not replaced", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "k.aes")
		if err := os.WriteFile(file, []byte("this is not a keyfile\n"), 0600); err != nil {
			t.Fatal(err)
		}
		k := KeyFile(file)
		crc, created, err := k.ReadOrCreate(h)
		if err == nil {
			t.Fatalf("ReadOrCreate() = %v, %v, nil; want an error for a corrupt keyfile", crc, created)
		}
		if b, _ := os.ReadFile(file); string(b) != "this is not a keyfile\n" {
			t.Errorf("corrupt keyfile was changed: %q", b)
		}
	})

	t.Run("unreadable keyfile is an error", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Getuid() == 0 {
			t.Skip("cannot make a file unreadable")
		}
		file := filepath.Join(t.TempDir(), "k.aes")
		kv := NewKeyValues()
		if err := os.WriteFile(file, []byte(kv.String()), 0000); err != nil {
			t.Fatal(err)
		}
		k := KeyFile(file)
		if _, created, err := k.ReadOrCreate(h); err == nil || created || errors.Is(err, fs.ErrNotExist) {
			t.Errorf("ReadOrCreate() = _, %v, %v; want a permission error", created, err)
		}
	})
}

func TestKeyFileCreateWithBackup(t *testing.T) {
	h := host.NewLocal()

	t.Run("without backup overwrites", func(t *testing.T) {
		dir := t.TempDir()
		k := KeyFile(filepath.Join(dir, "k.aes"))
		first, _, err := k.ReadOrCreate(h)
		if err != nil {
			t.Fatal(err)
		}
		second, err := k.CreateWithBackup(h, "")
		if err != nil {
			t.Fatal(err)
		}
		if second == first {
			t.Error("CreateWithBackup(h, \"\") returned the checksum of the existing keyfile, nothing was overwritten")
		}
		if now, err := k.ReadCRC(h); err != nil || now != second {
			t.Errorf("keyfile checksum on disk = %v, %v; want %v", now, err, second)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 1 {
			t.Errorf("unexpected files: %v", entries)
		}
	})

	t.Run("with backup keeps the old file", func(t *testing.T) {
		dir := t.TempDir()
		k := KeyFile(filepath.Join(dir, "k.aes"))
		first, _, err := k.ReadOrCreate(h)
		if err != nil {
			t.Fatal(err)
		}
		second, err := k.CreateWithBackup(h, ".prev")
		if err != nil || second == first {
			t.Fatalf("CreateWithBackup() = %v, %v; want a new keyfile", second, err)
		}
		backup := KeyFile(filepath.Join(dir, "k.prev.aes"))
		if crc, err := backup.ReadCRC(h); err != nil || crc != first {
			t.Errorf("backup checksum = %v, %v; want %v", crc, err, first)
		}
	})

	t.Run("no existing file", func(t *testing.T) {
		k := KeyFile(filepath.Join(t.TempDir(), "k.aes"))
		if crc, err := k.CreateWithBackup(h, ""); err != nil || crc == 0 {
			t.Errorf("CreateWithBackup() = %v, %v", crc, err)
		}
	})
}

func TestMigrateFile(t *testing.T) {
	h := host.NewLocal()
	write := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0775); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(filepath.Base(p)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	exists := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}

	t.Run("old file moved to a new path that does not exist", func(t *testing.T) {
		dir := t.TempDir()
		oldPath, newPath := filepath.Join(dir, "old.json"), filepath.Join(dir, "sub", "new.json")
		write(oldPath)
		if got := MigrateFile(h, newPath, filepath.Join(dir, "missing.json"), oldPath); got != newPath {
			t.Fatalf("MigrateFile() = %q, want %q", got, newPath)
		}
		if exists(oldPath) || !exists(newPath) {
			t.Error("file was not moved")
		}
	})

	t.Run("existing new path wins", func(t *testing.T) {
		dir := t.TempDir()
		oldPath, newPath := filepath.Join(dir, "old.json"), filepath.Join(dir, "new.json")
		write(oldPath)
		write(newPath)
		if got := MigrateFile(h, newPath, oldPath); got != newPath || !exists(oldPath) {
			t.Errorf("MigrateFile() = %q, old file exists: %v", got, exists(oldPath))
		}
	})

	t.Run("nothing to migrate", func(t *testing.T) {
		dir := t.TempDir()
		if got := MigrateFile(h, filepath.Join(dir, "new.json"), filepath.Join(dir, "old.json"), ""); got != "" {
			t.Errorf("MigrateFile() = %q, want empty", got)
		}
	})

	t.Run("move into a directory", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target")
		oldPath := filepath.Join(dir, "old", "f.txt")
		write(oldPath)
		if err := os.MkdirAll(target, 0775); err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(target, "f.txt")
		if got := MigrateFile(h, target, oldPath); got != want || !exists(want) || exists(oldPath) {
			t.Fatalf("MigrateFile() = %q", got)
		}
		// a second call finds it where it was moved to
		if got := MigrateFile(h, target, oldPath); got != want {
			t.Errorf("second MigrateFile() = %q, want %q", got, want)
		}
	})

	t.Run("no new path", func(t *testing.T) {
		dir := t.TempDir()
		oldPath := filepath.Join(dir, "old.json")
		write(oldPath)
		if got := MigrateFile(h, "", filepath.Join(dir, "none"), oldPath); got != oldPath {
			t.Errorf("MigrateFile() = %q, want %q", got, oldPath)
		}
	})
}

func TestAbbreviateHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("home directory is not taken from $HOME")
	}
	t.Setenv("HOME", "/home/peter")
	tests := map[string]string{
		"/home/peter":           "~",
		"/home/peter/.config/x": "~/.config/x",
		"/home/peter2/x":        "/home/peter2/x",
		"/home/peterx":          "/home/peterx",
		"/elsewhere//x/":        "/elsewhere/x",
		"/home/pete":            "/home/pete",
	}
	for in, want := range tests {
		if got := AbbreviateHome(in); got != want {
			t.Errorf("AbbreviateHome(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCredentials(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "creds")
	options := []FileOption{FromDir(dir), Format("json")}

	for _, c := range []Credentials{
		{Domain: "itrsgroup.com", Username: "plain"},
		{Domain: "gateway:*", Username: "glob"},
		{Domain: "gateway:gw1", Username: "exact"},
	} {
		if err := AddCreds(c, options...); err != nil {
			t.Fatal(err)
		}
	}

	if runtime.GOOS != "windows" {
		if st, err := os.Stat(filepath.Join(dir, "credentials.json")); err != nil || st.Mode().Perm() != 0600 {
			t.Errorf("credentials file mode = %v, %v; want 0600", st, err)
		}
		if st, err := os.Stat(dir); err != nil || st.Mode().Perm() != 0700 {
			t.Errorf("credentials directory mode = %v, %v; want 0700", st, err)
		}
	}

	tests := []struct {
		find string
		want string // username, or empty for no match
	}{
		{"https://resources.itrsgroup.com/download/x", "plain"},
		{"gateway:gw1", "exact"},
		{"gateway:gw2", "glob"},
		{"gateway:*", "glob"},
		{"https://example.com", ""},
	}
	for _, test := range tests {
		creds := FindCreds(test.find, options...)
		if test.want == "" {
			if creds != nil {
				t.Errorf("FindCreds(%q) matched %q, want none", test.find, Get[string](creds, "username"))
			}
			continue
		}
		if creds == nil {
			t.Errorf("FindCreds(%q) found nothing, want %q", test.find, test.want)
			continue
		}
		if got := Get[string](creds, "username"); got != test.want {
			t.Errorf("FindCreds(%q) = %q, want %q", test.find, got, test.want)
		}
	}

	if err := DeleteCreds("gateway:gw1", options...); err != nil {
		t.Fatal(err)
	}
	if creds := FindCreds("gateway:gw1", options...); creds == nil || Get[string](creds, "username") != "glob" {
		t.Error("deleted credentials still found")
	}
}

func TestCredentialsDefaultDelimiterCanBeOverridden(t *testing.T) {
	opts := evalFileOptions(credentialsOptions([]FileOption{KeyDelimiter("/")})...)
	if opts.delimiter != "/" {
		t.Errorf("delimiter = %q, want the caller's", opts.delimiter)
	}
	opts = evalFileOptions(credentialsOptions(nil)...)
	if opts.delimiter != "::" {
		t.Errorf("default delimiter = %q, want ::", opts.delimiter)
	}
}

func TestReadPEM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("-----BEGIN TEST-----\n"))
	}))
	url := srv.URL

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.WriteFile(filepath.Join(home, "x.pem"), []byte("from file"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		from    string
		want    string
		wantErr bool
	}{
		{name: "empty", from: "", want: ""},
		{name: "literal", from: "pem:abc", want: "abc"},
		{name: "file in home", from: "~/x.pem", want: "from file"},
		{name: "missing file", from: filepath.Join(home, "nothing.pem"), wantErr: true},
		{name: "url", from: url + "/ok", want: "-----BEGIN TEST-----\n"},
		{name: "url with error status", from: url + "/missing", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ReadPEM(test.from, "test")
			if (err != nil) != test.wantErr {
				t.Fatalf("ReadPEM() error = %v, wantErr %v", err, test.wantErr)
			}
			if string(got) != test.want {
				t.Errorf("ReadPEM() = %q, want %q", got, test.want)
			}
		})
	}

	// a failure to connect must be an error, not a panic
	srv.Close()
	if got, err := ReadPEM(url+"/ok", "test"); err == nil || got != nil {
		t.Errorf("ReadPEM() on a closed server = %q, %v; want an error", got, err)
	}
}

// bodyCloseSpy records whether the response body was closed
type bodyCloseSpy struct {
	io.Reader
	closed *bool
}

func (b bodyCloseSpy) Close() error { *b.closed = true; return nil }

type spyTransport struct{ closed *bool }

func (s spyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     "500 Internal Server Error",
		Body:       bodyCloseSpy{Reader: strings.NewReader("oops"), closed: s.closed},
	}, nil
}

func TestFetchURLClosesBodyOnErrorStatus(t *testing.T) {
	var closed bool
	old := fetchClient
	fetchClient = &http.Client{Transport: spyTransport{&closed}}
	t.Cleanup(func() { fetchClient = old })

	if _, err := fetchURL(nil, "http://example.invalid/x", true); err == nil {
		t.Fatal("fetchURL() with a 500 status did not return an error")
	}
	if !closed {
		t.Error("response body not closed after an error status")
	}
	if old.Timeout == 0 {
		t.Error("the default client for fetching URLs has no timeout")
	}
}
