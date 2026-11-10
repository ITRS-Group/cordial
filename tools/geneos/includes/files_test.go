package includes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

type includeTestGateway struct {
	geneos.Instance
	conf *config.Config
	home string
}

func (g *includeTestGateway) Name() string           { return filepath.Base(g.home) }
func (g *includeTestGateway) Home() string           { return g.home }
func (g *includeTestGateway) Config() *config.Config { return g.conf }
func (g *includeTestGateway) Load() error            { return nil }

func writeTestGatewaySetup(t *testing.T, h *geneos.Host, setup string, names ...string) {
	t.Helper()
	previousHost := geneos.LOCAL
	geneos.LOCAL = h
	config.Set(h.Config, "name", "localhost")
	t.Cleanup(func() { geneos.LOCAL = previousHost })
	if len(names) == 0 {
		names = []string{"test"}
	}
	gateways := make(map[string]*includeTestGateway)
	for _, name := range names {
		home := h.PathTo("gateway", "gateways", name)
		if err := os.MkdirAll(home, 0755); err != nil {
			t.Fatal(err)
		}
		filename := filepath.Join(home, "gateway.setup.xml")
		if err := os.WriteFile(filename, []byte(setup), 0644); err != nil {
			t.Fatal(err)
		}
		conf := config.New()
		config.Set(conf, "setup", filename)
		gateways[name] = &includeTestGateway{conf: conf, home: home}
	}
	ct := geneos.ParseComponent("gateway")
	previous := ct.New
	t.Cleanup(func() { ct.New = previous })
	ct.New = func(name string) geneos.Instance {
		local, _, _ := strings.Cut(name, "@")
		if gateway, ok := gateways[local]; ok {
			return gateway
		}
		if previous != nil {
			return previous(name)
		}
		return nil
	}
}
