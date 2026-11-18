package restore_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsnet/compress/bzip2"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/san"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/restore"
)

// newTestRoot sets up a temporary Geneos directory as the local
// installation for the test
func newTestRoot(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	key := cordial.ExecutableName()
	previousRoot := config.Get[string](config.Global(), key)
	previousLocal, previousAll, previousHostname := geneos.LOCAL, geneos.ALL, geneos.LOCALHOST
	previousHandler := logger.LogHandler
	t.Cleanup(func() {
		config.Set(config.Global(), key, previousRoot)
		geneos.LOCAL, geneos.ALL, geneos.LOCALHOST = previousLocal, previousAll, previousHostname
		logger.LogHandler = previousHandler
	})
	logger.LogHandler = slog.NewTextHandler(io.Discard, nil)
	config.Set(config.Global(), key, root)
	geneos.LOCALHOST = "localhost"
	geneos.LOCAL = nil
	geneos.LOCAL = geneos.NewHost(geneos.LOCALHOST)
	geneos.ALL = geneos.NewHost(geneos.ALLHOSTS)
	return
}

type tarEntry struct {
	name string
	body string // directories end in "/"
}

// makeTar returns a tar archive of the entries
func makeTar(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: 0o644, Size: int64(len(e.body)), Typeflag: tar.TypeReg}
		if strings.HasSuffix(e.name, "/") {
			hdr.Typeflag, hdr.Mode, hdr.Size = tar.TypeDir, 0o755, 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func compress(t *testing.T, kind string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	switch kind {
	case "gzip":
		w := gzip.NewWriter(&buf)
		w.Write(data)
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	case "bzip2":
		w, err := bzip2.NewWriter(&buf, nil)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	default:
		return data
	}
	return buf.Bytes()
}

func writeArchive(t *testing.T, name string, kind string, entries []tarEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, compress(t, kind, makeTar(t, entries)), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

var licdEntries = []tarEntry{
	{"licd/licds/restored/", ""},
	{"licd/licds/restored/licd.json", `{"name":"restored"}`},
	{"licd/licds/restored/licd.txt", "some content"},
}

func TestRestoreCompressionByFileName(t *testing.T) {
	for _, tt := range []struct{ name, kind string }{
		{"backup.tar.gz", "gzip"},
		{"backup.tgz", "gzip"},
		{"backup.tar.bz2", "bzip2"},
		{"backup.tar", "none"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := newTestRoot(t)
			archive := writeArchive(t, tt.name, tt.kind, licdEntries)

			var progress bytes.Buffer
			err := restore.Restore(archive, restore.Host(geneos.LOCAL), restore.Component(&licd.Component), restore.ProgressTo(&progress))
			if err != nil {
				t.Fatalf("restore.Restore() error = %v", err)
			}
			if data, err := os.ReadFile(filepath.Join(root, "licd", "licds", "restored", "licd.txt")); err != nil || string(data) != "some content" {
				t.Errorf("restored file = %q, %v\n%s", data, err, progress.String())
			}
		})
	}
}

func TestRestoreRequiresKnownCompression(t *testing.T) {
	newTestRoot(t)

	t.Run("unknown extension", func(t *testing.T) {
		archive := writeArchive(t, "backup.zip", "none", licdEntries)
		if err := restore.Restore(archive, restore.Host(geneos.LOCAL)); !errors.Is(err, geneos.ErrInvalidArgs) {
			t.Errorf("restore.Restore() of an unknown file type error = %v, want ErrInvalidArgs", err)
		}
	})

	t.Run("stdin without a type", func(t *testing.T) {
		if err := restore.Restore("-", restore.Host(geneos.LOCAL), restore.Compression("")); !errors.Is(err, geneos.ErrInvalidArgs) {
			t.Errorf("restore.Restore() from STDIN with no type error = %v, want ErrInvalidArgs", err)
		}
	})

	t.Run("unknown stdin type", func(t *testing.T) {
		if err := restore.Restore("-", restore.Host(geneos.LOCAL), restore.Compression("xz")); !errors.Is(err, geneos.ErrInvalidArgs) {
			t.Errorf("restore.Restore() from STDIN with an unknown type error = %v, want ErrInvalidArgs", err)
		}
	})

	t.Run("stdin", func(t *testing.T) {
		for _, kind := range []string{"gzip", "bz2", "bzip2"} {
			root := newTestRoot(t)
			compression := map[string]string{"gzip": "gzip", "bz2": "bzip2", "bzip2": "bzip2"}[kind]
			archive := writeArchive(t, "stdin", compression, licdEntries)
			stdin, err := os.Open(archive)
			if err != nil {
				t.Fatal(err)
			}
			previous := os.Stdin
			os.Stdin = stdin
			err = restore.Restore("-", restore.Host(geneos.LOCAL), restore.Component(&licd.Component), restore.Compression(kind))
			os.Stdin = previous
			stdin.Close()
			if err != nil {
				t.Fatalf("restore.Restore() from STDIN with %q error = %v", kind, err)
			}
			if _, err := os.Stat(filepath.Join(root, "licd", "licds", "restored", "licd.txt")); err != nil {
				t.Errorf("STDIN %q: file was not restored: %v", kind, err)
			}
		}
	})
}

// TestRestoreFirstEntryWithNoSize checks that the files of an instance
// are restored when the first entry for it, a directory, has no size
func TestRestoreFirstEntryWithNoSize(t *testing.T) {
	root := newTestRoot(t)
	archive := writeArchive(t, "backup.tar.gz", "gzip", []tarEntry{
		{"licd/licds/zero/", ""},
		{"licd/licds/zero/empty.txt", ""},
		{"licd/licds/zero/licd.json", `{"name":"zero"}`},
		{"licd/licds/zero/after.txt", "after the empty file"},
	})

	var progress bytes.Buffer
	if err := restore.Restore(archive, restore.Host(geneos.LOCAL), restore.Component(&licd.Component), restore.ProgressTo(&progress)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"empty.txt", "licd.json", "after.txt"} {
		if _, err := os.Stat(filepath.Join(root, "licd", "licds", "zero", name)); err != nil {
			t.Errorf("%s was not restored: %v", name, err)
		}
	}
	if !strings.Contains(progress.String(), "licd zero 4 files") {
		t.Errorf("progress = %q, want 4 files restored", progress.String())
	}
}

func TestRestoreExistingAndList(t *testing.T) {
	root := newTestRoot(t)
	existing := filepath.Join(root, "licd", "licds", "restored")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existing, "licd.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := writeArchive(t, "backup.tar.gz", "gzip", licdEntries)

	var progress bytes.Buffer
	if err := restore.Restore(archive, restore.Host(geneos.LOCAL), restore.Component(&licd.Component), restore.ProgressTo(&progress)); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(existing, "licd.txt")); string(data) != "original" {
		t.Errorf("an existing instance was overwritten: %q", data)
	}
	if !strings.Contains(progress.String(), "already exists, skipping") {
		t.Errorf("progress = %q, want the existing instance to be skipped", progress.String())
	}

	// listing does not restore anything
	list := writeArchive(t, "list.tar.gz", "gzip", []tarEntry{
		{"licd/licds/listed/licd.json", `{"name":"listed"}`},
		{"licd/licds/listed/licd.txt", "listed content"},
	})
	progress.Reset()
	if err := restore.Restore(list, restore.Host(geneos.LOCAL), restore.List(true), restore.ProgressTo(&progress)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "licd", "licds", "listed")); !os.IsNotExist(err) {
		t.Errorf("listing an archive created an instance: %v", err)
	}
	if !strings.Contains(progress.String(), "licd listed 2 files") {
		t.Errorf("list output = %q", progress.String())
	}
}

// TestRestoreSameNameDifferentTypes checks that instances of different
// types with the same name are both restored and rebuilt
func TestRestoreSameNameDifferentTypes(t *testing.T) {
	root := newTestRoot(t)
	archive := writeArchive(t, "backup.tar.gz", "gzip", []tarEntry{
		{"gateway/gateways/prod/gateway.json", `{"name":"prod","config":{"rebuild":"always"}}`},
		{"netprobe/sans/prod/san.json", `{"name":"prod","config":{"rebuild":"always"}}`},
	})

	var progress bytes.Buffer
	if err := restore.Restore(archive, restore.Host(geneos.LOCAL), restore.ProgressTo(&progress)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gateway/gateways/prod/gateway.json", "netprobe/sans/prod/san.json"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Errorf("%s was not restored: %v", name, err)
		}
	}
	for _, want := range []string{"gateway prod 1 files", "san prod 1 files"} {
		if !strings.Contains(progress.String(), want) {
			t.Errorf("progress = %q, want %q", progress.String(), want)
		}
	}
	for _, want := range []string{"gateway prod config rebuild complete", "san prod config rebuild complete"} {
		if !strings.Contains(progress.String(), want) {
			t.Errorf("progress = %q, want %q: each type must be rebuilt", progress.String(), want)
		}
	}
}
