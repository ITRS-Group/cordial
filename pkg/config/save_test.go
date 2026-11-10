package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial/pkg/host"
)

func TestWritePreservesSettingsAndPermissions(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "gateway.json")
	if err := os.WriteFile(filename, []byte(`{"number":7039,"location":"old.xml"}`), 0640); err != nil {
		t.Fatal(err)
	}
	settings, err := Read("gateway", FilePath(filename), UseDefaults(false), MustExist())
	if err != nil {
		t.Fatal(err)
	}
	Set(settings, "location", "new.xml")
	if err := settings.Write("gateway", FilePath(filename)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("7039")) || !bytes.Contains(data, []byte("new.xml")) {
		t.Fatalf("write changed unrelated values: %s", data)
	}
	if info, err := os.Stat(filename); err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("write changed permissions: %v, %v", info, err)
	}
}

type configRenameFailureHost struct{ host.Host }

func (configRenameFailureHost) Rename(string, string) error { return fs.ErrPermission }

func TestWriteReturnsRenameFailure(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "gateway.json")
	original := []byte(`{"location":"old.xml"}`)
	if err := os.WriteFile(filename, original, 0640); err != nil {
		t.Fatal(err)
	}
	settings := New()
	Set(settings, "location", "new.xml")
	err := settings.Write("gateway", FilePath(filename), Host(configRenameFailureHost{Host: host.NewLocal()}))
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Write() error = %v; want rename permission error", err)
	}
	if data, err := os.ReadFile(filename); err != nil || !bytes.Equal(data, original) {
		t.Fatalf("failed write changed original: %s, %v", data, err)
	}
	if temporary, err := filepath.Glob(filename + ".*"); err != nil || len(temporary) != 0 {
		t.Fatalf("failed write left temporary files: %v, %v", temporary, err)
	}
}
