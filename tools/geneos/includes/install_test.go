package includes

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

type includeArchiveEntry struct {
	name     string
	contents string
	kind     byte
}

func writeTestIncludeArchive(t *testing.T, entries ...includeArchiveEntry) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "includes.tar.gz")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	compressed := gzip.NewWriter(file)
	archive := tar.NewWriter(compressed)
	for _, entry := range entries {
		kind := entry.kind
		if kind == 0 {
			kind = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Mode: 0644, Typeflag: kind}
		if kind == tar.TypeReg {
			header.Size = int64(len(entry.contents))
		}
		if kind == tar.TypeSymlink || kind == tar.TypeLink {
			header.Linkname = "../../outside.xml"
		}
		if err := archive.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if _, err := archive.Write([]byte(entry.contents)); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, close := range []func() error{archive.Close, compressed.Close, file.Close} {
		if err := close(); err != nil {
			t.Fatal(err)
		}
	}
	return filename
}

func TestInstallCreatesVersionedIncludesAndConfig(t *testing.T) {
	h := newTestGatewayHost(t)
	version := "v2.3.4-rc.1+build.7"
	archive := writeTestIncludeArchive(t,
		includeArchiveEntry{name: version + "/", kind: tar.TypeDir},
		includeArchiveEntry{name: "config/", kind: tar.TypeDir},
		includeArchiveEntry{name: version + "/itrs.test.xml", contents: "managed"},
		includeArchiveEntry{name: "config/config.test.xml", contents: "template"},
	)
	if err := Install(h, archive, false, false); err != nil {
		t.Fatal(err)
	}
	for _, file := range []struct {
		path     string
		contents string
		mode     os.FileMode
	}{
		{h.PathTo("gateway", "includes", version, "itrs.test.xml"), "managed", 0444},
		{h.PathTo("gateway", "includes", "config", "config.test.xml"), "template", 0664},
	} {
		data, err := os.ReadFile(file.path)
		if err != nil || string(data) != file.contents {
			t.Fatalf("ReadFile(%q) = %q, %v", file.path, data, err)
		}
		info, err := os.Stat(file.path)
		if err != nil || info.Mode().Perm()&^file.mode != 0 || info.Mode().Perm()&0600 != file.mode&0600 {
			t.Fatalf("Stat(%q) = %v, %v; want mode %04o", file.path, info, err, file.mode)
		}
	}
	link := h.PathTo("gateway", "includes", "itrs.test.xml")
	if target, err := os.Readlink(link); err != nil || target != version+"/itrs.test.xml" {
		t.Fatalf("include link = %q, %v", target, err)
	}
}

func TestInstallLinkModes(t *testing.T) {
	for _, existingFile := range []bool{false, true} {
		for _, options := range []struct {
			name   string
			force  bool
			update bool
		}{
			{name: "default"},
			{name: "force", force: true},
			{name: "update", update: true},
			{name: "update-force", force: true, update: true},
		} {
			name := "symlink/" + options.name
			if existingFile {
				name = "file/" + options.name
			}
			t.Run(name, func(t *testing.T) {
				h := newTestGatewayHost(t)
				includesDir := h.PathTo("gateway", "includes")
				if err := os.MkdirAll(includesDir, 0755); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join(includesDir, "itrs.test.xml")
				if existingFile {
					if err := os.WriteFile(link, []byte("local"), 0644); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Symlink("previous/itrs.test.xml", link); err != nil {
					t.Fatal(err)
				}
				archive := writeTestIncludeArchive(t,
					includeArchiveEntry{name: "v1.2.3/itrs.test.xml", contents: "managed"},
					includeArchiveEntry{name: "v1.2.3/itrs.new.xml", contents: "new"},
				)
				err := Install(h, archive, options.force, options.update)
				wantErr := options.update && !options.force
				if (err != nil) != wantErr {
					t.Fatalf("Install() error = %v, want error %v", err, wantErr)
				}
				if options.force && options.update {
					if target, err := os.Readlink(link); err != nil || target != "v1.2.3/itrs.test.xml" {
						t.Fatalf("updated link = %q, %v", target, err)
					}
				} else if existingFile {
					if data, err := os.ReadFile(link); err != nil || string(data) != "local" {
						t.Fatalf("existing file = %q, %v", data, err)
					}
				} else if target, err := os.Readlink(link); err != nil || target != "previous/itrs.test.xml" {
					t.Fatalf("existing link = %q, %v", target, err)
				}
				if !wantErr {
					if target, err := os.Readlink(filepath.Join(includesDir, "itrs.new.xml")); err != nil || target != "v1.2.3/itrs.new.xml" {
						t.Fatalf("new link = %q, %v", target, err)
					}
				}
			})
		}
	}
}

func TestInstallReinstallAndConfigPreservation(t *testing.T) {
	h := newTestGatewayHost(t)
	first := writeTestIncludeArchive(t,
		includeArchiveEntry{name: "v1.2.3/itrs.test.xml", contents: "first"},
		includeArchiveEntry{name: "config/config.test.xml", contents: "local"},
	)
	if err := Install(h, first, false, false); err != nil {
		t.Fatal(err)
	}
	if err := Install(h, first, false, false); err == nil {
		t.Fatal("reinstall without force succeeded")
	}
	if err := Install(h, first, true, false); err != nil {
		t.Fatal(err)
	}
	managed := h.PathTo("gateway", "includes", "v1.2.3", "itrs.test.xml")
	if backups, err := filepath.Glob(managed + ".*.bak"); err != nil || len(backups) != 0 {
		t.Fatalf("unchanged include backups = %v, %v", backups, err)
	}
	second := writeTestIncludeArchive(t,
		includeArchiveEntry{name: "v1.2.3/itrs.test.xml", contents: "second"},
		includeArchiveEntry{name: "config/config.test.xml", contents: "replacement"},
	)
	if err := Install(h, second, true, true); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(managed); err != nil || string(data) != "second" {
		t.Fatalf("reinstalled include = %q, %v", data, err)
	}
	backups, err := filepath.Glob(managed + ".*.bak")
	if err != nil || len(backups) != 1 {
		t.Fatalf("changed include backups = %v, %v", backups, err)
	}
	if data, err := os.ReadFile(backups[0]); err != nil || string(data) != "first" {
		t.Fatalf("include backup = %q, %v", data, err)
	}
	config := h.PathTo("gateway", "includes", "config", "config.test.xml")
	if data, err := os.ReadFile(config); err != nil || string(data) != "local" {
		t.Fatalf("preserved config = %q, %v", data, err)
	}
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing.xml", config); err != nil {
		t.Fatal(err)
	}
	if err := Install(h, second, true, true); err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(config); err != nil || target != "missing.xml" {
		t.Fatalf("preserved config symlink = %q, %v", target, err)
	}
}

func TestInstallRejectsInvalidArchiveEntries(t *testing.T) {
	valid := includeArchiveEntry{name: "v1.2.3/itrs.test.xml", contents: "managed"}
	for _, test := range []struct {
		name    string
		entries []includeArchiveEntry
	}{
		{name: "empty"},
		{name: "config-only", entries: []includeArchiveEntry{{name: "config/config.test.xml"}}},
		{name: "no-prefix", entries: []includeArchiveEntry{{name: "1.2.3/itrs.test.xml"}}},
		{name: "short-version", entries: []includeArchiveEntry{{name: "v1.2/itrs.test.xml"}}},
		{name: "leading-zero", entries: []includeArchiveEntry{{name: "v01.2.3/itrs.test.xml"}}},
		{name: "prerelease-leading-zero", entries: []includeArchiveEntry{{name: "v1.2.3-01/itrs.test.xml"}}},
		{name: "multiple-versions", entries: []includeArchiveEntry{valid, {name: "v2.0.0/itrs.other.xml"}}},
		{name: "traversal", entries: []includeArchiveEntry{valid, {name: "../outside.xml"}}},
		{name: "nested-traversal", entries: []includeArchiveEntry{valid, {name: "v1.2.3/../../outside.xml"}}},
		{name: "absolute", entries: []includeArchiveEntry{valid, {name: "/outside.xml"}}},
		{name: "backslash", entries: []includeArchiveEntry{valid, {name: `v1.2.3/itrs.\outside.xml`}}},
		{name: "duplicate", entries: []includeArchiveEntry{valid, valid}},
		{name: "symlink", entries: []includeArchiveEntry{valid, {name: "v1.2.3/itrs.link.xml", kind: tar.TypeSymlink}}},
		{name: "hardlink", entries: []includeArchiveEntry{valid, {name: "v1.2.3/itrs.link.xml", kind: tar.TypeLink}}},
		{name: "unexpected-file", entries: []includeArchiveEntry{valid, {name: "v1.2.3/README.md"}}},
		{name: "versioned-config", entries: []includeArchiveEntry{valid, {name: "v1.2.3/config.test.xml"}}},
		{name: "nested-file", entries: []includeArchiveEntry{valid, {name: "v1.2.3/sub/itrs.test.xml"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newTestGatewayHost(t)
			archive := writeTestIncludeArchive(t, test.entries...)
			if err := Install(h, archive, true, true); err == nil {
				t.Fatal("Install() accepted an invalid archive")
			}
			if _, err := os.Stat(h.PathTo("gateway", "includes")); !os.IsNotExist(err) {
				t.Fatalf("invalid archive changed destination, stat error = %v", err)
			}
		})
	}
}

func TestInstallRejectsCorruptArchive(t *testing.T) {
	for _, kind := range []string{"checksum", "truncated", "not-gzip"} {
		t.Run(kind, func(t *testing.T) {
			h := newTestGatewayHost(t)
			archive := writeTestIncludeArchive(t, includeArchiveEntry{name: "v1.2.3/itrs.test.xml", contents: "managed"})
			data, err := os.ReadFile(archive)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "checksum":
				data[len(data)-8] ^= 0xff
			case "truncated":
				data = data[:len(data)-10]
			case "not-gzip":
				data = []byte("not an archive")
			}
			if err := os.WriteFile(archive, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := Install(h, archive, false, false); err == nil {
				t.Fatal("Install() accepted a corrupt archive")
			}
			if _, err := os.Stat(h.PathTo("gateway", "includes")); !os.IsNotExist(err) {
				t.Fatalf("corrupt archive changed destination, stat error = %v", err)
			}
		})
	}
}
