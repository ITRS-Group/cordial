package includes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeleteProtectsEntireSelectedVersion(t *testing.T) {
	for _, kind := range []string{"relative", "absolute", "chain", "nested", "dangling", "directory", "external-target"} {
		t.Run(kind, func(t *testing.T) {
			h := newTestGatewayHost(t)
			parent := h.PathTo("gateway", "includes")
			for _, version := range []string{"v1.0.0", "v1.2.3"} {
				if err := os.MkdirAll(filepath.Join(parent, version, "nested"), 0755); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"linked.xml", "unlinked.xml", "nested/unlinked.xml"} {
					if err := os.WriteFile(filepath.Join(parent, version, name), []byte(name), 0444); err != nil {
						t.Fatal(err)
					}
				}
			}
			target := "v1.2.3/linked.xml"
			switch kind {
			case "absolute":
				target = filepath.Join(parent, target)
			case "chain":
				alias := filepath.Join(t.TempDir(), "alias.xml")
				if err := os.Symlink(filepath.Join(parent, target), alias); err != nil {
					t.Fatal(err)
				}
				target = alias
			case "nested":
				target = "v1.2.3/nested/unlinked.xml"
			case "dangling":
				target = "v1.2.3/missing.xml"
			case "directory":
				target = "v1.2.3"
			case "external-target":
				file := filepath.Join(parent, target)
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(t.TempDir(), "manual.xml"), file); err != nil {
					t.Fatal(err)
				}
			}
			link := filepath.Join(parent, "current")
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			for _, pattern := range []string{"v1.2.3", "v1.*", "*"} {
				err := Delete(h, pattern)
				if !errors.Is(err, ErrIncludeVersionLinked) {
					t.Fatalf("Delete(%q) error = %v; want linked-version error", pattern, err)
				}
				if !strings.Contains(err.Error(), "v1.2.3") || !strings.Contains(err.Error(), link) || !strings.Contains(err.Error(), "no selected versions were deleted") {
					t.Fatalf("linked-version error does not identify the protected selection: %v", err)
				}
				for _, version := range []string{"v1.0.0", "v1.2.3"} {
					for _, name := range []string{"linked.xml", "unlinked.xml", "nested/unlinked.xml"} {
						if _, err := os.Lstat(filepath.Join(parent, version, name)); err != nil {
							t.Fatalf("selected version %q lost file %q: %v", version, name, err)
						}
					}
				}
			}
		})
	}
}

func TestDeleteDefaultRemovesOnlyUnlinkedVersions(t *testing.T) {
	h := newTestGatewayHost(t)
	parent := h.PathTo("gateway", "includes")
	for _, version := range []string{"v1.0.0", "v1.2.3", "1.0.0", "config", "custom"} {
		if err := os.MkdirAll(filepath.Join(parent, version), 0755); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"linked.xml", "unlinked.xml"} {
			if err := os.WriteFile(filepath.Join(parent, version, name), nil, 0444); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.Symlink("v1.2.3/linked.xml", filepath.Join(parent, "current.xml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(parent, "v1.0.0/linked.xml"), filepath.Join(t.TempDir(), "manual.xml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("custom", filepath.Join(parent, "v3.0.0")); err != nil {
		t.Fatal(err)
	}
	if err := Delete(h, ""); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1.0.0", "1.0.0"} {
		if _, err := os.Lstat(filepath.Join(parent, version)); !os.IsNotExist(err) {
			t.Fatalf("unlinked version %q was not removed: %v", version, err)
		}
	}
	for _, version := range []string{"v1.2.3", "config", "custom"} {
		for _, name := range []string{"linked.xml", "unlinked.xml"} {
			if _, err := os.Stat(filepath.Join(parent, version, name)); err != nil {
				t.Fatalf("protected directory %q lost %q: %v", version, name, err)
			}
		}
	}
	if _, err := os.Lstat(filepath.Join(parent, "v3.0.0")); err != nil {
		t.Fatalf("directory symlink was removed: %v", err)
	}
}

func TestDeleteVersionPatterns(t *testing.T) {
	for _, pattern := range []string{"v1.*", "v1.2.[34]", "v1.2.?", "v1.2.3", "v9.*"} {
		t.Run(pattern, func(t *testing.T) {
			h := newTestGatewayHost(t)
			parent := h.PathTo("gateway", "includes")
			for _, version := range []string{"v1.2.3", "v1.2.4", "v2.0.0", "config"} {
				if err := os.MkdirAll(filepath.Join(parent, version), 0755); err != nil {
					t.Fatal(err)
				}
			}
			if err := Delete(h, pattern); err != nil {
				t.Fatal(err)
			}
			for _, version := range []string{"v1.2.3", "v1.2.4", "v2.0.0", "config"} {
				removed := version == "v1.2.3" && pattern != "v9.*" || version == "v1.2.4" && pattern != "v9.*" && pattern != "v1.2.3"
				_, err := os.Stat(filepath.Join(parent, version))
				if removed && !os.IsNotExist(err) || !removed && err != nil {
					t.Fatalf("version %q stat error = %v; want removed %v", version, err, removed)
				}
			}
		})
	}
}

func TestDeleteRejectsInvalidPatternsAndUnsafeLinks(t *testing.T) {
	for _, pattern := range []string{"../*", "/tmp/*", `v1\*`, ".", "..", "["} {
		t.Run(pattern, func(t *testing.T) {
			h := newTestGatewayHost(t)
			directory := h.PathTo("gateway", "includes", "v1.2.3")
			if err := os.MkdirAll(directory, 0755); err != nil {
				t.Fatal(err)
			}
			if err := Delete(h, pattern); err == nil {
				t.Fatal("Delete() accepted an invalid pattern")
			}
			if _, err := os.Stat(directory); err != nil {
				t.Fatalf("invalid pattern changed installation: %v", err)
			}
		})
	}
	t.Run("cycle", func(t *testing.T) {
		h := newTestGatewayHost(t)
		directory := h.PathTo("gateway", "includes", "v1.2.3")
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(filepath.Dir(directory), "loop")
		if err := os.Symlink("loop", link); err != nil {
			t.Fatal(err)
		}
		if err := Delete(h, ""); err == nil || !strings.Contains(err.Error(), link) {
			t.Fatalf("unsafe link error = %v; want error identifying %q", err, link)
		}
		if _, err := os.Stat(directory); err != nil {
			t.Fatalf("unsafe link did not prevent deletion: %v", err)
		}
	})
}

func TestDeleteMissingIncludesDirectory(t *testing.T) {
	if err := Delete(newTestGatewayHost(t), ""); err != nil {
		t.Fatal(err)
	}
}

func TestIncludesDeleteArgumentsAndAlias(t *testing.T) {
	flag := includesDeleteCmd.Flags().Lookup("version")
	previousChanged := flag.Changed
	t.Cleanup(func() { flag.Changed = previousChanged })
	for _, test := range []struct {
		name    string
		args    []string
		flag    bool
		wantErr bool
	}{
		{name: "all"},
		{name: "positional", args: []string{"v1.*"}},
		{name: "flag", flag: true},
		{name: "both", args: []string{"v1.*"}, flag: true, wantErr: true},
		{name: "multiple", args: []string{"v1.*", "v2.*"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			flag.Changed = test.flag
			if err := includesDeleteCmd.Args(includesDeleteCmd, test.args); (err != nil) != test.wantErr {
				t.Fatalf("argument validation error = %v; want error %v", err, test.wantErr)
			}
		})
	}
	if len(includesDeleteCmd.Aliases) != 1 || includesDeleteCmd.Aliases[0] != "remove" {
		t.Fatalf("delete aliases = %v; want remove", includesDeleteCmd.Aliases)
	}
}

func TestDeleteWorksThroughPatterns(t *testing.T) {
	h := newTestGatewayHost(t)
	parent := h.PathTo("gateway", "includes")
	for _, version := range []string{"v1.0.0", "v1.2.3", "v2.0.0", "v3.0.0"} {
		if err := os.MkdirAll(filepath.Join(parent, version), 0755); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"linked.xml", "unlinked.xml"} {
			if err := os.WriteFile(filepath.Join(parent, version, name), nil, 0444); err != nil {
				t.Fatal(err)
			}
		}
	}
	link := filepath.Join(parent, "current.xml")
	if err := os.Symlink("v1.2.3/linked.xml", link); err != nil {
		t.Fatal(err)
	}
	err := Delete(h, "v1.*", "[", "v2.*", "v2.0.0", "v3.0.0")
	if !errors.Is(err, ErrIncludeVersionLinked) || !strings.Contains(err.Error(), link) || !strings.Contains(err.Error(), "invalid include version pattern") {
		t.Fatalf("Delete() error = %v; want linked and invalid pattern errors", err)
	}
	for _, version := range []string{"v1.0.0", "v1.2.3"} {
		for _, name := range []string{"linked.xml", "unlinked.xml"} {
			if _, err := os.Stat(filepath.Join(parent, version, name)); err != nil {
				t.Fatalf("blocked pattern lost %s/%s: %v", version, name, err)
			}
		}
	}
	for _, version := range []string{"v2.0.0", "v3.0.0"} {
		if _, err := os.Stat(filepath.Join(parent, version)); !os.IsNotExist(err) {
			t.Fatalf("later pattern was not processed for %q: %v", version, err)
		}
	}
}
