package geneos

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

func TestUnzipRejectsMalformedArchive(t *testing.T) {
	installDir := t.TempDir()
	localHost := &Host{Host: host.NewLocal(), Config: config.New()}
	if err := unzip(localHost, installDir, bytes.NewReader([]byte("not a zip archive")), 0, nil); err == nil {
		t.Fatal("unzip() succeeded for malformed archive")
	}
}

func TestUntarRejectsSymlinkPathTraversal(t *testing.T) {
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	if err := writer.WriteHeader(&tar.Header{
		Name:     "link",
		Typeflag: tar.TypeSymlink,
		Linkname: "../outside",
		Mode:     0777,
	}); err != nil {
		t.Fatalf("WriteHeader(symlink) error = %v", err)
	}
	content := []byte("unexpected")
	if err := writer.WriteHeader(&tar.Header{
		Name:     "link/escaped",
		Typeflag: tar.TypeReg,
		Mode:     0600,
		Size:     int64(len(content)),
	}); err != nil {
		t.Fatalf("WriteHeader(file) error = %v", err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	root := t.TempDir()
	installDir := filepath.Join(root, "install")
	if err := os.Mkdir(installDir, 0700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "outside"), 0700); err != nil {
		t.Fatalf("Mkdir(outside) error = %v", err)
	}
	localHost := &Host{Host: host.NewLocal(), Config: config.New()}
	err := untar(localHost, installDir, bytes.NewReader(archive.Bytes()), int64(archive.Len()), nil)
	if err == nil {
		t.Fatal("untar() succeeded for a symlink target outside the install directory")
	}
	if _, err := os.Stat(filepath.Join(root, "outside", "escaped")); !os.IsNotExist(err) {
		t.Fatalf("archive wrote outside install directory, stat error = %v", err)
	}
}

func TestUntarRejectsWritesThroughPreexistingSymlink(t *testing.T) {
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	if err := writer.WriteHeader(&tar.Header{
		Name:     "sub/link",
		Typeflag: tar.TypeSymlink,
		Linkname: "../target",
		Mode:     0777,
	}); err != nil {
		t.Fatalf("WriteHeader(symlink) error = %v", err)
	}
	content := []byte("unexpected")
	if err := writer.WriteHeader(&tar.Header{
		Name:     "link/escaped",
		Typeflag: tar.TypeReg,
		Mode:     0600,
		Size:     int64(len(content)),
	}); err != nil {
		t.Fatalf("WriteHeader(file) error = %v", err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	root := t.TempDir()
	installDir := filepath.Join(root, "install")
	outsideDir := filepath.Join(root, "outside")
	if err := os.Mkdir(installDir, 0700); err != nil {
		t.Fatalf("Mkdir(install) error = %v", err)
	}
	if err := os.Mkdir(outsideDir, 0700); err != nil {
		t.Fatalf("Mkdir(outside) error = %v", err)
	}
	if err := os.Symlink("../outside", filepath.Join(installDir, "target")); err != nil {
		t.Fatalf("Symlink(target) error = %v", err)
	}
	localHost := &Host{Host: host.NewLocal(), Config: config.New()}
	err := untar(localHost, installDir, bytes.NewReader(archive.Bytes()), int64(archive.Len()), nil)
	if err == nil {
		t.Fatal("untar() succeeded when an archive entry traversed a pre-existing symlink")
	}
	if _, err := os.Stat(filepath.Join(outsideDir, "escaped")); !os.IsNotExist(err) {
		t.Fatalf("archive wrote outside install directory, stat error = %v", err)
	}
}

func TestUntarAllowsWritesThroughInternalSymlink(t *testing.T) {
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	if err := writer.WriteHeader(&tar.Header{
		Name:     "sub/link",
		Typeflag: tar.TypeSymlink,
		Linkname: "../target",
		Mode:     0777,
	}); err != nil {
		t.Fatalf("WriteHeader(symlink) error = %v", err)
	}
	content := []byte("inside extraction root")
	if err := writer.WriteHeader(&tar.Header{
		Name:     "sub/link/file.txt",
		Typeflag: tar.TypeReg,
		Mode:     0600,
		Size:     int64(len(content)),
	}); err != nil {
		t.Fatalf("WriteHeader(file) error = %v", err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	installDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(installDir, "target"), 0700); err != nil {
		t.Fatalf("Mkdir(target) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(installDir, "sub"), 0700); err != nil {
		t.Fatalf("Mkdir(sub) error = %v", err)
	}
	localHost := &Host{Host: host.NewLocal(), Config: config.New()}
	if err := untar(localHost, installDir, bytes.NewReader(archive.Bytes()), int64(archive.Len()), nil); err != nil {
		t.Fatalf("untar() rejected an internal symlink traversal: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(installDir, "target", "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile(target file) error = %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("extracted content = %q, want %q", got, content)
	}
}

func TestUnzipRejectsPathTraversal(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create("../escaped")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := entry.Write([]byte("unexpected")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	root := t.TempDir()
	installDir := filepath.Join(root, "install")
	if err := os.Mkdir(installDir, 0700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	localHost := &Host{Host: host.NewLocal(), Config: config.New()}
	if err := unzip(localHost, installDir, bytes.NewReader(archive.Bytes()), int64(archive.Len()), nil); err == nil {
		t.Fatal("unzip() succeeded for path traversal entry")
	}
	if _, err := os.Stat(filepath.Join(root, "escaped")); !os.IsNotExist(err) {
		t.Fatalf("archive wrote outside install directory, stat error = %v", err)
	}
}
