package includes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
)

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
