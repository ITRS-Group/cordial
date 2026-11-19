package values_test

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/san"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

func newProfileInstance(t *testing.T, options map[string]any) geneos.Instance {
	t.Helper()
	geneostest.NewRoot(t, "")
	i := geneos.ParseComponent("profile").New("test-profile")
	if i == nil {
		t.Fatal("cannot create profile instance")
	}
	cf := i.Config()
	config.Set(cf, cf.Join("config", "options"), options)
	return i
}

func profileOptions(i geneos.Instance) map[string]any {
	cf := i.Config()
	return config.Get[map[string]any](cf, cf.Join("config", "options"))
}

func TestSetProfileKeepsOptionTypes(t *testing.T) {
	i := newProfileInstance(t, map[string]any{"insecure": true, "minimal": false, "gateway": "a:7039"})

	cf, err := values.Set(i, values.Values{Params: values.NameValues{"port=8080"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	options := config.Get[map[string]any](cf, cf.Join("config", "options"))
	if got, ok := options["insecure"].(bool); !ok || !got {
		t.Errorf("insecure = %#v, want bool true", options["insecure"])
	}
	if got, ok := options["minimal"].(bool); !ok || got {
		t.Errorf("minimal = %#v, want bool false", options["minimal"])
	}
	if options["port"] != "8080" {
		t.Errorf("port = %#v, want 8080", options["port"])
	}
}

func TestSetProfileAccumulatesTypesAndParams(t *testing.T) {
	i := newProfileInstance(t, map[string]any{})

	cf, err := values.Set(i, values.Values{
		Types:  values.Types{"Layer 1", "Layer 2"},
		Params: values.NameValues{"port=7036", "options=-x", "port=7037"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	options := config.Get[map[string]any](cf, cf.Join("config", "options"))
	if want := "Layer 1,Layer 2"; options["type"] != want {
		t.Errorf("type = %#v, want %q", options["type"], want)
	}
	if options["port"] != "7037" || options["options"] != "-x" {
		t.Errorf("named options = %#v, want port=7037 and options=-x", options)
	}

	// setting again replaces by key and keeps the others
	i.SetConfig(cf)
	cf, err = values.Set(i, values.Values{Types: values.Types{"Layer 1"}, Params: values.NameValues{"port=1"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	options = config.Get[map[string]any](cf, cf.Join("config", "options"))
	if want := "Layer 1,Layer 2"; options["type"] != want {
		t.Errorf("type = %#v, want %q", options["type"], want)
	}
	if options["port"] != "1" || options["options"] != "-x" {
		t.Errorf("named options = %#v, want port=1 and options=-x", options)
	}
}

func TestSetProfileIncludesOnly(t *testing.T) {
	i := newProfileInstance(t, nil)
	cf, err := values.Set(i, values.Values{Includes: values.Includes{"100": "example.xml"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	options := config.Get[map[string]any](cf, "config.options", config.NoExpand())
	if options["include"] != "100:example.xml" {
		t.Errorf("include = %#v, want 100:example.xml", options["include"])
	}
}

func TestSetProfileRejectsInvalidOrUnencodableValues(t *testing.T) {
	i := newProfileInstance(t, map[string]any{"site": "original"})
	for _, test := range []struct {
		name string
		vals values.Values
		key  config.KeyFile
	}{
		{name: "invalid option", vals: values.Values{Params: values.NameValues{"site=new", "invalid"}}},
		{name: "empty option name", vals: values.Values{Params: values.NameValues{"=value"}}},
		{name: "secure parameter without key", vals: values.Values{SecureParams: values.SecureValues{{Name: "token", Secret: config.Secret("test-secret")}}}},
		{name: "secure env without key", vals: values.Values{SecureEnvs: values.SecureValues{{Name: "TOKEN", Secret: config.Secret("test-secret")}}}},
		{name: "unreadable key", vals: values.Values{SecureParams: values.SecureValues{{Name: "token", Secret: config.Secret("test-secret")}}}, key: config.KeyFile(t.TempDir() + "/missing")},
		{name: "secret variable without key", vals: values.Values{Variables: values.Variables{{Type: "secret", Name: "token", Value: "test-secret"}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cf, err := values.Set(i, test.vals, test.key)
			if err == nil || cf != nil {
				t.Fatal("Set should return an error and no configuration")
			}
			if profileOptions(i)["site"] != "original" {
				t.Fatal("failed Set changed the original profile")
			}
			if strings.Contains(err.Error(), "test-secret") {
				t.Fatal("error contains the plaintext secret")
			}
		})
	}
}

func TestUnsetProfileKeepsOptionTypes(t *testing.T) {
	i := newProfileInstance(t, map[string]any{"insecure": true, "type": "a,b,ab", "param": "port=1,portrange=2", "port": "8080"})

	if !values.Unset(i, values.UnsetConfigValues{Types: values.UnsetValues{"a"}, Keys: values.UnsetValues{"port"}}) {
		t.Fatal("Unset reported no change")
	}
	options := profileOptions(i)
	if got, ok := options["insecure"].(bool); !ok || !got {
		t.Errorf("insecure = %#v, want bool true", options["insecure"])
	}
	if want := "b,ab"; options["type"] != want {
		t.Errorf("type = %#v, want %q", options["type"], want)
	}
	if want := "portrange=2"; options["param"] != want {
		t.Errorf("param = %#v, want %q", options["param"], want)
	}
	if _, found := options["port"]; found {
		t.Error("Unset left the named port option behind")
	}

	// unsetting from an instance without any options must not panic
	j := geneos.ParseComponent("profile").New("empty-profile")
	if values.Unset(j, values.UnsetConfigValues{Types: values.UnsetValues{"a"}}) {
		t.Error("Unset of a missing option reported a change")
	}
}

func TestSetReturnsInvalidParameterErrors(t *testing.T) {
	geneostest.NewRoot(t, "", "netprobe")
	i, err := instance.Add(geneos.LOCAL, &san.Component, "san-params", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}

	cf, err := values.Set(i, values.Values{Params: values.NameValues{"valid=1", "not a setting"}}, "")
	if err == nil {
		t.Fatal("Set accepted an invalid parameter")
	}
	if cf != nil {
		t.Error("Set returned a config with an error")
	}

	// secure parameters without a keyfile are an error, not ignored
	if _, err = values.Set(i, values.Values{SecureParams: values.SecureValues{{Name: "password"}}}, ""); err == nil {
		t.Error("Set accepted secure parameters without a keyfile")
	}

	// blank parameters are ignored
	if _, err = values.Set(i, values.Values{Params: values.NameValues{"", "ok=1"}}, ""); err != nil {
		t.Errorf("Set rejected a blank parameter: %v", err)
	}
}

func gatewaysOf(i geneos.Instance) []string {
	gateways, _ := values.GatewaysFromConfig(i.Config())
	return gateways
}

func TestSANGatewaysSetAndUnset(t *testing.T) {
	geneostest.NewRoot(t, "", "netprobe")
	i, err := instance.Add(geneos.LOCAL, &san.Component, "san-gateways", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}

	// start with the legacy map form, which must not be lost
	config.Set(i.Config(), values.GATEWAYS, map[string]any{"legacy": "7100"})

	cf, err := values.Set(i, values.Values{Gateways: values.Gateways{"host1:7039", "host2:7100", "host1:7038"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	i.SetConfig(cf)
	if got, want := gatewaysOf(i), []string{"legacy:7100", "host1:7039", "host2:7100", "host1:7038"}; !slices.Equal(got, want) {
		t.Fatalf("gateways = %v, want %v", got, want)
	}

	// a host removes all entries for that host only
	if !values.Unset(i, values.UnsetConfigValues{Gateways: values.UnsetValues{"host1"}}) {
		t.Fatal("Unset reported no change")
	}
	if got, want := gatewaysOf(i), []string{"legacy:7100", "host2:7100"}; !slices.Equal(got, want) {
		t.Fatalf("after unsetting host1, gateways = %v, want %v", got, want)
	}

	// a name that matches nothing changes nothing
	if values.Unset(i, values.UnsetConfigValues{Gateways: values.UnsetValues{"nothere"}}) {
		t.Error("Unset of a missing gateway reported a change")
	}
	if got, want := gatewaysOf(i), []string{"legacy:7100", "host2:7100"}; !slices.Equal(got, want) {
		t.Fatalf("after unsetting a missing gateway, gateways = %v, want %v", got, want)
	}

	// a full entry removes only that entry
	values.Unset(i, values.UnsetConfigValues{Gateways: values.UnsetValues{"host2:7100"}})
	if got, want := gatewaysOf(i), []string{"legacy:7100"}; !slices.Equal(got, want) {
		t.Fatalf("gateways = %v, want %v", got, want)
	}
}

func TestSetManagedEntityIndexesAreUnique(t *testing.T) {
	geneostest.NewRoot(t, "", "netprobe")
	i, err := instance.Add(geneos.LOCAL, &san.Component, "san-entities", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}

	set := func(types ...string) {
		t.Helper()
		cf, err := values.Set(i, values.Values{Types: types}, "")
		if err != nil {
			t.Fatal(err)
		}
		i.SetConfig(cf)
		// save and reload, as each command line invocation does
		if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
			t.Fatal(resp.Err)
		}
		_ = i.Unload()
		if err := i.Load(); err != nil {
			t.Fatal(err)
		}
	}
	names := func() []string {
		entities := config.Get[map[string]map[string]any](i.Config(), values.MANAGED_ENTITIES)
		var names []string
		for _, e := range entities {
			names = append(names, fmt.Sprint(e[values.MAPKEY_NAME]))
		}
		slices.Sort(names)
		return names
	}

	set("one/type", "two/type", "three/type")
	if got, want := names(), []string{"one", "three", "two"}; !slices.Equal(got, want) {
		t.Fatalf("entities = %v, want %v", got, want)
	}

	// remove an entity that is not the last, the next new entity must
	// not take the place of an existing one
	values.Unset(i, values.UnsetConfigValues{Entities: values.UnsetValues{"two"}})
	set("four/type")
	if got, want := names(), []string{"four", "one", "three"}; !slices.Equal(got, want) {
		t.Fatalf("entities after removal = %v, want %v", got, want)
	}
}

func TestSetVariablesWithLegacyValuesKeepsThem(t *testing.T) {
	geneostest.NewRoot(t, "", "netprobe")
	i, err := instance.Add(geneos.LOCAL, &san.Component, "san-variables", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	config.Set(i.Config(), values.VARIABLES, map[string]string{"old": "integer:2"})

	cf, err := values.Set(i, values.Values{Variables: values.Variables{{Type: "string", Name: "new", Value: "x"}}}, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := config.Lookup[any](cf, values.VARIABLES, config.NoExpand())
	got, _ := values.NormaliseVars(raw)
	want := []values.Variable{{Type: "integer", Name: "old", Value: "2"}, {Type: "string", Name: "new", Value: "x"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("variables = %#v, want %#v", got, want)
	}
}
