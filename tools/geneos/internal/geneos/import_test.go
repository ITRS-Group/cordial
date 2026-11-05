package geneos

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

func newImportTestHost() *Host {
	cf := config.New()
	config.Set(cf, "name", "localhost")
	return &Host{Host: host.NewLocal(), Config: cf}
}

func TestImportFileSkipsUnchangedContent(t *testing.T) {
	h := newImportTestHost()
	destination := filepath.Join(t.TempDir(), "geneos.lic")
	if changed, err := importFile(h, destination, strings.NewReader("original"), 0o600); err != nil || !changed {
		t.Fatalf("initial import = %t, %v", changed, err)
	}
	oldTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(destination, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if changed, err := importFile(h, destination, strings.NewReader("original"), 0o600); err != nil || changed {
		t.Fatalf("unchanged import = %t, %v", changed, err)
	}
	info, err := os.Stat(destination)
	if err != nil || !info.ModTime().Equal(oldTime) {
		t.Fatalf("unchanged file modified: %v, %v", info, err)
	}
	backups, err := filepath.Glob(destination + ".*.old")
	if err != nil || len(backups) != 0 {
		t.Fatalf("unchanged import backups = %v, %v", backups, err)
	}
	if changed, err := importFile(h, destination, strings.NewReader("modified"), 0o600); err != nil || !changed {
		t.Fatalf("changed import = %t, %v", changed, err)
	}
	backups, err = filepath.Glob(destination + ".*.old")
	if err != nil || len(backups) != 1 {
		t.Fatalf("changed import backups = %v, %v", backups, err)
	}
	contents, err := os.ReadFile(backups[0])
	if err != nil || string(contents) != "original" {
		t.Fatalf("backup = %q, %v", contents, err)
	}
}

func TestReserveImportBackupSameSecond(t *testing.T) {
	h := newImportTestHost()
	destination := filepath.Join(t.TempDir(), "geneos.lic")
	now := time.Date(2026, 11, 5, 1, 2, 3, 0, time.UTC)
	for sequence := 0; sequence < 3; sequence++ {
		backup, err := reserveImportBackup(h, destination, now)
		if err != nil {
			t.Fatal(err)
		}
		want := destination + ".20261105010203.old"
		if sequence > 0 {
			want = fmt.Sprintf("%s.20261105010203.%d.old", destination, sequence)
		}
		if backup != want {
			t.Fatalf("backup = %q, want %q", backup, want)
		}
		if err := os.WriteFile(backup, []byte("preserved"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	backups, _ := filepath.Glob(destination + ".*.old")
	for _, backup := range backups {
		contents, err := os.ReadFile(backup)
		if err != nil || string(contents) != "preserved" {
			t.Fatalf("backup overwritten: %q, %v", contents, err)
		}
	}
}

func TestImportFileConcurrentWriters(t *testing.T) {
	h := newImportTestHost()
	destination := filepath.Join(t.TempDir(), "geneos.lic")
	const writers = 12
	var workers sync.WaitGroup
	for index := range writers {
		workers.Go(func() {
			if _, err := importFile(h, destination, strings.NewReader(fmt.Sprintf("version-%d", index)), 0o600); err != nil {
				t.Errorf("import: %v", err)
			}
		})
	}
	workers.Wait()
	backups, err := filepath.Glob(destination + ".*.old")
	if err != nil || len(backups) != writers-1 {
		t.Fatalf("backups = %v, %v", backups, err)
	}
	versions := make(map[string]bool)
	for _, filename := range append(backups, destination) {
		contents, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		versions[string(contents)] = true
	}
	if len(versions) != writers {
		t.Fatalf("preserved %d versions, want %d", len(versions), writers)
	}
}

func TestImportFileReadFailurePreservesDestination(t *testing.T) {
	h := newImportTestHost()
	directory := t.TempDir()
	destination := filepath.Join(directory, "geneos.lic")
	if err := os.WriteFile(destination, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("source read failed")
	reader := io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(failure))
	if _, err := importFile(h, destination, reader, 0o600); !errors.Is(err, failure) {
		t.Fatalf("import error = %v, want %v", err, failure)
	}
	contents, err := os.ReadFile(destination)
	if err != nil || string(contents) != "original" {
		t.Fatalf("original changed: %q, %v", contents, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files or backups left behind: %v, %v", entries, err)
	}
}

func TestImportDirectorySkipsUnchangedFiles(t *testing.T) {
	h := newImportTestHost()
	source, destination := t.TempDir(), t.TempDir()
	sourceFile := filepath.Join(source, "recipe.txt")
	if err := os.WriteFile(sourceFile, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := importDirectory(h, destination, source+"/"); err != nil {
			t.Fatal(err)
		}
	}
	backups, err := filepath.Glob(filepath.Join(destination, "*.old"))
	if err != nil || len(backups) != 0 {
		t.Fatalf("unchanged directory backups = %v, %v", backups, err)
	}
	if err := os.WriteFile(sourceFile, []byte("modified"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := importDirectory(h, destination, source+"/"); err != nil {
		t.Fatal(err)
	}
	backups, err = filepath.Glob(filepath.Join(destination, "*.old"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("changed directory backups = %v, %v", backups, err)
	}
	if err := importDirectory(h, destination, filepath.Join(source, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("directory walk error = %v, want missing source", err)
	}
}

type failedImportRenameHost struct {
	host.Host
	failure error
}

func (h *failedImportRenameHost) Rename(oldpath, newpath string) error {
	if strings.HasSuffix(oldpath, ".new") {
		return h.failure
	}
	return h.Host.Rename(oldpath, newpath)
}

func TestImportFileInstallFailureRestoresBackup(t *testing.T) {
	h := newImportTestHost()
	failure := errors.New("install rename failed")
	h.Host = &failedImportRenameHost{Host: h.Host, failure: failure}
	directory := t.TempDir()
	destination := filepath.Join(directory, "geneos.lic")
	if err := os.WriteFile(destination, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := importFile(h, destination, strings.NewReader("replacement"), 0o600); !errors.Is(err, failure) {
		t.Fatalf("import error = %v, want %v", err, failure)
	}
	contents, err := os.ReadFile(destination)
	if err != nil || string(contents) != "original" {
		t.Fatalf("restored file = %q, %v", contents, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files or backups left behind: %v, %v", entries, err)
	}
}

func TestImportFileRespectsDestinationLock(t *testing.T) {
	h := newImportTestHost()
	destination := filepath.Join(t.TempDir(), "geneos.lic")
	lockPath := destination + ".import.lock"
	if err := os.WriteFile(lockPath, []byte("another importer"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := importFile(h, destination, strings.NewReader("replacement"), 0o600); !errors.Is(err, os.ErrExist) {
		t.Fatalf("import error = %v, want destination lock conflict", err)
	}
	contents, err := os.ReadFile(lockPath)
	if err != nil || string(contents) != "another importer" {
		t.Fatalf("another importer's lock changed: %q, %v", contents, err)
	}
}

func TestImportSourceMissingReturnsError(t *testing.T) {
	h := newImportTestHost()
	root := t.TempDir()
	_, _, err := ImportSource(h, filepath.Join(root, "destination"), filepath.Join(root, "missing.lic"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("import error = %v, want missing source", err)
	}
}
