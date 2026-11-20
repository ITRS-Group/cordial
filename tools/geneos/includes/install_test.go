package includes

import (
	"archive/tar"
	"compress/gzip"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/itrs-group/cordial/pkg/config"
	holidaycontent "github.com/itrs-group/cordial/pkg/holidays"
	includecontent "github.com/itrs-group/cordial/pkg/includes"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
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
			link := filepath.Join(includesDir, "itrs.layer0.foundation.xml")
			if err := os.Symlink("previous/itrs.layer0.foundation.xml", link); err != nil {
				t.Fatal(err)
			}
			if err := Install(h, "", true, update); err != nil {
				t.Fatal(err)
			}
			want := "previous/itrs.layer0.foundation.xml"
			if update {
				want = filepath.Join(embeddedIncludeVersion(), "itrs.layer0.foundation.xml")
			}
			if target, err := os.Readlink(link); err != nil || target != want {
				t.Fatalf("include link = %q, %v; want %q", target, err, want)
			}
		})
	}
}

func TestInstallPrefixSelection(t *testing.T) {
	archive := writeTestIncludeArchive(t,
		includeArchiveEntry{name: "geneos-includes-v1.2.3/", kind: tar.TypeDir},
		includeArchiveEntry{name: "geneos-includes-v1.2.3/holidays-v1.2.3/holidays.test.xml", contents: "<gateway/>"},
	)
	for _, test := range []struct {
		name, archive, prefix string
		wantError             bool
	}{
		{"inferred-archive", archive, "", false},
		{"matching-archive", archive, "holidays", false},
		{"mismatched-archive", archive, "itrs", true},
		{"not-embedded", "", "holidays", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newTestGatewayHost(t)
			err := installIncludes(h, test.archive, true, true, test.prefix)
			if test.wantError {
				if err == nil {
					t.Fatal("mismatched prefix accepted")
				}
				if _, err := os.Lstat(h.PathTo("gateway", "includes")); !os.IsNotExist(err) {
					t.Fatalf("rejected installation changed destination: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			link := h.PathTo("gateway", "includes", "holidays.test.xml")
			if target, err := os.Readlink(link); err != nil || target != "holidays-v1.2.3/holidays.test.xml" {
				t.Fatalf("installed link = %q, %v", target, err)
			}
		})
	}
}

func TestInstallReportsActions(t *testing.T) {
	capture := func(t *testing.T, run func() error) string {
		t.Helper()
		output, err := os.CreateTemp(t.TempDir(), "install-output-*")
		if err != nil {
			t.Fatal(err)
		}
		defer output.Close()
		previous := os.Stdout
		os.Stdout = output
		defer func() { os.Stdout = previous }()
		if err := run(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(output.Name())
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	t.Run("templates-and-links", func(t *testing.T) {
		h := newTestGatewayHost(t)
		configs, err := fs.Glob(includecontent.Files, "files/config.*.xml")
		if err != nil {
			t.Fatal(err)
		}
		for _, action := range []string{"Extracted", "Skipping existing"} {
			output := capture(t, func() error { return Install(h, "", true, false) })
			for _, conf := range configs {
				path := h.PathTo("gateway", geneos.SiteName(), geneos.SiteName()+"-"+filepath.Base(conf))
				want := action + " site config file " + path + "\n"
				if !strings.Contains(output, want) {
					t.Errorf("missing %q in output:\n%s", want, output)
				}
			}
			if action == "Skipping existing" {
				want := "Skipping existing include path " + h.PathTo("gateway", "includes", "itrs.layer0.foundation.xml") + "\n"
				if !strings.Contains(output, want) {
					t.Errorf("missing %q in output:\n%s", want, output)
				}
			}
		}
	})

	t.Run("legacy-config", func(t *testing.T) {
		h := newTestGatewayHost(t)
		site := geneos.SiteName()
		legacy := h.PathTo("gateway", "includes", "config")
		destination := h.PathTo("gateway", site)
		for _, directory := range []string{legacy, destination} {
			if err := os.MkdirAll(directory, 0755); err != nil {
				t.Fatal(err)
			}
		}
		for _, path := range []string{
			filepath.Join(legacy, "config.used.xml"),
			filepath.Join(legacy, "config.kept.xml"),
			filepath.Join(legacy, "config.unused.xml"),
			filepath.Join(destination, site+"-config.kept.xml"),
			filepath.Join(destination, site+".previous.xml"),
		} {
			if err := os.WriteFile(path, []byte("custom"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		writeTestGatewaySetup(t, h, `<gateway><includes>
<include><location>../../includes/config/config.used.xml</location></include>
<include><location>../../includes/config/config.kept.xml</location></include>
<include><location>../../`+site+`/`+site+`.previous.xml</location></include>
</includes></gateway>`)
		output := capture(t, func() error { return Install(h, "", true, false) })
		for _, want := range []string{
			"Copied legacy config file " + filepath.Join(legacy, "config.used.xml") + " -> " + filepath.Join(destination, site+"-config.used.xml"),
			"Skipping legacy config file " + filepath.Join(legacy, "config.kept.xml") + " -> " + filepath.Join(destination, site+"-config.kept.xml") + " (destination exists)",
			"Skipping unused legacy config file " + filepath.Join(legacy, "config.unused.xml"),
			"Copied legacy config file " + filepath.Join(destination, site+".previous.xml") + " -> " + filepath.Join(destination, site+"-config.previous.xml"),
		} {
			if !strings.Contains(output, want+"\n") {
				t.Errorf("missing %q in output:\n%s", want, output)
			}
		}
	})
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
	versionDir := h.PathTo("gateway", "includes", embeddedIncludeVersion())
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
			existing := filepath.Join(includesDir, "itrs.layer0.foundation.xml")
			if existingFile {
				if err := os.WriteFile(existing, []byte("local"), 0644); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink("previous/itrs.layer0.foundation.xml", existing); err != nil {
				t.Fatal(err)
			}
			if err := Install(h, "", true, false); err != nil {
				t.Fatal(err)
			}
			if existingFile {
				if data, err := os.ReadFile(existing); err != nil || string(data) != "local" {
					t.Fatalf("existing file = %q, %v", data, err)
				}
			} else if target, err := os.Readlink(existing); err != nil || target != "previous/itrs.layer0.foundation.xml" {
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
	versionDir := h.PathTo("gateway", "includes", embeddedIncludeVersion())
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
	configFiles, err := os.ReadDir(h.PathTo("gateway", "local"))
	if err != nil {
		t.Fatal(err)
	}
	if len(configFiles) == 0 {
		t.Fatal("Install() created no config template files")
	}
	for _, path := range []string{
		h.PathTo("gateway", "includes", "itrs.layer0.foundation.xml"),
		h.PathTo("gateway", "local", "local-config.layer0.authentication.xml"),
		h.PathTo("gateway", "local", "local-config.layer0.foundation.xml"),
		h.PathTo("gateway", "includes", "itrs.layer1.server-health.xml"),
		h.PathTo("gateway", "local", "local-config.layer1.server-health.xml"),
		h.PathTo("gateway", "includes", "itrs.layer2.programs-and-logs.xml"),
		h.PathTo("gateway", "local", "local-config.layer2.programs-and-logs.xml"),
		h.PathTo("gateway", "includes", "itrs.layer3.geneos.xml"),
		h.PathTo("gateway", "local", "local-config.layer3.geneos.xml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("installed layer file %q: %v", path, err)
		}
	}
	for _, path := range []string{
		h.PathTo("gateway", "includes", "itrs.level0.foundation.xml"),
		h.PathTo("gateway", "includes", "itrs.level1.server-health.xml"),
		h.PathTo("gateway", "includes", "itrs.level2.programs-and-logs.xml"),
		h.PathTo("gateway", "includes", "itrs.level3.geneos.xml"),
		h.PathTo("gateway", "local", "local-config.level3.geneos.xml"),
	} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Errorf("obsolete file %q was installed: %v", path, err)
		}
	}
}

func TestInstallOptionalHolidayCorpus(t *testing.T) {
	h := newTestGatewayHost(t)
	if err := Initialise(h); err != nil {
		t.Fatal(err)
	}
	dir := h.PathTo("gateway", "includes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "holidays") || strings.Contains(entry.Name(), ".holidays-") {
			t.Fatalf("initialization installed optional calendar %q", entry.Name())
		}
	}
	const version = "holidays-v1.2.3"
	const wrapper = "geneos-includes-v1.2.3/"
	files, err := fs.Glob(holidaycontent.Files, "files/holidays.*.xml")
	if err != nil || len(files) == 0 {
		t.Fatalf("holiday corpus: %v, %v", files, err)
	}
	archiveEntries := []includeArchiveEntry{
		{name: wrapper, kind: tar.TypeDir},
		{name: wrapper + version + "/", kind: tar.TypeDir},
		{name: wrapper + "docs/", kind: tar.TypeDir},
		{name: wrapper + "docs/README.md", contents: holidaycontent.Documentation},
	}
	for _, file := range files {
		data, err := holidaycontent.Files.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		archiveEntries = append(archiveEntries, includeArchiveEntry{
			name: wrapper + version + "/" + filepath.Base(file), contents: string(data),
		})
	}
	archive := writeTestIncludeArchive(t, archiveEntries...)
	publishedName := filepath.Join(filepath.Dir(archive), "geneos-holidays-include-v1.2.3.tar.gz")
	if err := os.Rename(archive, publishedName); err != nil {
		t.Fatal(err)
	}
	if err := Install(h, publishedName, false, false); err != nil {
		t.Fatal(err)
	}
	if err := Install(h, "", true, true); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		name := filepath.Base(file)
		want, err := holidaycontent.Files.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(dir, version, name)); err != nil || string(got) != string(want) {
			t.Errorf("installed holiday %q differs from corpus: %v", name, err)
		}
		if target, err := os.Readlink(filepath.Join(dir, name)); err != nil || target != version+"/"+name {
			t.Errorf("holiday link %q = %q, %v", name, target, err)
		}
	}
	if target, err := os.Readlink(filepath.Join(dir, "itrs.layer0.foundation.xml")); err != nil ||
		target != embeddedIncludeVersion()+"/itrs.layer0.foundation.xml" {
		t.Errorf("standard include link changed: %q, %v", target, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "docs")); !os.IsNotExist(err) {
		t.Errorf("archive documentation installed into live includes: %v", err)
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
		{h.PathTo("gateway", "local", "local-config.test.xml"), "template", 0664},
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

func TestInstallSourceVersions(t *testing.T) {
	for _, source := range []string{"itrs", "acme", "acme-ops"} {
		t.Run(source, func(t *testing.T) {
			h := newTestGatewayHost(t)
			previous := config.Get[string](config.Global(), "site")
			config.Set(config.Global(), "site", source)
			t.Cleanup(func() { config.Set(config.Global(), "site", previous) })
			version := source + "-v1.2.3-rc.1+build.7"
			root := "geneos-includes-v1.2.3-rc.1+build.7"
			archive := writeTestIncludeArchive(t,
				includeArchiveEntry{name: root + "/", kind: tar.TypeDir},
				includeArchiveEntry{name: root + "/" + version + "/" + source + ".test.xml", contents: "managed"},
				includeArchiveEntry{name: root + "/config/config.test.xml", contents: "template"},
			)
			if err := Install(h, archive, false, false); err != nil {
				t.Fatal(err)
			}
			link := h.PathTo("gateway", "includes", source+".test.xml")
			if target, err := os.Readlink(link); err != nil || target != version+"/"+source+".test.xml" {
				t.Fatalf("source link = %q, %v", target, err)
			}
			if data, err := os.ReadFile(link); err != nil || string(data) != "managed" {
				t.Fatalf("functional include = %q, %v", data, err)
			}
			template := h.PathTo("gateway", source, source+"-config.test.xml")
			if data, err := os.ReadFile(template); err != nil || string(data) != "template" {
				t.Fatalf("site template = %q, %v", data, err)
			}
		})
	}
}

func TestInstallPreservesPreviousSiteConfig(t *testing.T) {
	h := newTestGatewayHost(t)
	site := geneos.SiteName()
	directory := h.PathTo("gateway", site)
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(directory, site+".test.xml")
	if err := os.WriteFile(legacy, []byte("custom"), 0640); err != nil {
		t.Fatal(err)
	}
	location := "../../" + site + "/" + site + ".test.xml"
	writeTestGatewaySetup(t, h, "<gateway><includes><include><location>"+location+"</location></include></includes></gateway>")
	if err := os.WriteFile(h.PathTo("gateway", "gateways", "test", "gateway.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	source := fstest.MapFS{
		"files/itrs.test.xml":   &fstest.MapFile{Data: []byte("include")},
		"files/config.test.xml": &fstest.MapFile{Data: []byte("template")},
	}
	if err := extractIncludes(h, source, "itrs-v1.2.3", true); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(directory, site+"-config.test.xml")); err != nil || string(data) != "custom" {
		t.Fatalf("migrated site config = %q, %v", data, err)
	}
	if err := Fix(h, false); err != nil {
		t.Fatal(err)
	}
	setup, err := os.ReadFile(h.PathTo("gateway", "gateways", "test", "gateway.setup.xml"))
	if err != nil || !strings.Contains(string(setup), "../../"+site+"/"+site+"-config.test.xml") {
		t.Fatalf("migrated site reference = %s, %v", setup, err)
	}
	if data, err := os.ReadFile(legacy); err != nil || string(data) != "custom" {
		t.Fatalf("legacy site config changed = %q, %v", data, err)
	}
}

func TestInstallMigratesLegacyConfig(t *testing.T) {
	for _, test := range []struct {
		site   string
		prefix string
	}{
		{"", "local"},
		{"Acme Site", "Acme_Site"},
	} {
		t.Run(test.prefix, func(t *testing.T) {
			previous := config.Get[string](config.Global(), "site")
			config.Set(config.Global(), "site", test.site)
			t.Cleanup(func() { config.Set(config.Global(), "site", previous) })

			h := newTestGatewayHost(t)
			legacy := h.PathTo("gateway", "includes", "config")
			target := h.PathTo("gateway", test.prefix)
			if err := os.MkdirAll(filepath.Join(legacy, "nested"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(target, 0755); err != nil {
				t.Fatal(err)
			}
			for name, contents := range map[string]string{
				"config.test.xml":   "custom",
				"config.kept.xml":   "legacy",
				"config.unused.xml": "unused",
				"nested/other.txt":  "nested",
				"nested/unused.txt": "unused",
			} {
				if err := os.WriteFile(filepath.Join(legacy, name), []byte(contents), 0640); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink("config.test.xml", filepath.Join(legacy, "config.link.xml")); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, test.prefix+"-config.kept.xml"), []byte("existing"), 0644); err != nil {
				t.Fatal(err)
			}
			var locations strings.Builder
			for _, name := range []string{"config.test.xml", "config.kept.xml", "config.link.xml", "nested/other.txt"} {
				locations.WriteString("<include><location>../../includes/config/" + name + "</location></include>")
			}
			writeTestGatewaySetup(t, h, "<gateway><includes>"+locations.String()+"</includes></gateway>")
			archive := writeTestIncludeArchive(t,
				includeArchiveEntry{name: "geneos-includes-v1.2.3/", kind: tar.TypeDir},
				includeArchiveEntry{name: "geneos-includes-v1.2.3/v1.2.3/itrs.test.xml", contents: "managed"},
				includeArchiveEntry{name: "geneos-includes-v1.2.3/config/config.test.xml", contents: "template"},
				includeArchiveEntry{name: "geneos-includes-v1.2.3/config/config.new.xml", contents: "new template"},
			)
			if err := Install(h, archive, false, false); err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]string{
				test.prefix + "-config.test.xml": "custom",
				test.prefix + "-config.link.xml": "custom",
				test.prefix + "-config.kept.xml": "existing",
				test.prefix + "-config.new.xml":  "new template",
				"nested/other.txt":               "nested",
			} {
				filename := filepath.Join(target, name)
				data, err := os.ReadFile(filename)
				if err != nil || string(data) != want {
					t.Errorf("file %q = %q, %v; want %q", filename, data, err, want)
				}
				info, err := os.Lstat(filename)
				if err != nil || !info.Mode().IsRegular() {
					t.Errorf("file %q is not regular: %v, %v", filename, info, err)
				}
			}
			for _, name := range []string{test.prefix + "-config.unused.xml", "nested/unused.txt"} {
				if _, err := os.Lstat(filepath.Join(target, name)); !os.IsNotExist(err) {
					t.Errorf("unused legacy file %q was copied: %v", name, err)
				}
			}
			if data, err := os.ReadFile(filepath.Join(legacy, "config.test.xml")); err != nil || string(data) != "custom" {
				t.Errorf("legacy config changed: %q, %v", data, err)
			}
			shared, err := getSharedIncludes(h, h.PathTo("gateway"))
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, file := range shared {
				if file.Path == filepath.Join(target, test.prefix+"-config.test.xml") {
					found = true
				}
			}
			if !found {
				t.Error("site configuration include not discovered")
			}
		})
	}
}

func TestInstallRejectsBrokenLegacyConfigSymlink(t *testing.T) {
	h := newTestGatewayHost(t)
	legacy := h.PathTo("gateway", "includes", "config")
	if err := os.MkdirAll(legacy, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing.xml", filepath.Join(legacy, "config.test.xml")); err != nil {
		t.Fatal(err)
	}
	writeTestGatewaySetup(t, h, "<gateway><includes><include><location>../../includes/config/config.test.xml</location></include></includes></gateway>")
	source := fstest.MapFS{
		"files/itrs.test.xml":   &fstest.MapFile{Data: []byte("include")},
		"files/config.test.xml": &fstest.MapFile{Data: []byte("template")},
	}
	if err := extractIncludes(h, source, "v1.2.3", true); err == nil {
		t.Fatal("broken legacy symlink was silently ignored")
	}
}

func TestStandaloneInstallerMigratesLegacyConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("standalone installer requires Unix shell tools")
	}
	root, source := t.TempDir(), t.TempDir()
	legacy := filepath.Join(root, "gateway", "includes", "config")
	for _, directory := range []string{
		filepath.Join(legacy, "nested"),
		filepath.Join(source, "v1.2.3"),
		filepath.Join(source, "config"),
	} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for name, contents := range map[string]string{
		filepath.Join(legacy, "config.test.xml"):            "custom",
		filepath.Join(legacy, "nested", "config.extra.xml"): "nested",
		filepath.Join(source, "v1.2.3", "itrs.test.xml"):    "managed",
		filepath.Join(source, "config", "config.test.xml"):  "template",
	} {
		if err := os.WriteFile(name, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("config.test.xml", filepath.Join(legacy, "config.link.xml")); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("sh", "../../../pkg/includes/scripts/install-includes.sh", root)
	command.Env = append(os.Environ(), "INCLUDES_SOURCE="+source)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("standalone install failed: %v\n%s", err, output)
	}
	for name, want := range map[string]string{
		"local-config.test.xml":         "custom",
		"local-config.link.xml":         "custom",
		"nested/local-config.extra.xml": "nested",
	} {
		filename := filepath.Join(root, "gateway", "local", name)
		data, err := os.ReadFile(filename)
		if err != nil || string(data) != want {
			t.Errorf("%s = %q, %v; want %q", name, data, err, want)
		}
		info, err := os.Lstat(filename)
		if err != nil || !info.Mode().IsRegular() {
			t.Errorf("%s is not regular: %v, %v", name, info, err)
		}
	}
	if data, err := os.ReadFile(filepath.Join(legacy, "config.test.xml")); err != nil || string(data) != "custom" {
		t.Errorf("legacy config changed: %q, %v", data, err)
	}
}

func TestStandaloneInstallerSourceVersions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("standalone installer requires Unix shell tools")
	}
	for _, source := range []string{"itrs", "acme-ops"} {
		t.Run(source, func(t *testing.T) {
			root, archive := t.TempDir(), t.TempDir()
			version := source + "-v1.2.3-rc-vendor+build.7"
			for _, directory := range []string{version, "config"} {
				if err := os.MkdirAll(filepath.Join(archive, directory), 0755); err != nil {
					t.Fatal(err)
				}
			}
			for name, content := range map[string]string{
				version + "/" + source + ".test.xml": "managed",
				"config/config.test.xml":             "template",
			} {
				if err := os.WriteFile(filepath.Join(archive, name), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			siteDir := filepath.Join(root, "gateway", "local")
			if err := os.MkdirAll(siteDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(siteDir, "local.test.xml"), []byte("custom"), 0644); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("sh", "../../../pkg/includes/scripts/install-includes.sh", root)
			command.Env = append(os.Environ(), "INCLUDES_SOURCE="+archive)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("standalone install failed: %v\n%s", err, output)
			}
			link := filepath.Join(root, "gateway", "includes", source+".test.xml")
			if target, err := os.Readlink(link); err != nil || target != version+"/"+source+".test.xml" {
				t.Fatalf("standalone source link = %q, %v", target, err)
			}
			template := filepath.Join(root, "gateway", "local", "local-config.test.xml")
			if data, err := os.ReadFile(template); err != nil || string(data) != "custom" {
				t.Fatalf("standalone site template = %q, %v", data, err)
			}
		})
	}
}

func TestStandaloneHolidayInstallerPreservesExistingFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("standalone installer requires Unix shell tools")
	}
	root, archive := t.TempDir(), t.TempDir()
	const version = "holidays-v1.2.3"
	parent := filepath.Join(root, "gateway", "includes")
	for _, directory := range []string{
		filepath.Join(archive, version),
		filepath.Join(parent, "itrs-v1.0.0"),
		filepath.Join(parent, "holidays-v1.0.0"),
	} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"holidays.new.xml", "holidays.old.xml", "holidays.custom.xml"} {
		if err := os.WriteFile(filepath.Join(archive, version, name), []byte("<gateway/>"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range map[string]string{
		"itrs-v1.0.0/itrs.test.xml":        "standard",
		"holidays-v1.0.0/holidays.old.xml": "old holiday",
		"holidays.custom.xml":              "custom holiday",
	} {
		if err := os.WriteFile(filepath.Join(parent, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	links := map[string]string{
		"itrs.test.xml":    "itrs-v1.0.0/itrs.test.xml",
		"holidays.old.xml": "holidays-v1.0.0/holidays.old.xml",
		"holidays.new.xml": "missing/holidays.new.xml",
		"config":           "missing-config",
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(parent, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(archive, version, "holidays.missing.xml"), []byte("<gateway/>"), 0644); err != nil {
		t.Fatal(err)
	}
	run := func() ([]byte, error) {
		command := exec.Command("sh", "../../../pkg/includes/scripts/install-includes.sh", root)
		command.Env = append(os.Environ(), "INCLUDES_SOURCE="+archive)
		return command.CombinedOutput()
	}
	if output, err := run(); err != nil {
		t.Fatalf("holiday shell installation failed: %v\n%s", err, output)
	}
	for name, target := range links {
		if got, err := os.Readlink(filepath.Join(parent, name)); err != nil || got != target {
			t.Errorf("existing %s link changed to %q: %v", name, got, err)
		}
	}
	if got, err := os.ReadFile(filepath.Join(parent, "holidays.custom.xml")); err != nil || string(got) != "custom holiday" {
		t.Errorf("custom holiday file changed: %q, %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(parent, "itrs.test.xml")); err != nil || string(got) != "standard" {
		t.Errorf("standard include changed: %q, %v", got, err)
	}
	link := filepath.Join(parent, "holidays.missing.xml")
	if got, err := os.Readlink(link); err != nil || got != version+"/holidays.missing.xml" {
		t.Errorf("missing holiday link = %q, %v", got, err)
	}
	if info, err := os.Stat(link); err != nil || info.Mode().Perm() != 0444 {
		t.Errorf("installed holiday permissions = %v, %v", info, err)
	}
	if _, err := os.Lstat(filepath.Join(root, "gateway", "local")); !os.IsNotExist(err) {
		t.Errorf("holiday installer touched site configuration: %v", err)
	}
	if output, err := run(); err == nil || !strings.Contains(string(output), "Version already installed") {
		t.Fatalf("reinstallation should refuse the existing version: %v\n%s", err, output)
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
	config := h.PathTo("gateway", "local", "local-config.test.xml")
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
		{name: "source-version-mismatch", entries: []includeArchiveEntry{{name: "geneos-includes-v1.2.3/itrs-v2.0.0/itrs.test.xml"}}},
		{name: "wrong-source-file", entries: []includeArchiveEntry{{name: "geneos-includes-v1.2.3/acme-v1.2.3/itrs.test.xml"}}},
		{name: "multiple-sources", entries: []includeArchiveEntry{valid, {name: "geneos-includes-v1.2.3/acme-v1.2.3/acme.test.xml"}}},
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
