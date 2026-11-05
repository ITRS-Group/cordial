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
