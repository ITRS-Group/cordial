package includes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
)

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
