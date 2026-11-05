package includes

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var testGatewayComponentOnce sync.Once

func TestExtractIncludesPreservesExistingConfig(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(fmt.Sprint(symlink), func(t *testing.T) {
			h := newTestGatewayHost(t)
			configDir := h.PathTo("gateway", "includes", "config")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(configDir, "config.test.xml")
			if symlink {
				if err := os.Symlink("missing.xml", configPath); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(configPath, []byte("local"), 0664); err != nil {
				t.Fatal(err)
			}
			source := fstest.MapFS{
				"files/itrs.test.xml":   &fstest.MapFile{Data: []byte("include")},
				"files/config.test.xml": &fstest.MapFile{Data: []byte("template")},
			}
			if err := extractIncludes(h, source, "v1.2.3", true); err != nil {
				t.Fatal(err)
			}
			if symlink {
				if target, err := os.Readlink(configPath); err != nil || target != "missing.xml" {
					t.Fatalf("config symlink = %q, %v", target, err)
				}
			} else if data, err := os.ReadFile(configPath); err != nil || string(data) != "local" {
				t.Fatalf("config contents = %q, %v", data, err)
			}
		})
	}
}

func newTestGatewayHost(t *testing.T) *geneos.Host {
	t.Helper()
	testGatewayComponentOnce.Do(func() {
		(&geneos.Component{Name: "gateway"}).Register(nil)
	})

	h := &geneos.Host{Host: host.NewLocal(), Config: config.New()}
	config.Set(h.Config, cordial.ExecutableName(), t.TempDir())
	return h
}

func TestExtractRejectsExistingVersionDirectoryWithoutForce(t *testing.T) {
	h := newTestGatewayHost(t)
	versionDir := h.PathTo("gateway", "includes", cordial.VERSION)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := Extract(h, false); err == nil {
		t.Fatal("Extract() succeeded with an existing version directory and no force")
	}
}

func TestExtractCreatesMissingLinksAndPreservesExistingPaths(t *testing.T) {
	for _, existingFile := range []bool{false, true} {
		t.Run(fmt.Sprint(existingFile), func(t *testing.T) {
			h := newTestGatewayHost(t)
			includesDir := h.PathTo("gateway", "includes")
			if err := os.MkdirAll(includesDir, 0755); err != nil {
				t.Fatal(err)
			}
			existing := filepath.Join(includesDir, "itrs.level0.authentication.xml")
			if existingFile {
				if err := os.WriteFile(existing, []byte("local"), 0644); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink("previous/itrs.level0.authentication.xml", existing); err != nil {
				t.Fatal(err)
			}
			if err := Extract(h, true); err != nil {
				t.Fatal(err)
			}
			if existingFile {
				if data, err := os.ReadFile(existing); err != nil || string(data) != "local" {
					t.Fatalf("existing file = %q, %v", data, err)
				}
			} else if target, err := os.Readlink(existing); err != nil || target != "previous/itrs.level0.authentication.xml" {
				t.Fatalf("existing link = %q, %v", target, err)
			}
			missing := filepath.Join(includesDir, "itrs.level0.geneos.xml")
			if target, err := os.Readlink(missing); err != nil || target != filepath.Join(cordial.VERSION, "itrs.level0.geneos.xml") {
				t.Fatalf("new link = %q, %v", target, err)
			}
		})
	}
}

func TestExtractCreatesVersionedAndConfigFiles(t *testing.T) {
	h := newTestGatewayHost(t)
	if err := Extract(h, false); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	versionDir := h.PathTo("gateway", "includes", cordial.VERSION)
	versionFiles, err := os.ReadDir(versionDir)
	if err != nil {
		t.Fatalf("ReadDir(versioned includes) error = %v", err)
	}
	if len(versionFiles) == 0 {
		t.Fatal("Extract() created no versioned include files")
	}
	for _, entry := range versionFiles {
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("Info(%q) error = %v", entry.Name(), err)
		}
		if got := info.Mode().Perm(); got != 0444 {
			t.Errorf("versioned file %q permissions = %04o, want 0444", entry.Name(), got)
		}
	}

	configDir := filepath.Join(h.PathTo("gateway"), "includes", "config")
	configFiles, err := os.ReadDir(configDir)
	if err != nil {
		t.Fatalf("ReadDir(config includes) error = %v", err)
	}
	if len(configFiles) == 0 {
		t.Fatal("Extract() created no config template files")
	}
}
