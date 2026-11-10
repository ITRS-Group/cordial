package includes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/host"
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
					archive := writeTestIncludeArchive(t, includeArchiveEntry{name: "v2.0.0/itrs.managed.xml", contents: "include"})
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
