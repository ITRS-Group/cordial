package geneos

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
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

type zipTestEntry struct {
	name string
	data []byte
	mode fs.FileMode
}

func buildZip(t *testing.T, entries ...zipTestEntry) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.mode != 0 {
			hdr.SetMode(e.mode)
		}
		w, err := writer.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(e.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

// streamOf hides everything but Read, so that the archive has to be
// copied to a file
type streamOf struct {
	io.Reader
	onRead func(reads int)
	reads  int
}

func (s *streamOf) Read(p []byte) (int, error) {
	s.reads++
	if s.onRead != nil {
		s.onRead(s.reads)
	}
	return s.Reader.Read(p)
}

func spoolFiles(t *testing.T, dir string) (found []string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), "unzip-") {
			found = append(found, e.Name())
		}
	}
	return
}

func TestUnzipSpoolsStreamToTemporaryFile(t *testing.T) {
	archive := buildZip(t,
		zipTestEntry{name: "top/", mode: fs.ModeDir | 0o755},
		zipTestEntry{name: "top/a.txt", data: []byte("alpha")},
		zipTestEntry{name: "top/sub/b.txt", data: []byte("bravo")},
	)
	strip := func(name string) string { _, after, _ := strings.Cut(name, "/"); return after }

	for name, h := range map[string]*Host{
		"local host":  {Host: host.NewLocal(), Config: config.New()},
		"remote host": {Host: notLocalHost{host.NewLocal()}, Config: config.New()},
	} {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			t.Setenv("TMP", tmp)
			t.Setenv("TEMP", tmp)
			installDir := t.TempDir()

			var spooled int
			// the spool file exists while the archive is read
			src := &streamOf{Reader: bytes.NewReader(archive), onRead: func(reads int) {
				if reads == 2 {
					spooled = len(spoolFiles(t, installDir)) + len(spoolFiles(t, tmp))
				}
			}}
			if err := unzip(h, installDir, src, -1, strip); err != nil {
				t.Fatalf("unzip() error = %v", err)
			}
			if spooled != 1 {
				t.Errorf("%d temporary files while spooling the archive, want 1", spooled)
			}

			for file, want := range map[string]string{"a.txt": "alpha", filepath.Join("sub", "b.txt"): "bravo"} {
				if got, err := os.ReadFile(filepath.Join(installDir, file)); err != nil || string(got) != want {
					t.Errorf("%s = %q, %v, want %q", file, got, err, want)
				}
			}
			if left := append(spoolFiles(t, installDir), spoolFiles(t, tmp)...); len(left) != 0 {
				t.Errorf("temporary files left behind: %v", left)
			}
		})
	}
}

func TestUnzipLargeArchive(t *testing.T) {
	// incompressible data, so the archive is as large as its contents
	rng := rand.New(rand.NewSource(1))
	sums := map[string][32]byte{}
	var entries []zipTestEntry
	for i := range 8 {
		data := make([]byte, 3<<20)
		rng.Read(data)
		name := filepath.ToSlash(filepath.Join("dir", string(rune('a'+i))+".bin"))
		sums[name] = sha256.Sum256(data)
		entries = append(entries, zipTestEntry{name: name, data: data})
	}
	archive := buildZip(t, entries...)

	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	installDir := t.TempDir()
	h := &Host{Host: notLocalHost{host.NewLocal()}, Config: config.New()}
	if err := unzip(h, installDir, &streamOf{Reader: bytes.NewReader(archive)}, int64(len(archive)), nil); err != nil {
		t.Fatal(err)
	}
	for name, want := range sums {
		got, err := os.ReadFile(filepath.Join(installDir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if sha256.Sum256(got) != want {
			t.Errorf("%s was not extracted correctly", name)
		}
	}
	if left := spoolFiles(t, tmp); len(left) != 0 {
		t.Errorf("temporary files left behind: %v", left)
	}
}

func TestUnzipOfFileUsesItDirectly(t *testing.T) {
	archive := buildZip(t, zipTestEntry{name: "a.txt", data: []byte("alpha")})
	file := filepath.Join(t.TempDir(), "archive.zip")
	if err := os.WriteFile(file, archive, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	installDir := t.TempDir()
	h := &Host{Host: host.NewLocal(), Config: config.New()}
	// the limit is only for archives that are copied
	if err := unzipLimit(h, installDir, f, -1, nil, 1); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(installDir, "a.txt")); err != nil || string(got) != "alpha" {
		t.Errorf("a.txt = %q, %v", got, err)
	}
}

func TestUnzipArchiveSizeLimit(t *testing.T) {
	data := make([]byte, 100_000)
	rand.New(rand.NewSource(2)).Read(data)
	archive := buildZip(t, zipTestEntry{name: "a.bin", data: data})
	h := &Host{Host: host.NewLocal(), Config: config.New()}

	t.Run("unknown size", func(t *testing.T) {
		installDir := t.TempDir()
		err := unzipLimit(h, installDir, &streamOf{Reader: bytes.NewReader(archive)}, -1, nil, int64(len(archive)-1))
		if err == nil || !strings.Contains(err.Error(), "larger than the limit") {
			t.Fatalf("unzipLimit() error = %v, want an error for the limit", err)
		}
		if left := spoolFiles(t, installDir); len(left) != 0 {
			t.Errorf("temporary files left behind: %v", left)
		}
		if _, err := os.Stat(filepath.Join(installDir, "a.bin")); err == nil {
			t.Error("a file was extracted from an archive over the limit")
		}
	})

	t.Run("known size is rejected before reading", func(t *testing.T) {
		src := &streamOf{Reader: bytes.NewReader(archive), onRead: func(int) { t.Error("the archive was read") }}
		if err := unzipLimit(h, t.TempDir(), src, int64(len(archive)), nil, 1000); err == nil {
			t.Fatal("unzipLimit() succeeded for an archive over the limit")
		}
	})

	t.Run("at the limit and no limit", func(t *testing.T) {
		for _, limit := range []int64{int64(len(archive)), 0} {
			installDir := t.TempDir()
			if err := unzipLimit(h, installDir, &streamOf{Reader: bytes.NewReader(archive)}, -1, nil, limit); err != nil {
				t.Errorf("unzipLimit(%d) error = %v", limit, err)
			}
			if got, err := os.ReadFile(filepath.Join(installDir, "a.bin")); err != nil || !bytes.Equal(got, data) {
				t.Errorf("a.bin was not extracted with limit %d: %v", limit, err)
			}
		}
	})
}

// TestUnzipProtectsDestination checks that the entries of a zip archive
// that is spooled cannot be written outside of the destination
func TestUnzipProtectsDestination(t *testing.T) {
	for name, entry := range map[string]zipTestEntry{
		"parent":          {name: "../escaped", data: []byte("x")},
		"nested parent":   {name: "a/../../escaped", data: []byte("x")},
		"absolute":        {name: "/escaped", data: []byte("x")},
		"parent dir":      {name: "../", mode: fs.ModeDir | 0o755},
		"link to parent":  {name: "link", data: []byte("../outside"), mode: fs.ModeSymlink | 0o777},
		"link to nested":  {name: "sub/link", data: []byte("../../outside"), mode: fs.ModeSymlink | 0o777},
		"link to absolut": {name: "link", data: []byte("/etc"), mode: fs.ModeSymlink | 0o777},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			installDir := filepath.Join(root, "install")
			if err := os.Mkdir(installDir, 0o700); err != nil {
				t.Fatal(err)
			}
			h := &Host{Host: host.NewLocal(), Config: config.New()}
			archive := buildZip(t, entry)
			if err := unzip(h, installDir, &streamOf{Reader: bytes.NewReader(archive)}, -1, nil); err == nil {
				t.Fatal("unzip() succeeded")
			}
			if left := spoolFiles(t, installDir); len(left) != 0 {
				t.Errorf("temporary files left behind: %v", left)
			}
			entries, _ := os.ReadDir(root)
			if len(entries) != 1 {
				t.Errorf("files outside of the install directory: %v", entries)
			}
			if entries, _ := os.ReadDir(installDir); len(entries) != 0 {
				t.Errorf("files in the install directory: %v", entries)
			}
		})
	}

	t.Run("link inside is not created", func(t *testing.T) {
		installDir := t.TempDir()
		h := &Host{Host: host.NewLocal(), Config: config.New()}
		archive := buildZip(t,
			zipTestEntry{name: "target.txt", data: []byte("t")},
			zipTestEntry{name: "link", data: []byte("target.txt"), mode: fs.ModeSymlink | 0o777},
		)
		if err := unzip(h, installDir, &streamOf{Reader: bytes.NewReader(archive)}, -1, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(filepath.Join(installDir, "link")); err == nil {
			t.Error("a symbolic link entry was extracted")
		}
		if _, err := os.Stat(filepath.Join(installDir, "target.txt")); err != nil {
			t.Error(err)
		}
	})
}
