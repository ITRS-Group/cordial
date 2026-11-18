package geneos

import (
	"errors"
	"io"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/logger"
)

func TestComponentIteration(t *testing.T) {
	names := []string{"itz", "ita", "itm", "itb"}
	for _, name := range names {
		registeredComponents[name] = &Component{Name: name, UsesKeyfiles: name != "itm"}
		t.Cleanup(func() { delete(registeredComponents, name) })
	}
	sorted := []string{"ita", "itb", "itm", "itz"}

	nameOf := func(cts []*Component) (got []string) {
		for _, ct := range cts {
			got = append(got, ct.Name)
		}
		return
	}

	// the order must not depend on map iteration, so repeat
	for range 20 {
		var orList []*Component
		for ct := range (*Component)(nil).OrList() {
			orList = append(orList, ct)
		}
		if got := nameOf(orList); !slices.Equal(got, sorted) {
			t.Fatalf("OrList() = %v, want %v", got, sorted)
		}
		if got := nameOf(AllComponents()); !slices.Equal(got, sorted) {
			t.Fatalf("AllComponents() = %v, want %v", got, sorted)
		}
		if got := nameOf(UsesKeyFiles()); !slices.Equal(got, []string{"ita", "itb", "itz"}) {
			t.Fatalf("UsesKeyFiles() = %v", got)
		}
	}
}

func TestComponentIsA(t *testing.T) {
	ct := &Component{Name: "Gateway", Aliases: []string{"gw", "Gateway2"}}
	for names, want := range map[string]bool{
		"gateway": true, "GW": true, "gateway2": true, "netprobe": false, "": false,
	} {
		if got := ct.IsA(names); got != want {
			t.Errorf("IsA(%q) = %v, want %v", names, got, want)
		}
	}
	if !ct.IsA("netprobe", "gw") {
		t.Error("IsA() with several names did not match")
	}
	if (*Component)(nil).IsA("gateway") {
		t.Error("nil component IsA() matched")
	}
}

// TestComponentGlobalSetting checks that the registered default is used,
// that an empty default is empty and not the name of the legacy alias,
// and that a legacy setting is honoured
func TestComponentGlobalSetting(t *testing.T) {
	cf := config.Global()
	ct := &Component{
		Name:           "settingtest",
		GlobalSettings: map[string]string{"settingtest.clean": "*.old:*.bak"},
		ConfigAliases: map[string]string{
			"settingtest.clean": "settingtestcleanlist",
			"settingtest.purge": "settingtestpurgelist",
			"settingtest.ports": "settingtestportrange",
		},
	}
	for k, v := range ct.GlobalSettings {
		cf.Default(k, v)
	}
	keys := []string{"settingtest.clean", "settingtestcleanlist", "settingtest.purge", "settingtestpurgelist", "settingtest.ports", "settingtestportrange"}
	t.Cleanup(func() {
		for _, k := range keys {
			config.Delete(cf, k)
		}
	})

	if got := ct.GlobalSetting("settingtest.clean"); got != "*.old:*.bak" {
		t.Errorf("default clean list = %q", got)
	}
	// no default is an empty value, not the name of the alias
	if got := ct.GlobalSetting("settingtest.purge"); got != "" {
		t.Errorf("purge list with no default = %q, want empty", got)
	}

	config.Set(cf, "settingtestpurgelist", "*.log")
	if got := ct.GlobalSetting("settingtest.purge"); got != "*.log" {
		t.Errorf("legacy purge list = %q, want *.log", got)
	}

	// a legacy setting is used over the registered default, but a new
	// setting overrides the legacy one
	config.Set(cf, "settingtestcleanlist", "legacy")
	if got := ct.GlobalSetting("settingtest.clean"); got != "legacy" {
		t.Errorf("legacy clean list over default = %q, want legacy", got)
	}
	config.Set(cf, "settingtest.clean", "new")
	if got := ct.GlobalSetting("settingtest.clean"); got != "new" {
		t.Errorf("new clean list = %q, want new", got)
	}

	config.Set(cf, "settingtestportrange", "7000-7100")
	if got := ct.GlobalSetting("settingtest.ports"); got != "7000-7100" {
		t.Errorf("legacy port range = %q", got)
	}

	var nilComponent *Component
	if got := nilComponent.GlobalSetting("x"); got != "" {
		t.Errorf("nil component setting = %q", got)
	}
}

// notLocalHost is the local file system, but not the local host
type notLocalHost struct{ host.Host }

func (notLocalHost) IsLocalhost() bool { return false }

// newNamedTestHost returns a host, on the local file system, called name
// with root as its Geneos directory
func newNamedTestHost(name, root string) *Host {
	h := &Host{Host: notLocalHost{host.NewLocal()}, Config: config.New(config.KeyDelimiter("::")), loaded: true}
	config.Set(h.Config, "name", name)
	config.Set(h.Config, cordial.ExecutableName(), root)
	return h
}

func TestInitialiseOnceIsPerHost(t *testing.T) {
	var mutex sync.Mutex
	var calls []string
	fail := true
	ct := &Component{Name: "inittest", Initialise: func(h *Host, ct *Component, override bool) error {
		mutex.Lock()
		defer mutex.Unlock()
		calls = append(calls, h.String())
		if h.String() == "failing" && fail {
			return errors.New("not yet")
		}
		return nil
	}}
	local := newNamedTestHost("localhost", t.TempDir())
	second := newNamedTestHost("second", t.TempDir())

	for range 3 {
		for _, h := range []*Host{local, second} {
			if err := ct.InitialiseOnce(h, false); err != nil {
				t.Fatalf("InitialiseOnce(%s) error = %v", h, err)
			}
		}
	}
	if want := []string{"localhost", "second"}; !slices.Equal(calls, want) {
		t.Errorf("Initialise was called for %v, want once for each of %v", calls, want)
	}

	t.Run("failure is retried", func(t *testing.T) {
		calls = nil
		failing := newNamedTestHost("failing", t.TempDir())
		if err := ct.InitialiseOnce(failing, false); err == nil {
			t.Fatal("InitialiseOnce() did not return the error from Initialise")
		}
		fail = false
		for range 2 {
			if err := ct.InitialiseOnce(failing, false); err != nil {
				t.Fatal(err)
			}
		}
		if want := []string{"failing", "failing"}; !slices.Equal(calls, want) {
			t.Errorf("Initialise was called for %v, want %v", calls, want)
		}
	})

	t.Run("concurrent", func(t *testing.T) {
		var n atomic.Int64
		ct := &Component{Name: "inittest2", Initialise: func(*Host, *Component, bool) error {
			n.Add(1)
			return nil
		}}
		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() { _ = ct.InitialiseOnce(local, false) })
		}
		wg.Wait()
		if n.Load() != 1 {
			t.Errorf("Initialise was called %d times, want 1", n.Load())
		}
	})

	t.Run("no initialise function", func(t *testing.T) {
		if err := (&Component{Name: "none"}).InitialiseOnce(local, false); err != nil {
			t.Error(err)
		}
		if err := (*Component)(nil).InitialiseOnce(local, false); err != nil {
			t.Error(err)
		}
	})
}

// TestInitialiseInitialisesEachHost checks that a second host gets the
// component initialisation, and not only the first
func TestInitialiseInitialisesEachHost(t *testing.T) {
	useTestConfigDir(t)
	useTestComponents(t)
	first := filepath.Join(t.TempDir(), "geneos")
	second := filepath.Join(t.TempDir(), "geneos")
	setTestHosts(t, filepath.Join(t.TempDir(), "not-the-root"))

	var calls []string
	ct := &Component{Name: "inithost", Initialise: func(h *Host, ct *Component, override bool) error {
		calls = append(calls, h.String()+":"+config.Get[string](h.Config, cordial.ExecutableName()))
		return nil
	}}
	ct.Register(nil)
	t.Cleanup(func() {
		delete(registeredComponents, ct.Name)
		delete(initDirs, ct.Name)
	})
	t.Cleanup(func() {
		globalAuditMutex.Lock()
		defer globalAuditMutex.Unlock()
		if globalAuditWriter != nil {
			globalAuditWriter.Close()
			globalAuditWriter = nil
		}
		logger.NewGlobalAuditLogger(io.Discard)
		config.Delete(config.Global(), cordial.ExecutableName())
	})

	remote := newNamedTestHost("second", second)
	for _, c := range []struct {
		h    *Host
		root string
	}{{LOCAL, first}, {remote, second}, {LOCAL, first}, {remote, second}} {
		if err := Initialise("init", c.h, UseRoot(c.root), Insecure(true), Override(true), ForComponent(ct)); err != nil {
			t.Fatalf("Initialise(%s) error = %v", c.h, err)
		}
	}
	if want := []string{"localhost:" + first, "second:" + second}; !slices.Equal(calls, want) {
		t.Errorf("Initialise was called for %v, want %v", calls, want)
	}
}
