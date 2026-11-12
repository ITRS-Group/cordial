package includes

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial"
)

func TestInstallEmbeddedLinkModes(t *testing.T) {
	for _, update := range []bool{false, true} {
		name := "preserve"
		if update {
			name = "update"
		}
		t.Run(name, func(t *testing.T) {
			h := newTestGatewayHost(t)
			includesDir := h.PathTo("gateway", "includes")
			if err := os.MkdirAll(includesDir, 0755); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(includesDir, "itrs.level0.foundation.xml")
			if err := os.Symlink("previous/itrs.level0.foundation.xml", link); err != nil {
				t.Fatal(err)
			}
			if err := Install(h, "", true, update); err != nil {
				t.Fatal(err)
			}
			want := "previous/itrs.level0.foundation.xml"
			if update {
				want = filepath.Join(cordial.VERSION, "itrs.level0.foundation.xml")
			}
			if target, err := os.Readlink(link); err != nil || target != want {
				t.Fatalf("include link = %q, %v; want %q", target, err, want)
			}
		})
	}
}

func TestIncludesInstallArgumentsAndAlias(t *testing.T) {
	for _, args := range [][]string{nil, {"includes.tar.gz"}} {
		if err := includesInstallCmd.Args(includesInstallCmd, args); err != nil {
			t.Fatalf("install rejected %v: %v", args, err)
		}
	}
	if err := includesInstallCmd.Args(includesInstallCmd, []string{"one", "two"}); err == nil {
		t.Fatal("install accepted multiple archives")
	}
	if !includesInstallCmd.HasAlias("extract") {
		t.Fatal("install has no extract alias")
	}
	command, args, err := includesCmd.Find([]string{"extract"})
	if err != nil || command != includesInstallCmd || len(args) != 0 {
		t.Fatalf("extract resolved to %v, %v, %v; want install", command, args, err)
	}
}

func TestInstallEmbeddedRejectsExistingVersionDirectoryWithoutForce(t *testing.T) {
	h := newTestGatewayHost(t)
	versionDir := h.PathTo("gateway", "includes", cordial.VERSION)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Install(h, "", false, false); err == nil {
		t.Fatal("Install() succeeded with an existing version directory and no force")
	}
}

func TestInstallEmbeddedCreatesMissingLinksAndPreservesExistingPaths(t *testing.T) {
	for _, existingFile := range []bool{false, true} {
		name := "symlink"
		if existingFile {
			name = "file"
		}
		t.Run(name, func(t *testing.T) {
			h := newTestGatewayHost(t)
			includesDir := h.PathTo("gateway", "includes")
			if err := os.MkdirAll(includesDir, 0755); err != nil {
				t.Fatal(err)
			}
			existing := filepath.Join(includesDir, "itrs.level0.foundation.xml")
			if existingFile {
				if err := os.WriteFile(existing, []byte("local"), 0644); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink("previous/itrs.level0.foundation.xml", existing); err != nil {
				t.Fatal(err)
			}
			if err := Install(h, "", true, false); err != nil {
				t.Fatal(err)
			}
			if existingFile {
				if data, err := os.ReadFile(existing); err != nil || string(data) != "local" {
					t.Fatalf("existing file = %q, %v", data, err)
				}
			} else if target, err := os.Readlink(existing); err != nil || target != "previous/itrs.level0.foundation.xml" {
				t.Fatalf("existing link = %q, %v", target, err)
			}
		})
	}
}

func TestInstallEmbeddedCreatesVersionedAndConfigFiles(t *testing.T) {
	h := newTestGatewayHost(t)
	if err := Install(h, "", false, false); err != nil {
		t.Fatal(err)
	}
	versionDir := h.PathTo("gateway", "includes", cordial.VERSION)
	versionFiles, err := os.ReadDir(versionDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(versionFiles) == 0 {
		t.Fatal("Install() created no versioned include files")
	}
	for _, entry := range versionFiles {
		info, err := entry.Info()
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0444 {
			t.Errorf("versioned file %q permissions = %04o, want 0444", entry.Name(), got)
		}
	}
	configFiles, err := os.ReadDir(h.PathTo("gateway", "includes", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if len(configFiles) == 0 {
		t.Fatal("Install() created no config template files")
	}
}

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
	root := "geneos-includes-" + version
	archive := writeTestIncludeArchive(t,
		includeArchiveEntry{name: root + "/", kind: tar.TypeDir},
		includeArchiveEntry{name: root + "/" + version + "/", kind: tar.TypeDir},
		includeArchiveEntry{name: root + "/config/", kind: tar.TypeDir},
		includeArchiveEntry{name: root + "/scripts/", kind: tar.TypeDir},
		includeArchiveEntry{name: root + "/docs/", kind: tar.TypeDir},
		includeArchiveEntry{name: root + "/" + version + "/itrs.test.xml", contents: "managed"},
		includeArchiveEntry{name: root + "/config/config.test.xml", contents: "template"},
		includeArchiveEntry{name: root + "/scripts/install-includes.sh", contents: "#!/bin/sh\\n"},
		includeArchiveEntry{name: root + "/docs/README.md", contents: "# Includes"},
		includeArchiveEntry{name: root + "/docs/README.pdf", contents: "PDF"},
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
	if _, err := os.Stat(h.PathTo("gateway", "includes", "scripts", "install-includes.sh")); !os.IsNotExist(err) {
		t.Fatalf("install script was unexpectedly installed into the Gateway include directory: %v", err)
	}
	if _, err := os.Stat(h.PathTo("gateway", "includes", "docs")); !os.IsNotExist(err) {
		t.Fatalf("archive documentation was unexpectedly installed into the Gateway include directory: %v", err)
	}
}

func TestInstallRejectsUnexpectedArchiveDocumentation(t *testing.T) {
	h := newTestGatewayHost(t)
	root := "geneos-includes-v1.2.3"
	archive := writeTestIncludeArchive(t,
		includeArchiveEntry{name: root + "/", kind: tar.TypeDir},
		includeArchiveEntry{name: root + "/v1.2.3/", kind: tar.TypeDir},
		includeArchiveEntry{name: root + "/docs/", kind: tar.TypeDir},
		includeArchiveEntry{name: root + "/v1.2.3/itrs.test.xml", contents: "managed"},
		includeArchiveEntry{name: root + "/docs/unexpected.txt", contents: "unexpected"},
	)
	if err := Install(h, archive, true, true); err == nil {
		t.Fatal("Install() accepted unexpected documentation in the archive")
	}
	if _, err := os.Stat(h.PathTo("gateway", "includes")); !os.IsNotExist(err) {
		t.Fatalf("invalid archive changed destination, stat error = %v", err)
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
					includeArchiveEntry{name: "geneos-includes-v1.2.3/", kind: tar.TypeDir},
					includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/", kind: tar.TypeDir},
					includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/itrs.test.xml", contents: "managed"},
					includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/itrs.new.xml", contents: "new"},
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
		includeArchiveEntry{name: "geneos-includes-v1.2.3/", kind: tar.TypeDir},
		includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/", kind: tar.TypeDir},
		includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/itrs.test.xml", contents: "first"},
		includeArchiveEntry{name: "geneos-includes-v1.2.3/config/config.test.xml", contents: "local"},
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
		includeArchiveEntry{name: "geneos-includes-v1.2.3/", kind: tar.TypeDir},
		includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/", kind: tar.TypeDir},
		includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/itrs.test.xml", contents: "second"},
		includeArchiveEntry{name: "geneos-includes-v1.2.3/config/config.test.xml", contents: "replacement"},
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
	valid := includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/itrs.test.xml", contents: "managed"}
	for _, test := range []struct {
		name    string
		entries []includeArchiveEntry
	}{
		{name: "empty"},
		{name: "unwrapped", entries: []includeArchiveEntry{{name: "v1.2.3/itrs.test.xml"}}},
		{name: "config-only", entries: []includeArchiveEntry{{name: "geneos-includes-v1.2.3/config/config.test.xml"}}},
		{name: "no-prefix", entries: []includeArchiveEntry{{name: "geneos-includes-v1.2.3/1.2.3/itrs.test.xml"}}},
		{name: "short-version", entries: []includeArchiveEntry{{name: "geneos-includes-v1.2.3/v1.2/itrs.test.xml"}}},
		{name: "leading-zero", entries: []includeArchiveEntry{{name: "geneos-includes-v1.2.3/v01.2.3/itrs.test.xml"}}},
		{name: "prerelease-leading-zero", entries: []includeArchiveEntry{{name: "geneos-includes-v1.2.3/v1.2.3-01/itrs.test.xml"}}},
		{name: "multiple-versions", entries: []includeArchiveEntry{valid, {name: "geneos-includes-v1.2.3/v2.0.0/itrs.other.xml"}}},
		{name: "traversal", entries: []includeArchiveEntry{valid, {name: "../outside.xml"}}},
		{name: "nested-traversal", entries: []includeArchiveEntry{valid, {name: "geneos-includes-v1.2.3/v1.2.3/../../outside.xml"}}},
		{name: "absolute", entries: []includeArchiveEntry{valid, {name: "/outside.xml"}}},
		{name: "backslash", entries: []includeArchiveEntry{valid, {name: `geneos-includes-v1.2.3/v1.2.3/itrs.\outside.xml`}}},
		{name: "duplicate", entries: []includeArchiveEntry{valid, valid}},
		{name: "symlink", entries: []includeArchiveEntry{valid, {name: "geneos-includes-v1.2.3/v1.2.3/itrs.link.xml", kind: tar.TypeSymlink}}},
		{name: "hardlink", entries: []includeArchiveEntry{valid, {name: "geneos-includes-v1.2.3/v1.2.3/itrs.link.xml", kind: tar.TypeLink}}},
		{name: "unexpected-file", entries: []includeArchiveEntry{valid, {name: "geneos-includes-v1.2.3/v1.2.3/README.md"}}},
		{name: "unexpected-script", entries: []includeArchiveEntry{valid, {name: "geneos-includes-v1.2.3/scripts/other.sh"}}},
		{name: "versioned-config", entries: []includeArchiveEntry{valid, {name: "geneos-includes-v1.2.3/v1.2.3/config.test.xml"}}},
		{name: "nested-file", entries: []includeArchiveEntry{valid, {name: "geneos-includes-v1.2.3/v1.2.3/sub/itrs.test.xml"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newTestGatewayHost(t)
			entries := test.entries
			if test.name != "empty" && test.name != "unwrapped" {
				entries = append([]includeArchiveEntry{
					{name: "geneos-includes-v1.2.3/", kind: tar.TypeDir},
				}, entries...)
			}
			archive := writeTestIncludeArchive(t, entries...)
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
			archive := writeTestIncludeArchive(t,
				includeArchiveEntry{name: "geneos-includes-v1.2.3/", kind: tar.TypeDir},
				includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/", kind: tar.TypeDir},
				includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/itrs.test.xml", contents: "managed"},
			)
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
