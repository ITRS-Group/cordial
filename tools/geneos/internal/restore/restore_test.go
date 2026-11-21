package restore_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
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
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
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
	// instances are cached for the life of the process, even if they
	// cannot be loaded, and must not be found by a later test, or run,
	// with a different root
	t.Cleanup(func() {
		for _, ct := range append(geneos.RealComponents(), geneos.ParseComponent("profile")) {
			if ct == nil {
				continue
			}
			for _, name := range instance.AllInstanceNames(geneos.LOCAL, ct) {
				if i := ct.New(name); i != nil {
					_ = i.Unload()
				}
			}
		}
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
func makeTar(t *testing.T, entries []tarEntry, links ...map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, group := range links {
		for name, target := range group {
			if err := tw.WriteHeader(&tar.Header{Name: name, Linkname: target, Mode: 0o777, Typeflag: tar.TypeSymlink}); err != nil {
				t.Fatal(err)
			}
		}
	}
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

func writeArchive(t *testing.T, name string, kind string, entries []tarEntry, links ...map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, compress(t, kind, makeTar(t, entries, links...)), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

var licdEntries = []tarEntry{
	{"licd/licds/restored/", ""},
	{"licd/licds/restored/licd.json", `{"name":"restored"}`},
	{"licd/licds/restored/licd.txt", "some content"},
}

func TestRestoreGatewaySiteShared(t *testing.T) {
	previous := config.Get[string](config.Global(), "site")
	t.Cleanup(func() { config.Set(config.Global(), "site", previous) })
	for _, test := range []struct{ site, directory string }{
		{"", "local"},
		{"Acme Site", "Acme_Site"},
	} {
		t.Run(test.directory, func(t *testing.T) {
			root := newTestRoot(t)
			config.Set(config.Global(), "site", test.site)
			name := "gateway/" + test.directory + "/site.xml"
			archive := writeArchive(t, "backup.tar", "none", []tarEntry{{name, "site configuration"}})
			if err := restore.Restore(archive, restore.Host(geneos.LOCAL)); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("site file restored without shared option: %v", err)
			}
			if err := restore.Restore(archive, restore.Host(geneos.LOCAL), restore.Shared(true)); err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(filepath.Join(root, name)); err != nil || string(got) != "site configuration" {
				t.Errorf("restored site file = %q, %v", got, err)
			}
		})
	}
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

func TestRestoreSharedSymlinks(t *testing.T) {
	root := newTestRoot(t)
	const linkedFile = "gateway/includes/itrs-v0.0.1/itrs.custom.xml"
	const includeLink = "gateway/includes/itrs.custom.xml"
	const danglingLink = "gateway/includes/itrs.dangling.xml"
	const existingLink = "gateway/includes/itrs.existing.xml"
	if err := os.MkdirAll(filepath.Join(root, "gateway", "includes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("primary-missing.xml", filepath.Join(root, existingLink)); err != nil {
		t.Fatal(err)
	}
	archive := writeArchive(t, "backup.tar.gz", "gzip", []tarEntry{
		{linkedFile, "primary include contents"},
	}, map[string]string{
		includeLink:                          "itrs-v0.0.1/itrs.custom.xml",
		danglingLink:                         "itrs-v0.0.1/missing.xml",
		existingLink:                         "itrs-v0.0.1/itrs.custom.xml",
		"gateway/includes/itrs.absolute.xml": filepath.Join(root, linkedFile),
	})
	if err := restore.Restore(archive, restore.Host(geneos.LOCAL)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(root, includeLink)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("shared link restored without --shared: %v", err)
	}
	if err := restore.Restore(archive, restore.Host(geneos.LOCAL), restore.Shared(true)); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		includeLink:                          "itrs-v0.0.1/itrs.custom.xml",
		danglingLink:                         "itrs-v0.0.1/missing.xml",
		existingLink:                         "primary-missing.xml",
		"gateway/includes/itrs.absolute.xml": "itrs-v0.0.1/itrs.custom.xml",
	} {
		if got, err := os.Readlink(filepath.Join(root, name)); err != nil || got != want {
			t.Errorf("%s link = %q, %v; want %q", name, got, err, want)
		}
	}
	if got, err := os.ReadFile(filepath.Join(root, includeLink)); err != nil || string(got) != "primary include contents" {
		t.Errorf("restored include = %q, %v", got, err)
	}
}

func TestRestoreRejectsUnsafeSymlinks(t *testing.T) {
	for _, kind := range []string{"relative-escape", "absolute-escape", "indirect-escape", "write-through-link"} {
		t.Run(kind, func(t *testing.T) {
			root := newTestRoot(t)
			outside := t.TempDir()
			victim := filepath.Join(outside, "primary.xml")
			if err := os.WriteFile(victim, []byte("unchanged"), 0o664); err != nil {
				t.Fatal(err)
			}
			target := "../../../" + filepath.Base(outside) + "/primary.xml"
			var entries []tarEntry
			switch kind {
			case "absolute-escape":
				target = victim
			case "indirect-escape":
				dir := filepath.Join(root, "gateway", "includes")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(dir, "external")); err != nil {
					t.Fatal(err)
				}
				target = "external/primary.xml"
			case "write-through-link":
				target = "targets"
				entries = []tarEntry{{"gateway/includes/alias/primary.xml", "must not be written"}}
			}
			archive := writeArchive(t, "backup.tar", "none", entries, map[string]string{
				"gateway/includes/alias": target,
			})
			if err := restore.Restore(archive, restore.Host(geneos.LOCAL), restore.Shared(true)); err == nil {
				t.Fatal("unsafe symlink archive was accepted")
			}
			if got, err := os.ReadFile(victim); err != nil || string(got) != "unchanged" {
				t.Errorf("outside file changed: %q, %v", got, err)
			}
			if kind == "write-through-link" {
				if _, err := os.Stat(filepath.Join(root, "gateway", "includes", "targets", "primary.xml")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("file written through restored symlink: %v", err)
				}
			}
		})
	}
}

func TestRestoreRejectsSymlinkedTruststore(t *testing.T) {
	root := newTestRoot(t)
	dir := filepath.Join(root, "tls")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ca-bundle.pem"), []byte("existing bundle"), 0o664); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "primary.db")
	if err := os.WriteFile(target, []byte("primary truststore"), 0o664); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "ca-bundle.db")); err != nil {
		t.Fatal(err)
	}
	archive := writeArchive(t, "backup.tar", "none", []tarEntry{{"tls/ca-bundle.pem", "incoming bundle"}})
	if err := restore.Restore(archive, restore.Host(geneos.LOCAL), restore.Shared(true)); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("restore error = %v, want symlink rejection", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "primary truststore" {
		t.Errorf("symlinked truststore was changed: %q, %v", got, err)
	}
}

func TestRestoreBeforeRebuildFailureAndListing(t *testing.T) {
	for _, list := range []bool{false, true} {
		t.Run(fmt.Sprintf("list=%t", list), func(t *testing.T) {
			root := newTestRoot(t)
			archive := writeArchive(t, "backup.tar.gz", "gzip", licdEntries)
			wantErr := errors.New("cannot prepare components")
			called := false
			var progress bytes.Buffer
			err := restore.Restore(archive, restore.Host(geneos.LOCAL), restore.List(list), restore.ProgressTo(&progress),
				restore.BeforeRebuild(func() error {
					called = true
					if data, err := os.ReadFile(filepath.Join(root, "licd", "licds", "restored", "licd.txt")); err != nil || string(data) != "some content" {
						t.Errorf("callback must run after extraction: %q, %v", data, err)
					}
					return wantErr
				}),
			)
			if list {
				if err != nil || called {
					t.Errorf("listing: error = %v, callback called = %t", err, called)
				}
			} else if !errors.Is(err, wantErr) || !called {
				t.Errorf("restore: error = %v, callback called = %t; want callback error", err, called)
			}
			if strings.Contains(progress.String(), "config rebuild complete") {
				t.Errorf("instances rebuilt despite preparation error:\n%s", progress.String())
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
