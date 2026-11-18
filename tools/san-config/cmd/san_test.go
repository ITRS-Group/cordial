package cmd

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/internal/defaultscheck"
)

// testConf returns the embedded defaults, which must load without
// error, with the YAML merged over them
func testConf(t *testing.T, y string) *config.Config {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	c, err := config.Read("san-config",
		config.WithDefaults(defaults, "yaml"),
		config.StopOnInternalDefaultsErrors(),
		config.Format("yaml"),
		config.SkipWorkingDir(),
		config.SkipUserConfDir(),
		config.SkipSystemDir(),
		config.Reader(strings.NewReader(y)),
	)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	old := uuidNS
	uuidNS = uuid.MustParse(config.Get[string](c, "server.namespace-uuid"))
	t.Cleanup(func() { uuidNS = old })
	return c
}

func TestDefaultsFile(t *testing.T) {
	m := defaultscheck.Parse(t, defaults)

	defaultscheck.RequireKeys(t, m,
		"server.host", "server.port", "server.config-path", "server.connections-path",
		"server.tls.enable", "server.namespace-uuid",
		"server.logs.path", "server.logs.size", "server.logs.backups", "server.logs.age",
		"server.logs.compress", "server.logs.rotate-at-start",
		"geneos.check-interval", "geneos.timeout", "geneos.fallback-gateway.name",
		"geneos.fallback-gateway.primary", "geneos.fallback-gateway.standby", "geneos.fallback-gateway.secure",
		"geneos.sans.gateways", "geneos.sans.retry-interval", "geneos.sans.reverse-connections",
		"geneos.sans.xmlns", "geneos.sans.xsi",
		"inventory.check-interval", "inventory.check-modified", "inventory.type",
		"inventory.mappings", "inventory.indices", "inventory.source",
		"inventory.timeout", "inventory.insecure",
		"components.unknown.probe-name", "components.unknown.entities",
	)
	if _, ok := defaultscheck.Lookup(m, "inventory.timeeout"); ok {
		t.Error("inventory.timeeout is a typo of inventory.timeout")
	}

	defaultscheck.CheckRead(t, m, []string{"."},
		// the mappings are user defined names
		"inventory.mappings",
		// read as a whole through the component name from the inventory
		"components",
	)
}

func TestLogSettings(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantSize   int
		wantBackup int
		wantAge    int
		wantRotate bool
	}{
		{"documented defaults", "", 100, 10, 7, false},
		{"documented keys", "server:\n  logs:\n    size: 5\n    backups: 2\n    age: 3\n    rotate-at-start: true\n", 5, 2, 3, true},
		{"older keys still accepted", "server:\n  logs:\n    max-size: 6\n    max-backups: 4\n    stale-after: 9\n    rotate-on-start: true\n", 6, 4, 9, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := testConf(t, test.yaml)
			if got := logSetting[int](c, "size", "max-size"); got != test.wantSize {
				t.Errorf("size = %d, want %d", got, test.wantSize)
			}
			if got := logSetting[int](c, "backups", "max-backups"); got != test.wantBackup {
				t.Errorf("backups = %d, want %d", got, test.wantBackup)
			}
			if got := logSetting[int](c, "age", "stale-after"); got != test.wantAge {
				t.Errorf("age = %d, want %d", got, test.wantAge)
			}
			if got := logSetting[bool](c, "rotate-at-start", "rotate-on-start"); got != test.wantRotate {
				t.Errorf("rotate = %v, want %v", got, test.wantRotate)
			}
		})
	}
}

func TestInventoryErrorsDoNotPanic(t *testing.T) {
	c := testConf(t, "")
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("a: [unclosed\n  b: : :\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if inv, err := ReadInventory(c, bad); err == nil || inv != nil {
		t.Errorf("ReadInventory(bad YAML) = %v, %v, want an error", inv, err)
	}
	if inv, err := ReadInventory(c, bad, InventoryType("json")); err == nil || inv != nil {
		t.Errorf("ReadInventory(unsupported type) = %v, %v, want an error", inv, err)
	}
	if _, err := ReadInventory(c, filepath.Join(dir, "missing.yaml")); err == nil {
		t.Error("expected an error for a missing file")
	}
	if _, err := FetchInventory(c, "http://[::1", ""); err == nil {
		t.Error("expected an error for an invalid URL")
	}
}

func TestFetchInventory(t *testing.T) {
	c := testConf(t, "")

	var gotAccept, gotIfModified string
	var status int
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotIfModified = r.Header.Get("If-Modified-Since")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	t.Run("loads and caches", func(t *testing.T) {
		status, body = 200, "host1: linux\nHOST2: windows\n"
		cache := filepath.Join(t.TempDir(), "sub", "inventory.yaml")
		inv, err := FetchInventory(c, srv.URL, cache)
		if err != nil {
			t.Fatal(err)
		}
		// keys are lower-cased
		if inv.hosts["host1"] != "linux" || inv.hosts["host2"] != "windows" {
			t.Errorf("hosts = %v", inv.hosts)
		}
		if gotAccept != "application/json" {
			t.Errorf("Accept header = %q", gotAccept)
		}
		if b, err := os.ReadFile(cache); err != nil || string(b) != body {
			t.Errorf("cache = %q, %v", b, err)
		}
	})

	t.Run("not modified", func(t *testing.T) {
		status, body = http.StatusNotModified, ""
		prev := &Inventory{hosts: map[string]string{"a": "b"}, lastModified: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
		inv, err := FetchInventory(c, srv.URL, "", IfModified(prev))
		if err != nil || inv != prev {
			t.Errorf("inv = %v, err = %v, want the previous inventory", inv, err)
		}
		if want := "Fri, 02 Jan 2026 03:04:05 GMT"; gotIfModified != want {
			t.Errorf("If-Modified-Since = %q, want %q", gotIfModified, want)
		}
	})

	t.Run("not modified for an unconditional request", func(t *testing.T) {
		status, body = http.StatusNotModified, ""
		if inv, err := FetchInventory(c, srv.URL, ""); err == nil {
			t.Errorf("inv = %v, want an error", inv)
		}
	})

	t.Run("bad content", func(t *testing.T) {
		status, body = 200, "<html><body>login required: {{{ [</body></html>"
		if inv, err := FetchInventory(c, srv.URL, ""); err == nil {
			t.Errorf("inv = %v, want an error", inv)
		}
	})

	t.Run("error body is bounded", func(t *testing.T) {
		status, body = 500, strings.Repeat("x", 1<<20)
		_, err := FetchInventory(c, srv.URL, "")
		if err == nil || len(err.Error()) > maxErrorBody+100 {
			t.Errorf("err length = %d", len(fmt.Sprint(err)))
		}
	})
}

func TestInventoryTransportHonoursProxyEnvironment(t *testing.T) {
	if defaultInventoryTransport().Proxy == nil {
		t.Error("the default inventory transport ignores proxy settings")
	}
	if inventoryTransport(true).Proxy == nil || inventoryTransport(false).Proxy == nil {
		t.Error("the https inventory transports ignore proxy settings")
	}
	if !inventoryTransport(true).TLSClientConfig.InsecureSkipVerify || inventoryTransport(false).TLSClientConfig.InsecureSkipVerify {
		t.Error("TLS verification settings are wrong")
	}
	if inventoryTransport(true) != inventoryTransport(true) {
		t.Error("transports are not shared")
	}
}

func writeInventory(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "inventory.yaml")
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestReadHostsYAML(t *testing.T) {
	good := writeInventory(t, "winbox: windows\nweb1: linux\n")

	t.Run("hosts", func(t *testing.T) {
		c := testConf(t, "inventory:\n  source: "+good+"\n")
		hosts, err := ReadHostsYAML(c)
		if err != nil {
			t.Fatal(err)
		}
		if hosts["winbox"]["hosttype"] != "windows" || hosts["web1"]["hostname"] != "web1" {
			t.Errorf("hosts = %v", hosts)
		}
	})

	t.Run("all inventories failing is an error", func(t *testing.T) {
		c := testConf(t, "inventory:\n  source: "+filepath.Join(t.TempDir(), "missing.yaml")+"\n")
		if hosts, err := ReadHostsYAML(c); err == nil {
			t.Errorf("hosts = %v, want an error so that existing hosts are kept", hosts)
		}
	})

	t.Run("some inventories failing is not", func(t *testing.T) {
		c := testConf(t, "inventory:\n  indices: [ main, other ]\n  source: "+filepath.Dir(good)+"/${index}.yaml\n")
		if err := os.Rename(good, filepath.Join(filepath.Dir(good), "main.yaml")); err != nil {
			t.Fatal(err)
		}
		hosts, err := ReadHostsYAML(c)
		if err != nil || len(hosts) != 2 {
			t.Errorf("hosts = %v, err = %v", hosts, err)
		}
	})

	t.Run("unsupported inventory type", func(t *testing.T) {
		c := testConf(t, "inventory:\n  type: json\n")
		if hosts, err := LoadHosts(c); err == nil {
			t.Errorf("hosts = %v, want an error", hosts)
		}
	})
}

func TestCheckModifiedStateKey(t *testing.T) {
	Inventories.Clear()
	t.Cleanup(Inventories.Clear)

	file := writeInventory(t, "a: linux\n")
	for _, source := range []string{file, "file:" + file, "file://" + file} {
		t.Run(source, func(t *testing.T) {
			Inventories.Clear()
			c := testConf(t, "inventory:\n  check-modified: true\n  source: "+source+"\n")

			if _, err := ReadHostsYAML(c); err != nil {
				t.Fatal(err)
			}
			first, ok := Inventories.Load(source)
			if !ok {
				t.Fatalf("state not stored with the configured source as key")
			}
			if _, err := ReadHostsYAML(c); err != nil {
				t.Fatal(err)
			}
			second, _ := Inventories.Load(source)
			if first != second {
				t.Error("an unmodified file inventory was loaded again")
			}
		})
	}
}

func TestNetprobeConfig(t *testing.T) {
	inventory := writeInventory(t, "winbox: windows\nweb1: linux\nloopa: a\nloopb: b\nbroken: broken\n")
	c := testConf(t, `
inventory:
  source: `+inventory+`
geneos:
  gateways:
    - { name: gw1, primary: gwhost1, secure: false }
components:
  defaults:
    variables:
      - { name: V1, type: string, value: one }
      - { name: V2, type: string, value: two }
      - { name: V3, type: string, value: three }
  windows:
    probe-name: ${hostname}
    entities:
      - name: ${hostname}
        types: [ "Windows Defaults" ]
  linux:
    probe-name: ${hostname}
    entities:
      - name: one
        variables: [ { name: E, type: string, value: first } ]
      - name: two
        variables: [ { name: E, type: string, value: second } ]
      - name: three
        variables: [ { name: E, type: string, value: third } ]
  a:
    alias: b
  b:
    alias: a
  broken:
    probe-name: x
    entities: not-a-list
`)
	hosts, err := LoadHosts(c)
	if err != nil {
		t.Fatal(err)
	}
	cs := &ConfigServer{conf: c, hosts: hosts, gateways: []string{"gw1"}}

	t.Run("mixed case hostnames find the inventory entry", func(t *testing.T) {
		np, final := cs.NetprobeConfig("WINBOX", "")
		if np == nil || final != "windows" {
			t.Fatalf("np = %v, final = %q, want a windows component", np, final)
		}
		if got := np.SelfAnnounce.ManagedEntities[0].Types.Types; !slices.Equal(got, []string{"Windows Defaults"}) {
			t.Errorf("types = %v", got)
		}
	})

	t.Run("unknown hosts", func(t *testing.T) {
		np, final := cs.NetprobeConfig("nobody", "")
		if np == nil || final != "unknown" {
			t.Errorf("np = %v, final = %q", np, final)
		}
	})

	t.Run("reverse connections setting is used", func(t *testing.T) {
		np, _ := cs.NetprobeConfig("web1", "")
		b, err := xml.Marshal(np)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "<requireReverseConnection>true</requireReverseConnection>") {
			t.Errorf("XML = %s", b)
		}
	})

	t.Run("alias loops return nothing and do not panic", func(t *testing.T) {
		if np, _ := cs.NetprobeConfig("loopa", ""); np != nil {
			t.Errorf("np = %v, want nil", np)
		}
	})

	t.Run("entities must be a list", func(t *testing.T) {
		if np, _ := cs.NetprobeConfig("broken", ""); np != nil {
			t.Errorf("np = %v, want nil", np)
		}
	})

	t.Run("entities do not share variables", func(t *testing.T) {
		np, _ := cs.NetprobeConfig("web1", "")
		if len(np.SelfAnnounce.ManagedEntities) != 3 {
			t.Fatalf("entities = %v", np.SelfAnnounce.ManagedEntities)
		}
		for i, want := range []string{"first", "second", "third"} {
			vars := np.SelfAnnounce.ManagedEntities[i].Vars.Vars
			if len(vars) != 4 || vars[3].Name != "E" || vars[3].String != want {
				t.Errorf("entity %d variables = %+v, want last to be E=%s", i, vars, want)
			}
		}
	})

	t.Run("served config", func(t *testing.T) {
		e := echo.New()
		rec := httptest.NewRecorder()
		ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
		ctx.SetParamNames("hostname")
		ctx.SetParamValues("loopa")
		// an alias loop is a server error and not a panic
		if err := cs.ServeConfig(ctx); err != echo.ErrInternalServerError {
			t.Errorf("err = %v, want internal server error", err)
		}

		rec = httptest.NewRecorder()
		ctx = e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
		ctx.SetParamNames("hostname")
		ctx.SetParamValues("WEB1")
		if err := cs.ServeConfig(ctx); err != nil || rec.Code != 200 {
			t.Errorf("err = %v, status = %d", err, rec.Code)
		}
	})
}

func TestGetVars(t *testing.T) {
	c := testConf(t, `
components:
  test:
    variables:
      - { name: IntFromInt, type: int, value: 23 }
      - { name: IntFromString, type: integer, value: "42" }
      - { name: IntFromWholeFloat, type: int, value: 7.0 }
      - { name: IntFromFraction, type: int, value: 7.5 }
      - { name: DoubleFromInt, type: double, value: 2 }
      - { name: DoubleFromFloat, type: float, value: 1.25 }
      - { name: DoubleFromString, type: double, value: "3.5" }
      - { name: DoubleBad, type: double, value: abc }
      - { name: BoolFromBool, type: bool, value: true }
      - { name: BoolFromString, type: boolean, value: "false" }
      - { name: BoolBad, type: bool, value: maybe }
      - { name: StringFromNumber, type: string, value: 5 }
      - { name: List, type: stringlist, value: [ a, b ] }
      - { name: ListBad, type: stringlist, value: notalist }
      - { name: Regex, type: regex, value: "/a.*b/is" }
      - { name: Unsupported, type: macro, value: x }
`)
	got := map[string]string{}
	for _, v := range getVars(c, "components.test.variables") {
		switch {
		case v.Integer != nil:
			got[v.Name] = fmt.Sprintf("int:%d", *v.Integer)
		case v.Double != nil:
			got[v.Name] = fmt.Sprintf("double:%g", *v.Double)
		case v.Boolean != nil:
			got[v.Name] = fmt.Sprintf("bool:%t", *v.Boolean)
		case v.StringList != nil:
			got[v.Name] = "list:" + strings.Join(v.StringList.Strings, ",")
		case v.Regex != nil:
			got[v.Name] = fmt.Sprintf("regex:%s:%t:%t", v.Regex.Regex, *v.Regex.Flags.CaseInsensitive, *v.Regex.Flags.DotMatchesAll)
		default:
			got[v.Name] = "string:" + v.String
		}
	}
	want := map[string]string{
		"IntFromInt":        "int:23",
		"IntFromString":     "int:42",
		"IntFromWholeFloat": "int:7",
		"DoubleFromInt":     "double:2",
		"DoubleFromFloat":   "double:1.25",
		"DoubleFromString":  "double:3.5",
		"BoolFromBool":      "bool:true",
		"BoolFromString":    "bool:false",
		"StringFromNumber":  "string:5",
		"List":              "list:a,b",
		"Regex":             "regex:a.*b:true:true",
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s = %q, want %q", k, got[k], w)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("variable %s should have been skipped, got %q", k, got[k])
		}
	}
}

func TestNetprobeID(t *testing.T) {
	tests := []struct {
		name, grouping, host, want string
	}{
		{"no grouping", "", "hosta", "hosta"},
		{"capture group", "^(.*?)[ab]?$", "hosta", "host"},
		{"no match", "^zzz(.*)", "hosta", "hosta"},
		{"no capture group", "^host", "hosta", "hosta"},
		{"invalid", "([", "hosta", "hosta"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := testConf(t, "geneos:\n  sans:\n    grouping: "+fmt.Sprintf("%q", test.grouping)+"\n")
			// twice, to use the cached expression
			for range 2 {
				if got := netprobeID(c, test.host); got != test.want {
					t.Errorf("netprobeID = %q, want %q", got, test.want)
				}
			}
		})
	}
}

func TestGateways(t *testing.T) {
	c := testConf(t, `
geneos:
  sans:
    gateways: 0
  fallback-gateway: { name: Fallback, primary: fallback.example.com, standby: fallback2.example.com, secure: true }
  gateways:
    - { name: onlyname }
    - { name: gw2, primary: "host2:7100", standby: "host3:7100", secure: true }
`)

	t.Run("primary defaults to name", func(t *testing.T) {
		g := GatewayDetails("onlyname", config.Get[[]map[string]string](c, "geneos.gateways"))
		if g.Primary != "onlyname" || g.PrimaryPort != 7039 {
			t.Errorf("gateway = %+v", g)
		}
		if g := GatewayDetails("", config.Get[[]map[string]string](c, "geneos.gateways")); g.Primary != "" {
			t.Errorf("empty name matched %+v", g)
		}
	})

	t.Run("selection", func(t *testing.T) {
		cs := &ConfigServer{conf: c, gateways: []string{"gw2"}}
		gws := cs.Gateways("anyhost")
		if len(gws) != 2 || gws[0].Hostname != "host2" || gws[0].Port != 7100 || !gws[0].Secure || gws[1].Hostname != "host3" {
			t.Errorf("gateways = %+v", gws)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		cs := &ConfigServer{conf: c}
		gws := cs.Gateways("anyhost")
		if len(gws) != 2 || gws[0].Hostname != "fallback.example.com" || gws[1].Hostname != "fallback2.example.com" {
			t.Errorf("gateways = %+v", gws)
		}
	})
}

func TestCheckGateways(t *testing.T) {
	var live, down int
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/liveness" {
			live++
		}
	}))
	defer up.Close()
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		down++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer broken.Close()

	c := testConf(t, fmt.Sprintf(`
geneos:
  timeout: 5s
  gateways:
    - { name: up, primary: %q }
    - { name: standbyup, primary: "127.0.0.1:1", standby: %q }
    - { name: down, primary: %q }
    - { secure: false }
`, strings.TrimPrefix(up.URL, "http://"), strings.TrimPrefix(up.URL, "http://"), strings.TrimPrefix(broken.URL, "http://")))

	got := CheckGateways(c)
	slices.Sort(got)
	if want := []string{"standbyup", "up"}; !slices.Equal(got, want) {
		t.Errorf("live gateways = %v, want %v", got, want)
	}
	if live == 0 || down == 0 {
		t.Errorf("liveness requests: up = %d, down = %d", live, down)
	}
}

func TestServeConnectionKeepsSharedGateways(t *testing.T) {
	c := testConf(t, "geneos:\n  gateways:\n    - { name: b, primary: hostb }\n    - { name: a, primary: hosta }\n")
	cs := &ConfigServer{conf: c, gateways: []string{"b", "a"}}

	e := echo.New()
	rec := httptest.NewRecorder()
	if err := cs.ServeConnection(e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cs.gateways, []string{"b", "a"}) {
		t.Errorf("shared gateway list was re-ordered: %v", cs.gateways)
	}
	if lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n"); len(lines) != 2 || !strings.HasPrefix(lines[0], "hosta~") {
		t.Errorf("connections = %q", rec.Body.String())
	}
}

func TestSafelyRecovers(t *testing.T) {
	ran := false
	safely("test", func() { panic("boom") })
	safely("test", func() { ran = true })
	if !ran {
		t.Error("safely did not run the function")
	}
}

func TestWriteCache(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "cache.yaml")
	for _, content := range []string{"first", "second"} {
		if err := writeCache(target, []byte(content)); err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(target); string(b) != content {
			t.Errorf("cache = %q, want %q", b, content)
		}
	}
	if m, _ := filepath.Glob(filepath.Join(dir, "a", "b", "*")); len(m) != 1 {
		t.Errorf("temporary files left behind: %v", m)
	}
}

func TestInventoryCacheSetting(t *testing.T) {
	c := testConf(t, "inventory:\n  indices: [ main ]\n  cache: "+t.TempDir()+"/${index}.yaml\n")
	expand := []config.ExpandOption{config.LookupTable(map[string]string{"index": "main"})}
	if got := inventoryCache(c, expand); !strings.HasSuffix(got, "/main.yaml") {
		t.Errorf("cache = %q", got)
	}
	if got := inventoryCache(testConf(t, ""), nil); got != "" {
		t.Errorf("cache = %q, want none by default", got)
	}
}
