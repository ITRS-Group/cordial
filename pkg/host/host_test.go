package host

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileToDirectory(t *testing.T) {
	h := NewLocal()
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	destination := filepath.Join(root, "destination")
	if err := os.WriteFile(source, []byte("copy me"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(h, source, h, destination); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "source.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "copy me" {
		t.Errorf("copied contents = %q, want %q", got, "copy me")
	}
}

func TestCopyFileRejectsDirectory(t *testing.T) {
	h := NewLocal()
	root := t.TempDir()
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(h, directory, h, filepath.Join(root, "copy")); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("CopyFile(directory) error = %v, want %v", err, os.ErrInvalid)
	}
}

func TestCopyAllCopiesNestedFiles(t *testing.T) {
	h := NewLocal()
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "data.txt"), []byte("nested data"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := CopyAll(h, source, h, destination); err != nil {
		t.Fatalf("CopyAll() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "nested", "data.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "nested data" {
		t.Errorf("copied contents = %q, want %q", got, "nested data")
	}
}

func TestCopyAllReturnsMissingSourceError(t *testing.T) {
	h := NewLocal()
	root := t.TempDir()
	missing := filepath.Join(root, "missing")
	if err := CopyAll(h, missing, h, filepath.Join(root, "destination")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("CopyAll(missing source) error = %v, want %v", err, os.ErrNotExist)
	}
}

func TestLocalPathOperationsUseLocalRules(t *testing.T) {
	h := NewLocal()
	if got, want := h.Clean("a/../b"), filepath.Clean("a/../b"); got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
	base := filepath.Join("a", "b")
	target := filepath.Join("a", "c")
	got, err := h.Rel(base, target)
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	want, err := filepath.Rel(base, target)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Rel() = %q, want %q", got, want)
	}
}

func TestSSHPathOperationsUseSlashRules(t *testing.T) {
	h := NewSSHRemote("test")
	if got := h.Join("/opt/geneos", "packages", "../shared"); got != "/opt/geneos/shared" {
		t.Errorf("Join() = %q, want %q", got, "/opt/geneos/shared")
	}
	if got := h.Base("/opt/geneos/file"); got != "file" {
		t.Errorf("Base() = %q, want %q", got, "file")
	}
	if got := h.Dir("/opt/geneos/file"); got != "/opt/geneos" {
		t.Errorf("Dir() = %q, want %q", got, "/opt/geneos")
	}
	if got := h.Clean("/opt/geneos/../shared"); got != "/opt/shared" {
		t.Errorf("Clean() = %q, want %q", got, "/opt/shared")
	}
	if !h.IsAbs("/opt/geneos") || h.IsAbs(`C:\opt\geneos`) {
		t.Error("IsAbs() did not apply slash-path semantics")
	}
	got, err := h.Rel("/opt/geneos", "/opt/shared/file")
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	if got != "../shared/file" {
		t.Errorf("Rel() = %q, want %q", got, "../shared/file")
	}
	if _, err := h.Rel("/opt/geneos", "opt/shared"); err == nil {
		t.Error("Rel() with absolute and relative paths succeeded")
	}
	if got := h.Clean(`C:\opt\geneos`); got != `C:\opt\geneos` {
		t.Errorf("Clean() = %q, want slash-oriented unchanged input", got)
	}
	if got := h.ToSlash(`C:\opt\geneos`); got != "C:/opt/geneos" {
		t.Errorf("ToSlash() = %q, want %q", got, "C:/opt/geneos")
	}
	if got := h.VolumeName("/opt/geneos"); got != "" {
		t.Errorf("VolumeName() = %q, want empty", got)
	}
}
