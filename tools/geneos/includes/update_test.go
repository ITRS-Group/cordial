package includes

import (
	"archive/tar"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
	holidaycontent "github.com/itrs-group/cordial/pkg/holidays"
	"github.com/itrs-group/cordial/pkg/host"
	includecontent "github.com/itrs-group/cordial/pkg/includes"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

type canonicalizingIncludeHost struct {
	host.Host
	workingDirectory string
	absCalls         int
}

func (h *canonicalizingIncludeHost) Abs(name string) (string, error) {
	h.absCalls++
	if !filepath.IsAbs(name) {
		name = filepath.Join(h.workingDirectory, name)
	}
	return filepath.EvalSymlinks(name)
}

func (h *canonicalizingIncludeHost) Getwd() (string, error) { return h.workingDirectory, nil }
func (h *canonicalizingIncludeHost) IsLocalhost() bool      { return false }

func TestResolveIncludePathPreservesRemoteLinkReferences(t *testing.T) {
	for _, kind := range []string{"absolute", "relative", "chain", "dangling"} {
		t.Run(kind, func(t *testing.T) {
			h := newTestGatewayHost(t)
			parent := h.PathTo("gateway", "includes")
			version := filepath.Join(parent, "v1.0.0")
			if err := os.MkdirAll(version, 0755); err != nil {
				t.Fatal(err)
			}
			if kind != "dangling" {
				if err := os.WriteFile(filepath.Join(version, "managed.xml"), nil, 0444); err != nil {
					t.Fatal(err)
				}
			}
			link := filepath.Join(parent, "managed.xml")
			if err := os.Symlink(filepath.Join("v1.0.0", "managed.xml"), link); err != nil {
				t.Fatal(err)
			}
			name := link
			if kind == "relative" {
				name = "managed.xml"
			}
			if kind == "chain" {
				name = filepath.Join(parent, "alias.xml")
				if err := os.Symlink("managed.xml", name); err != nil {
					t.Fatal(err)
				}
			}
			remote := &canonicalizingIncludeHost{Host: h.Host, workingDirectory: parent}
			h.Host = remote
			resolved, err := resolveIncludePath(h, name, "", link)
			if err != nil || resolved != link {
				t.Fatalf("resolveIncludePath() = %q, %v; want protected link %q", resolved, err, link)
			}
			if remote.absCalls != 0 {
				t.Fatal("resolver used canonicalizing host Abs before inspecting links")
			}
			writeTestGatewaySetup(t, h, `<gateway><includes><include><location>`+link+`</location></include></includes></gateway>`)
			newDirectory := filepath.Join(parent, "v2.0.0")
			if err := os.MkdirAll(newDirectory, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(newDirectory, "new.xml"), nil, 0444); err != nil {
				t.Fatal(err)
			}
			if err := Update(h, "v2.0.0", true); !errors.Is(err, ErrIncludeVersionLinked) {
				t.Fatalf("remote Update() error = %v; want referenced obsolete link error", err)
			}
			if target, err := os.Readlink(link); err != nil || target != filepath.Join("v1.0.0", "managed.xml") {
				t.Fatalf("blocked remote update changed old link: target %q, error %v", target, err)
			}
			if _, err := os.Lstat(filepath.Join(parent, "new.xml")); !os.IsNotExist(err) {
				t.Fatalf("blocked remote update created a new link: %v", err)
			}
			if remote.absCalls != 0 {
				t.Fatal("update preflight used canonicalizing host Abs")
			}
		})
	}
}

func TestIncludesCommandsDoNotExposeForce(t *testing.T) {
	for _, command := range includesCmd.Commands() {
		t.Run(command.Name(), func(t *testing.T) {
			if command.Flags().Lookup("force") != nil || command.Flags().ShorthandLookup("F") != nil {
				t.Fatal("includes command still exposes a force option")
			}
		})
	}
}

func TestIncludeVersionLinked(t *testing.T) {
	for _, kind := range []string{"relative", "absolute", "nested", "chain", "directory-alias", "dangling", "external", "other-version", "cycle"} {
		t.Run(kind, func(t *testing.T) {
			h := newTestGatewayHost(t)
			directory := h.PathTo("gateway", "includes", "v1.2.3")
			if err := os.MkdirAll(filepath.Join(directory, "nested"), 0755); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(directory, "itrs.test.xml")
			if err := os.WriteFile(file, []byte("include"), 0444); err != nil {
				t.Fatal(err)
			}
			parent := filepath.Dir(directory)
			link := filepath.Join(parent, "current.xml")
			target := "v1.2.3/itrs.test.xml"
			wantLinked := true
			switch kind {
			case "absolute":
				target = file
			case "nested":
				target = "v1.2.3/nested/missing.xml"
			case "chain":
				if err := os.Symlink(target, filepath.Join(parent, "alias.xml")); err != nil {
					t.Fatal(err)
				}
				target = "alias.xml"
			case "directory-alias":
				if err := os.Symlink(directory, filepath.Join(parent, "active")); err != nil {
					t.Fatal(err)
				}
				target = "active/itrs.test.xml"
			case "dangling":
				target = "v1.2.3/missing.xml"
			case "external":
				link = filepath.Join(t.TempDir(), "manual.xml")
				target = file
				wantLinked = false
			case "other-version":
				target = "v1.2.30/itrs.test.xml"
				wantLinked = false
			case "cycle":
				target = "current.xml"
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			protectingLink, err := includeVersionLink(h, directory)
			if kind == "cycle" {
				if err == nil {
					t.Fatal("cyclic link did not prevent a safety check")
				}
				return
			}
			if err != nil || (protectingLink != "") != wantLinked {
				t.Fatalf("includeVersionLink() = %q, %v; want linked %v", protectingLink, err, wantLinked)
			}
		})
	}
}

func TestUpdateRejectsVersionPathTraversal(t *testing.T) {
	h := newTestGatewayHost(t)
	root := h.PathTo()
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(outside, "outside.xml"), nil, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := Update(h, "../../outside", true)
	if err == nil || !strings.Contains(err.Error(), "invalid include version") {
		t.Fatalf("Update() error = %v, want invalid version error", err)
	}
	if _, err := os.Lstat(h.PathTo("gateway", "includes", "outside.xml")); !os.IsNotExist(err) {
		t.Fatalf("Update() created an include link outside the version directory, stat error = %v", err)
	}
}

func TestParseIncludeVersion(t *testing.T) {
	for _, test := range []struct {
		name, source, version string
	}{
		{"itrs-v1.2.3", "itrs", "v1.2.3"},
		{"holidays-v2026.01.02", "holidays", "v2026.01.02"},
		{"holidays-v2024.02.29", "holidays", "v2024.02.29"},
		{"holidays-v2026.02.29", "", ""},
		{"holidays-v2026.11.31", "", ""},
		{"holidays-v2026.13.20", "", ""},
		{"holidays-v2026.1.02", "", ""},
		{"holidays-v2026.01.2", "", ""},
		{"holidays-v0000.01.01", "", ""},
		{"itrs-v2026.01.02", "", ""},
		{"acme-ops-v1.2.3-rc.1+build.7", "acme-ops", "v1.2.3-rc.1+build.7"},
		{"acme-v1.2.3-rc-vendor", "acme", "v1.2.3-rc-vendor"},
		{"v1.2.3", "itrs", "v1.2.3"},
		{"1.2.3", "itrs", "1.2.3"},
		{"../itrs-v1.2.3", "", ""},
		{"acme/v1.2.3", "", ""},
		{"-v1.2.3", "", ""},
		{"acme-1.2.3", "", ""},
		{"acme-v1.2", "", ""},
		{"acme-v01.2.3", "", ""},
		{"acme-v1.2.3-01", "", ""},
		{"acme-v1.2.3+build/7", "", ""},
		{"config-v1.2.3", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, version, err := parseIncludeVersion(test.name)
			if (err != nil) != (test.source == "") || source != test.source || version != test.version {
				t.Fatalf("parseIncludeVersion(%q) = %q, %q, %v; want %q, %q", test.name, source, version, err, test.source, test.version)
			}
		})
	}
}

func TestUpdateKeepsOtherSources(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint(legacy), func(t *testing.T) {
			h := newTestGatewayHost(t)
			parent := h.PathTo("gateway", "includes")
			old := "itrs-v1.0.0"
			if legacy {
				old = "v1.0.0"
			}
			for version, name := range map[string]string{
				old:           "itrs.obsolete.xml",
				"itrs-v2.0.0": "itrs.new.xml",
				"acme-v1.0.0": "acme.used.xml",
			} {
				directory := filepath.Join(parent, version)
				if err := os.MkdirAll(directory, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(directory, name), nil, 0444); err != nil {
					t.Fatal(err)
				}
				if version != "itrs-v2.0.0" {
					if err := os.Symlink(filepath.Join(version, name), filepath.Join(parent, name)); err != nil {
						t.Fatal(err)
					}
				}
			}
			writeTestGatewaySetup(t, h, `<gateway><includes><include><location>../../includes/acme.used.xml</location></include></includes></gateway>`)
			if err := Update(h, "itrs-v2.0.0", true); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(filepath.Join(parent, "itrs.obsolete.xml")); !os.IsNotExist(err) {
				t.Fatalf("obsolete ITRS link retained: %v", err)
			}
			if target, err := os.Readlink(filepath.Join(parent, "acme.used.xml")); err != nil || target != "acme-v1.0.0/acme.used.xml" {
				t.Fatalf("other source link = %q, %v", target, err)
			}
			if err := Delete(h, "itrs-*"); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(parent, "acme-v1.0.0", "acme.used.xml")); err != nil {
				t.Fatalf("source-specific deletion affected another source: %v", err)
			}
		})
	}
}

func TestUpdateRejectsOtherSourceLinkCollision(t *testing.T) {
	for _, chain := range []bool{false, true} {
		t.Run(fmt.Sprint(chain), func(t *testing.T) {
			h := newTestGatewayHost(t)
			parent := h.PathTo("gateway", "includes")
			for _, version := range []string{"itrs-v2.0.0", "acme-v1.0.0"} {
				directory := filepath.Join(parent, version)
				if err := os.MkdirAll(directory, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(directory, "shared.xml"), nil, 0444); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(parent, "itrs-v2.0.0", "aaa-new.xml"), nil, 0444); err != nil {
				t.Fatal(err)
			}
			target := "acme-v1.0.0/shared.xml"
			if chain {
				if err := os.Symlink(target, filepath.Join(parent, "alias.xml")); err != nil {
					t.Fatal(err)
				}
				target = "alias.xml"
			}
			if err := os.Symlink(target, filepath.Join(parent, "shared.xml")); err != nil {
				t.Fatal(err)
			}
			err := Update(h, "itrs-v2.0.0", true)
			if err == nil || !strings.Contains(err.Error(), `belongs to source "acme"`) {
				t.Fatalf("collision error = %v", err)
			}
			if got, err := os.Readlink(filepath.Join(parent, "shared.xml")); err != nil || got != target {
				t.Fatalf("collision replaced link = %q, %v", got, err)
			}
			if _, err := os.Lstat(filepath.Join(parent, "aaa-new.xml")); !os.IsNotExist(err) {
				t.Fatalf("collision preflight created new link: %v", err)
			}
		})
	}
}

func TestUpdateDefaultUsesSourceVersion(t *testing.T) {
	h := newTestGatewayHost(t)
	if err := Update(h, "", true); err != nil {
		t.Fatal(err)
	}
	link := h.PathTo("gateway", "includes", "itrs.layer0.foundation.xml")
	if target, err := os.Readlink(link); err != nil || target != "itrs-"+includecontent.VERSION+"/itrs.layer0.foundation.xml" {
		t.Fatalf("default version link = %q, %v", target, err)
	}
}

func TestUpdatePrefixSelection(t *testing.T) {
	oldHostname := cmd.Hostname
	cmd.Hostname = "localhost"
	t.Cleanup(func() { cmd.Hostname = oldHostname })
	for _, test := range []struct {
		name, prefix, version, want string
	}{
		{"default", "", "", embeddedIncludeVersion()},
		{"holiday-latest", "holidays", "", "holidays-" + holidaycontent.VERSION},
		{"holiday-calendar", "holidays", "v2026.01.02", "holidays-v2026.01.02"},
		{"holiday-version", "holidays", "v2.0.0", "holidays-v2.0.0"},
		{"holiday-unprefixed-version", "holidays", "2.0.0", "holidays-v2.0.0"},
		{"itrs-version", "itrs", "v2.0.0", "itrs-v2.0.0"},
		{"legacy-version", "", "v2.0.0", "v2.0.0"},
		{"full-source", "", "holidays-v2.0.0", "holidays-v2.0.0"},
		{"matching-prefix", "holidays", "holidays-v2.0.0", "holidays-v2.0.0"},
		{"mismatch", "itrs", "holidays-v2.0.0", ""},
		{"unsafe-prefix", "../holidays", "", ""},
		{"reserved-prefix", "config", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newTestGatewayHost(t)
			writeTestGatewaySetup(t, h, "<gateway/>")
			parent := h.PathTo("gateway", "includes")
			versions := []string{embeddedIncludeVersion(), "holidays-" + holidaycontent.VERSION, "holidays-v2026.01.02", "itrs-v2.0.0", "holidays-v2.0.0", "v2.0.0"}
			for _, version := range versions {
				if err := os.MkdirAll(filepath.Join(parent, version), 0755); err != nil {
					t.Fatal(err)
				}
				source, _, _ := parseIncludeVersion(version)
				if err := os.WriteFile(filepath.Join(parent, version, source+".test.xml"), nil, 0444); err != nil {
					t.Fatal(err)
				}
			}
			prefixFlag := includesUpdateCmd.Flags().Lookup("prefix")
			oldPrefix, oldChanged := prefixFlag.Value.String(), prefixFlag.Changed
			oldVersion := includesUpdateCmdVersion
			t.Cleanup(func() {
				_ = prefixFlag.Value.Set(oldPrefix)
				prefixFlag.Changed = oldChanged
				includesUpdateCmdVersion = oldVersion
			})
			prefix := test.prefix
			if prefix == "" {
				prefix = "itrs"
			}
			if err := prefixFlag.Value.Set(prefix); err != nil {
				t.Fatal(err)
			}
			prefixFlag.Changed = test.prefix != ""
			includesUpdateCmdVersion = test.version
			err := includesUpdateCmd.RunE(includesUpdateCmd, nil)
			if test.want == "" {
				if err == nil {
					t.Fatal("invalid source selection succeeded")
				}
				for _, source := range []string{"itrs", "holidays"} {
					if _, err := os.Lstat(filepath.Join(parent, source+".test.xml")); !os.IsNotExist(err) {
						t.Fatalf("rejected update changed links: %v", err)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			source, _, _ := parseIncludeVersion(test.want)
			link := filepath.Join(parent, source+".test.xml")
			if target, err := os.Readlink(link); err != nil || target != test.want+"/"+source+".test.xml" {
				t.Fatalf("selected link = %q, %v; want version %s", target, err, test.want)
			}
		})
	}
}

func TestLatestIncludeVersionUsesSourceVersionOrder(t *testing.T) {
	h := newTestGatewayHost(t)
	parent := h.PathTo("gateway", "includes")
	for _, name := range []string{
		"holidays-v2026.01.02", "holidays-v2026.11.20", "holidays-v2026.11.31",
		"acme-v1.9.0", "acme-v1.10.0", "acme-v1.10.0-rc.1",
	} {
		if err := os.MkdirAll(filepath.Join(parent, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for prefix, want := range map[string]string{"holidays": "holidays-v2026.11.20", "acme": "acme-v1.10.0"} {
		if got, err := latestIncludeVersion(h, prefix); err != nil || got != want {
			t.Errorf("latest %s = %q, %v; want %s", prefix, got, err, want)
		}
	}
	if _, err := latestIncludeVersion(h, "missing"); err == nil {
		t.Fatal("missing source did not return an error")
	}
}

func TestUpdateDoesNotReplaceExistingLinkWithoutForce(t *testing.T) {
	h := newTestGatewayHost(t)
	version := strings.TrimSpace(cordial.VERSION)
	versionDir := h.PathTo("gateway", "includes", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "managed.xml"), nil, 0444); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	linkPath := h.PathTo("gateway", "includes", "managed.xml")
	if err := os.Symlink("previous/managed.xml", linkPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if err := Update(h, version, false); err == nil {
		t.Fatal("Update() succeeded while replacing an existing link without force")
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if target != "previous/managed.xml" {
		t.Fatalf("existing link target changed to %q", target)
	}
}

func TestUpdateCreatesAndKeepsCurrentLink(t *testing.T) {
	h := newTestGatewayHost(t)
	version := strings.TrimSpace(cordial.VERSION)
	versionDir := h.PathTo("gateway", "includes", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "managed.xml"), nil, 0444); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := Update(h, version, false); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	linkPath := h.PathTo("gateway", "includes", "managed.xml")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if target != filepath.Join(version, "managed.xml") {
		t.Fatalf("link target = %q, want %q", target, filepath.Join(version, "managed.xml"))
	}
	if err := Update(h, version, false); err != nil {
		t.Fatalf("Update() on current link error = %v", err)
	}
}

func TestUpdateForceReplacesExistingLink(t *testing.T) {
	h := newTestGatewayHost(t)
	version := strings.TrimSpace(cordial.VERSION)
	versionDir := h.PathTo("gateway", "includes", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "managed.xml"), nil, 0444); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	linkPath := h.PathTo("gateway", "includes", "managed.xml")
	if err := os.Symlink("previous/managed.xml", linkPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if err := Update(h, version, true); err != nil {
		t.Fatalf("Update(force) error = %v", err)
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if target != filepath.Join(version, "managed.xml") {
		t.Fatalf("link target = %q, want %q", target, filepath.Join(version, "managed.xml"))
	}
}

func TestIncludeVersionLinkCleanup(t *testing.T) {
	for _, operation := range []string{"update", "install-update", "install-preserve", "delete"} {
		for _, reference := range []string{"unused", "relative-group", "absolute", "chain", "direct", "updated", "updated-chain", "direct-updated", "invalid-setup"} {
			t.Run(operation+"/"+reference, func(t *testing.T) {
				h := newTestGatewayHost(t)
				parent := h.PathTo("gateway", "includes")
				oldDirectory := filepath.Join(parent, "v1.0.0")
				newDirectory := filepath.Join(parent, "v2.0.0")
				for _, directory := range []string{oldDirectory, newDirectory} {
					if err := os.MkdirAll(directory, 0755); err != nil {
						t.Fatal(err)
					}
				}
				for _, name := range []string{"aaa-unused.xml", "zzz-used.xml", "itrs.managed.xml"} {
					if err := os.WriteFile(filepath.Join(oldDirectory, name), nil, 0444); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(filepath.Join("v1.0.0", name), filepath.Join(parent, name)); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(newDirectory, "itrs.managed.xml"), nil, 0444); err != nil {
					t.Fatal(err)
				}
				setup := `<gateway><includes/></gateway>`
				switch reference {
				case "relative-group":
					setup = `<gateway><includes><includeGroup name="outer"><includeGroup name="inner" disabled="true"><include disabled="true"><required>false</required><location>../../includes/zzz-used.xml</location></include></includeGroup></includeGroup></includes></gateway>`
				case "absolute":
					setup = `<gateway><includes><include><location>` + filepath.Join(parent, "zzz-used.xml") + `</location></include></includes></gateway>`
				case "chain":
					alias := filepath.Join(t.TempDir(), "alias.xml")
					if err := os.Symlink(filepath.Join(parent, "zzz-used.xml"), alias); err != nil {
						t.Fatal(err)
					}
					setup = `<gateway><includes><include><location>` + alias + `</location></include></includes></gateway>`
				case "direct":
					setup = `<gateway><includes><include><location>` + filepath.Join(oldDirectory, "zzz-used.xml") + `</location></include></includes></gateway>`
				case "updated":
					setup = `<gateway><includes><include><location>` + filepath.Join(parent, "itrs.managed.xml") + `</location></include></includes></gateway>`
				case "updated-chain":
					link := filepath.Join(parent, "itrs.managed.xml")
					if err := os.Remove(link); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink("zzz-used.xml", link); err != nil {
						t.Fatal(err)
					}
					setup = `<gateway><includes><include><location>` + link + `</location></include></includes></gateway>`
				case "direct-updated":
					setup = `<gateway><includes><include><location>` + filepath.Join(oldDirectory, "itrs.managed.xml") + `</location></include></includes></gateway>`
				case "invalid-setup":
					setup = `<gateway>`
				}
				writeTestGatewaySetup(t, h, setup)
				var err error
				switch operation {
				case "update":
					err = Update(h, "v2.0.0", true)
				case "install-update", "install-preserve":
					archive := writeTestIncludeArchive(t,
						includeArchiveEntry{name: "geneos-includes-v2.0.0/", kind: tar.TypeDir},
						includeArchiveEntry{name: "geneos-includes-v2.0.0/v2.0.0/", kind: tar.TypeDir},
						includeArchiveEntry{name: "geneos-includes-v2.0.0/v2.0.0/itrs.managed.xml", contents: "include"},
					)
					err = Install(h, archive, true, operation == "install-update")
				case "delete":
					err = Delete(h, "v1.0.0")
				}
				obsoleteReferenced := reference == "relative-group" || reference == "absolute" || reference == "chain"
				wantError := operation != "install-preserve" && (reference == "invalid-setup" || obsoleteReferenced || operation == "delete" && reference != "unused")
				if (err != nil) != wantError {
					t.Fatalf("cleanup error = %v; want error %v", err, wantError)
				}
				if operation == "delete" && reference != "unused" && reference != "invalid-setup" && !errors.Is(err, ErrIncludeVersionLinked) {
					t.Fatalf("delete error = %v; want referenced-version error", err)
				}
				if operation == "update" || operation == "install-update" {
					target, err := os.Readlink(filepath.Join(parent, "itrs.managed.xml"))
					wantVersion := "v2.0.0"
					if wantError {
						wantVersion = "v1.0.0"
					}
					if err != nil || target != filepath.Join(wantVersion, "itrs.managed.xml") {
						t.Fatalf("replacement link = %q, %v; want version %s", target, err, wantVersion)
					}
				}
				removed := operation != "install-preserve" && (reference == "unused" || operation != "delete" && (reference == "direct" || reference == "updated" || reference == "updated-chain" || reference == "direct-updated"))
				for _, name := range []string{"aaa-unused.xml", "zzz-used.xml"} {
					target, err := os.Readlink(filepath.Join(parent, name))
					if removed {
						if !os.IsNotExist(err) {
							t.Fatalf("unused link %q remains: %v", name, err)
						}
					} else if err != nil || target != filepath.Join("v1.0.0", name) {
						t.Fatalf("protected version lost link %q: target %q, error %v", name, target, err)
					}
				}
				if operation != "delete" || !removed {
					if _, err := os.Stat(oldDirectory); err != nil {
						t.Fatalf("version directory should remain: %v", err)
					}
				}
			})
		}
	}
}

func TestUpdatePreflightChecksAllVersions(t *testing.T) {
	h := newTestGatewayHost(t)
	parent := h.PathTo("gateway", "includes")
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatal(err)
	}
	originalTargets := map[string]string{
		"aaa-unused.xml":   filepath.Join("v0.5.0", "aaa-unused.xml"),
		"zzz-used.xml":     filepath.Join("v1.0.0", "zzz-used.xml"),
		"itrs.managed.xml": filepath.Join(t.TempDir(), "itrs.managed.xml"),
	}
	for name, target := range originalTargets {
		filename := target
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(parent, target)
		}
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, nil, 0444); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(parent, name)); err != nil {
			t.Fatal(err)
		}
	}
	newDirectory := filepath.Join(parent, "v2.0.0")
	if err := os.MkdirAll(newDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"itrs.managed.xml", "itrs.new.xml"} {
		if err := os.WriteFile(filepath.Join(newDirectory, name), nil, 0444); err != nil {
			t.Fatal(err)
		}
	}
	writeTestGatewaySetup(t, h, `<gateway><includes><include><location>../../includes/itrs.managed.xml</location></include><include><location>../../includes/zzz-used.xml</location></include></includes></gateway>`)
	if err := Update(h, "v2.0.0", true); !errors.Is(err, ErrIncludeVersionLinked) {
		t.Fatalf("Update() error = %v; want referenced obsolete link error", err)
	} else {
		for _, message := range []string{"in use by gateway test", "update the gateway setups", "no include links were changed"} {
			if !strings.Contains(err.Error(), message) {
				t.Errorf("Update() error = %v; missing guidance %q", err, message)
			}
		}
	}
	for name, want := range originalTargets {
		if target, err := os.Readlink(filepath.Join(parent, name)); err != nil || target != want {
			t.Fatalf("blocked preflight changed %q: target %q, error %v; want %q", name, target, err, want)
		}
	}
	if _, err := os.Lstat(filepath.Join(parent, "itrs.new.xml")); !os.IsNotExist(err) {
		t.Fatalf("blocked preflight created a new link: %v", err)
	}
	writeTestGatewaySetup(t, h, `<gateway><includes><include><location>../../includes/itrs.managed.xml</location></include></includes></gateway>`)
	if err := Update(h, "v2.0.0", true); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"itrs.managed.xml", "itrs.new.xml"} {
		if target, err := os.Readlink(filepath.Join(parent, name)); err != nil || target != filepath.Join("v2.0.0", name) {
			t.Fatalf("updated link %q = %q, %v; want selected version", name, target, err)
		}
	}
	for _, name := range []string{"aaa-unused.xml", "zzz-used.xml"} {
		if _, err := os.Lstat(filepath.Join(parent, name)); !os.IsNotExist(err) {
			t.Fatalf("obsolete link %q remains after successful preflight: %v", name, err)
		}
	}
}

func TestUpdatePreflightListsAllGateways(t *testing.T) {
	h := newTestGatewayHost(t)
	parent := h.PathTo("gateway", "includes")
	for _, version := range []string{"v1.0.0", "v2.0.0"} {
		if err := os.MkdirAll(filepath.Join(parent, version), 0755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(parent, "obsolete.xml")
	if err := os.Symlink(filepath.Join("v1.0.0", "obsolete.xml"), link); err != nil {
		t.Fatal(err)
	}
	setup := `<gateway><includes><include><location>../../includes/obsolete.xml</location></include><includeGroup name="duplicate"><include><location>../../includes/obsolete.xml</location></include></includeGroup></includes></gateway>`
	writeTestGatewaySetup(t, h, setup, "C", "A", "B")
	err := Update(h, "v2.0.0", true)
	if !errors.Is(err, ErrIncludeVersionLinked) {
		t.Fatalf("Update() error = %v; want referenced obsolete link error", err)
	}
	for _, message := range []string{link, "in use by gateways A, B, C", "update the gateway setups", "no include links were changed"} {
		if !strings.Contains(err.Error(), message) {
			t.Errorf("Update() error = %v; missing %q", err, message)
		}
	}
	if strings.Count(err.Error(), "in use by gateways") != 1 {
		t.Fatalf("duplicate include references were not combined: %v", err)
	}
	if target, err := os.Readlink(link); err != nil || target != filepath.Join("v1.0.0", "obsolete.xml") {
		t.Fatalf("blocked preflight changed obsolete link: target %q, error %v", target, err)
	}
}
