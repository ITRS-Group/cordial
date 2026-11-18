package geneos

import (
	"slices"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
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
