package component_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ac2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ca3"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fileagent"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/floating"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/mdgateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/minimal"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/profile"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/router"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/san"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ssoagent"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/trgateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/webserver"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// inPortRange reports whether port is in a port range specification,
// a comma-separated list of ports, `MIN-MAX` ranges and open-ended
// `MIN-` ranges.
func inPortRange(t *testing.T, spec string, port int) bool {
	t.Helper()
	for item := range strings.SplitSeq(spec, ",") {
		min, max, isRange := strings.Cut(item, "-")
		low, err := strconv.Atoi(min)
		if err != nil {
			t.Fatalf("invalid port range %q: %v", spec, err)
		}
		if !isRange {
			if port == low {
				return true
			}
			continue
		}
		high := 65535
		if max != "" {
			if high, err = strconv.Atoi(max); err != nil {
				t.Fatalf("invalid port range %q: %v", spec, err)
			}
		}
		if port >= low && port <= high {
			return true
		}
	}
	return false
}

func TestInPortRange(t *testing.T) {
	for _, test := range []struct {
		spec string
		port int
		want bool
	}{
		{"7036,7100-", 7036, true},
		{"7036,7100-", 7037, false},
		{"7036,7100-", 7100, true},
		{"7036,7100-", 65000, true},
		{"7038-7039,7100-", 7039, true},
		{"7038-7039,7100-", 7040, false},
		{"4317,4319-", 1180, false},
	} {
		if got := inPortRange(t, test.spec, test.port); got != test.want {
			t.Errorf("inPortRange(%q, %d) = %t, want %t", test.spec, test.port, got, test.want)
		}
	}
}

var defaultPortRE = regexp.MustCompile(`^port=(\d+)$`)

// TestComponentSettings checks the static settings of every registered
// component are consistent with each other, for example that the
// default port is in the port range for the component.
func TestComponentSettings(t *testing.T) {
	cts := append(geneos.RealComponents(), geneos.ParseComponent("profile"))
	if len(cts) < 15 {
		t.Fatalf("only %d components registered", len(cts))
	}

	for _, ct := range cts {
		t.Run(ct.Name, func(t *testing.T) {
			for _, key := range []string{ct.PortRange, ct.CleanList, ct.PurgeList} {
				if key == "" {
					continue
				}
				if _, ok := ct.GlobalSettings[key]; !ok {
					t.Errorf("%q is not in GlobalSettings", key)
				}
			}

			for key := range ct.GlobalSettings {
				alias, ok := ct.ConfigAliases[key]
				if !ok {
					t.Errorf("GlobalSettings key %q has no ConfigAliases entry", key)
					continue
				}
				prefix, _, _ := strings.Cut(key, ".")
				if !strings.HasPrefix(alias, prefix) {
					t.Errorf("alias %q for %q does not start with %q", alias, key, prefix)
				}
			}
			for key := range ct.ConfigAliases {
				if _, ok := ct.GlobalSettings[key]; !ok {
					t.Errorf("ConfigAliases key %q is not in GlobalSettings", key)
				}
			}

			var defaultPort int
			for _, d := range ct.Defaults {
				key, value, ok := strings.Cut(d, "=")
				if !ok {
					t.Errorf("default %q is not NAME=VALUE", d)
					continue
				}
				if strings.HasPrefix(value, `"`) || strings.HasSuffix(value, `"`) {
					t.Errorf("default %q includes literal quotes", d)
				}
				if m := defaultPortRE.FindStringSubmatch(d); m != nil {
					defaultPort, _ = strconv.Atoi(m[1])
				}
				_ = key
			}

			if ct.PortRange == "" {
				if defaultPort != 0 {
					t.Errorf("default port %d but no port range", defaultPort)
				}
				return
			}

			spec := ct.GlobalSettings[ct.PortRange]
			if spec == "" {
				t.Fatal("empty port range")
			}
			if defaultPort != 0 && !inPortRange(t, spec, defaultPort) {
				t.Errorf("default port %d is not in the port range %q", defaultPort, spec)
			}
		})
	}
}

// newComponentRoot creates a root with the package files that each
// component needs when it is added.
func newComponentRoot(t *testing.T) string {
	t.Helper()
	root := geneostest.NewRoot(t, "")
	files := map[string]string{
		"webserver/active_prod/config/security.properties": "port=8443\nkeyStore=config/keystore.db\nkeyStorePassword=changeit\n",
		"webserver/active_prod/config/security.xml":        "",
		"webserver/active_prod/config/config.xml.min.tmpl": "",
		"webserver/active_prod/config/log4j2.properties":   "",
		"webserver/active_prod/config/sso.properties":      "",
		"sso-agent/active_prod/conf/sso-agent.conf": `server {
  key_store { location = "conf/keystore.db", password = "changeit" }
  trust_store { location = "conf/keystore.db", password = "changeit" }
}`,
		"netprobe-router/active_prod/examples/unix/one-to-one/logback.xml": "<configuration/>",
		"netprobe/active_prod/collection_agent/collection-agent.yml":       "agent: {}\n",
		"netprobe/active_prod/collection_agent/logback.xml":                "<configuration/>",
	}
	for name, contents := range files {
		filename := filepath.Join(root, "packages", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"gateway", "licd", "netprobe", "minimal", "fa2", "fileagent", "ac2", "md-gateway", "tr-gateway", "profile"} {
		if err := os.MkdirAll(filepath.Join(root, "packages", name, "active_prod"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestAddedComponents adds every component type in a new installation
// and checks the settings of the new instance, and that its methods do
// not panic.
func TestAddedComponents(t *testing.T) {
	newComponentRoot(t)

	for _, ct := range geneos.RealComponents() {
		t.Run(ct.Name, func(t *testing.T) {
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}

			i, err := instance.Add(geneos.LOCAL, ct, "added-"+ct.Name, instance.Insecure(true))
			if err != nil {
				t.Fatalf("Add() = %v", err)
			}

			if now, _ := os.Getwd(); now != cwd {
				t.Errorf("Add() changed the working directory from %q to %q", cwd, now)
			}

			if ct.PortRange != "" {
				port := config.Get[int](i.Config(), "port")
				if spec := ct.GlobalSettings[ct.PortRange]; !inPortRange(t, spec, port) {
					t.Errorf("port %d is not in the port range %q", port, spec)
				}
			}

			// none of these may panic on a new instance
			for name, fn := range map[string]func(){
				"Command":             func() { _, _, _, _ = i.Command(true) },
				"Command with checks": func() { _, _, _, _ = i.Command(false) },
				"Rebuild":             func() { _, _ = i.Rebuild(true) },
				"Reload":              func() { _ = i.Reload() },
				"String":              func() { _ = i.String() },
			} {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("%s panicked: %v", name, r)
						}
					}()
					fn()
				}()
			}
		})
	}
}

// TestAddWithBadTemplate checks that components that accept a template
// return an error for one that cannot be imported, and not panic.
func TestAddWithBadTemplate(t *testing.T) {
	root := newComponentRoot(t)
	missing := filepath.Join(root, "missing.gotmpl")

	for _, name := range []string{"gateway", "san", "floating", "netprobe-router"} {
		t.Run(name, func(t *testing.T) {
			ct := geneos.ParseComponent(name)
			i, err := instance.Add(geneos.LOCAL, ct, "bad-template-"+name, instance.Insecure(true), instance.WithTemplate(missing))
			if err == nil {
				t.Fatalf("Add() with a missing template succeeded: %v", i)
			}
			if !slices.Contains([]string{"gateway", "san", "floating", "netprobe-router"}, ct.Name) {
				t.Fatal("unexpected component")
			}
			// the failed add must not leave an instance behind
			if existing := instance.Instances(geneos.LOCAL, ct, instance.MatchNames("bad-template-"+name)); len(existing) != 0 {
				t.Errorf("failed Add() left an instance: %v", existing)
			}
		})
	}
}
