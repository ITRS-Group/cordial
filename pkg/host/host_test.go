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
