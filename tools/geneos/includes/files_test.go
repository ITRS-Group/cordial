package includes

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	includecontent "github.com/itrs-group/cordial/pkg/includes"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var testGatewayComponentOnce sync.Once

func TestInitialisePreservesExistingIncludes(t *testing.T) {
	h := newTestGatewayHost(t)
	version := embeddedIncludeVersion()
	const oldVersion = "itrs-v0.0.1"
	files := map[string]string{
		h.PathTo("gateway", "includes", version, "itrs.layer0.foundation.xml"):                    "restored bundled-version include",
		h.PathTo("gateway", "includes", oldVersion, "itrs.layer0.foundation.xml"):                 "restored older-version include",
		h.PathTo("gateway", "includes", "itrs.layer1.server-health.xml"):                          "restored flat include",
		h.PathTo("gateway", geneos.SiteName(), geneos.SiteName()+"-config.layer0.foundation.xml"): "restored site config",
	}
	for name, content := range files {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(content), 0o664); err != nil {
			t.Fatal(err)
		}
	}
	links := map[string]string{
		h.PathTo("gateway", "includes", "itrs.layer0.foundation.xml"): oldVersion + "/itrs.layer0.foundation.xml",
		h.PathTo("gateway", "includes", "itrs.legacy.xml"):            oldVersion + "/itrs.legacy.xml",
	}
	for name, target := range links {
		if err := os.Symlink(target, name); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := Initialise(h); err != nil {
			t.Fatal(err)
		}
	}
	for name, want := range files {
		if got, err := os.ReadFile(name); err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want preserved %q", name, got, err, want)
		}
	}
	for name, want := range links {
		if got, err := os.Readlink(name); err != nil || got != want {
			t.Errorf("%s link = %q, %v; want preserved %q", name, got, err, want)
		}
	}
	const missing = "itrs.layer2.database-health.xml"
	want, err := includecontent.Files.ReadFile("files/" + missing)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(h.PathTo("gateway", "includes", version, missing)); err != nil || string(got) != string(want) {
		t.Errorf("missing bundled include = %q, %v; want embedded contents", got, err)
	}
	if got, err := os.Readlink(h.PathTo("gateway", "includes", missing)); err != nil || got != version+"/"+missing {
		t.Errorf("missing include link = %q, %v", got, err)
	}
}

func TestExtractIncludesPreservesExistingConfig(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(fmt.Sprint(symlink), func(t *testing.T) {
			h := newTestGatewayHost(t)
			configDir := h.PathTo("gateway", "local")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(configDir, "local-config.test.xml")
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

func TestExtractIncludesRejectsSymlinkedSiteDirectory(t *testing.T) {
	h := newTestGatewayHost(t)
	parent := h.PathTo("gateway")
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(parent, geneos.SiteName())); err != nil {
		t.Fatal(err)
	}
	source := fstest.MapFS{
		"files/itrs.test.xml":   &fstest.MapFile{Data: []byte("include")},
		"files/config.test.xml": &fstest.MapFile{Data: []byte("template")},
	}
	if err := extractIncludes(h, source, "v1.2.3", true); err == nil {
		t.Fatal("symlinked site directory was accepted")
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
	conf   *config.Config
	home   string
	host   *geneos.Host
	loaded time.Time
}

func (g *includeTestGateway) Name() string               { return filepath.Base(g.home) }
func (g *includeTestGateway) Home() string               { return g.home }
func (g *includeTestGateway) Config() *config.Config     { return g.conf }
func (g *includeTestGateway) Host() *geneos.Host         { return g.host }
func (g *includeTestGateway) Type() *geneos.Component    { return geneos.ParseComponent("gateway") }
func (g *includeTestGateway) Loaded() time.Time          { return g.loaded }
func (g *includeTestGateway) SetLoaded(loaded time.Time) { g.loaded = loaded }
func (g *includeTestGateway) Log() *slog.Logger          { return slog.Default() }
func (g *includeTestGateway) Load() error                { return nil }

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
		gateways[name] = &includeTestGateway{conf: conf, home: home, host: h}
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
