package values

import (
	"slices"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestGatewaysFromConfigAcceptsAllForms(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		want       []string
		wantLegacy bool
	}{
		{"slice", []string{"a:7039", "b:7100"}, []string{"a:7039", "b:7100"}, false},
		{"any slice", []any{"a:7039", 7, "b"}, []string{"a:7039", "b"}, false},
		{"legacy map", map[string]any{"b": "7100", "a": 7039, "c": nil}, []string{"a:7039", "b:7100", "c"}, true},
		{"legacy string map", map[string]string{"a": "7038"}, []string{"a:7038"}, true},
		{"comma separated", "a:1, b:2,,", []string{"a:1", "b:2"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cf := config.New()
			config.Set(cf, GATEWAYS, test.value)
			got, legacy := GatewaysFromConfig(cf)
			if !slices.Equal(got, test.want) || legacy != test.wantLegacy {
				t.Errorf("GatewaysFromConfig() = %v, %t, want %v, %t", got, legacy, test.want, test.wantLegacy)
			}
		})
	}

	if got, legacy := GatewaysFromConfig(config.New()); got != nil || legacy {
		t.Errorf("GatewaysFromConfig() with no gateways = %v, %t", got, legacy)
	}
}

func TestMigrateGateways(t *testing.T) {
	cf := config.New()
	config.Set(cf, GATEWAYS, map[string]any{"b": "7039", "a": "7100"})
	if !MigrateGateways(cf) {
		t.Fatal("MigrateGateways() did not change a legacy map")
	}
	got, legacy := GatewaysFromConfig(cf)
	if want := []string{"a:7100", "b:7039"}; !slices.Equal(got, want) || legacy {
		t.Errorf("after migration gateways = %v, legacy = %t, want %v", got, legacy, want)
	}
	if MigrateGateways(cf) {
		t.Error("MigrateGateways() changed an already migrated list")
	}
}

func TestTLSGatewayPorts(t *testing.T) {
	input := []string{"a:7039", "b:7038", "c:7100", "d", "[::1]:7039", "e:17039", "f:7039:7039"}
	tests := []struct {
		secure      bool
		want        []string
		wantChanged bool
	}{
		{true, []string{"a:7038", "b:7038", "c:7100", "d", "[::1]:7038", "e:17039", "f:7039:7038"}, true},
		{false, []string{"a:7039", "b:7039", "c:7100", "d", "[::1]:7039", "e:17039", "f:7039:7039"}, true},
	}
	for _, test := range tests {
		got, changed := TLSGatewayPorts(slices.Clone(input), test.secure)
		if !slices.Equal(got, test.want) || changed != test.wantChanged {
			t.Errorf("TLSGatewayPorts(secure=%t) = %v, %t, want %v, %t", test.secure, got, changed, test.want, test.wantChanged)
		}
	}

	// nothing to change, including no gateways
	for _, gateways := range [][]string{nil, {"c:7100", "d"}} {
		if got, changed := TLSGatewayPorts(gateways, true); changed || !slices.Equal(got, gateways) {
			t.Errorf("TLSGatewayPorts(%v, true) = %v, %t", gateways, got, changed)
		}
	}
	if _, changed := TLSGatewayPorts([]string{"a:7038"}, true); changed {
		t.Error("TLSGatewayPorts() moved a gateway already on the TLS port")
	}
	if got, _ := TLSGatewayPorts(nil, true); got == nil {
		t.Error("TLSGatewayPorts() returned a nil slice")
	}
}

func TestIncludesSetOnlyUsesNumericPriority(t *testing.T) {
	tests := []struct {
		input    string
		priority string
		path     string
	}{
		{"common.xml", "100", "common.xml"},
		{"20:site.xml", "20", "site.xml"},
		{"https://example.com/a.xml", "100", "https://example.com/a.xml"},
		{"30:https://example.com/a.xml", "30", "https://example.com/a.xml"},
		{`C:\geneos\a.xml`, "100", `C:\geneos\a.xml`},
		{`40:C:\geneos\a.xml`, "40", `C:\geneos\a.xml`},
		{":empty-priority.xml", "100", ":empty-priority.xml"},
	}
	for _, test := range tests {
		var includes Includes
		if err := includes.Set(test.input); err != nil {
			t.Fatal(err)
		}
		if got, ok := includes[test.priority]; !ok || got != test.path || len(includes) != 1 {
			t.Errorf("Set(%q) = %v, want %s:%s", test.input, includes, test.priority, test.path)
		}
	}
}

func TestVariablesSetRejectsInvalidType(t *testing.T) {
	var vars Variables
	if err := vars.Set("bogus:name=value"); err == nil {
		t.Fatal("Set accepted an invalid variable type")
	}
	if len(vars) != 0 {
		t.Errorf("an invalid variable was added: %v", vars)
	}
	if err := vars.Set("secret:name=value"); err != nil {
		t.Errorf("Set rejected the secret type: %v", err)
	}
}

func TestNormaliseVarsToleratesBadItems(t *testing.T) {
	// none of these may panic
	tests := []struct {
		name  string
		input any
		want  []Variable
	}{
		{"missing type", []any{map[string]any{"name": "a", "value": "1"}}, []Variable{{Type: "string", Name: "a", Value: "1"}}},
		{"missing value", []any{map[string]any{"name": "a"}}, []Variable{{Type: "string", Name: "a"}}},
		{"non-string value", []any{map[string]any{"name": "a", "value": 1}, map[string]any{"name": "b", "value": "2"}}, []Variable{{Type: "string", Name: "b", Value: "2"}}},
		{"slice of maps", []map[string]any{{"name": "a", "type": "integer", "value": "3"}}, []Variable{{Type: "integer", Name: "a", Value: "3"}}},
		{"keyed maps", map[string]any{"x": map[string]any{"name": "a", "value": "1"}, "y": "not a map-ish item", "z": 1}, []Variable{{Type: "string", Name: "a", Value: "1"}, {Type: "string", Name: "y", Value: "not a map-ish item"}}},
		{"unsupported", 42, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := NormaliseVars(test.input)
			if !slices.Equal(got, test.want) {
				t.Errorf("NormaliseVars() = %#v, want %#v", got, test.want)
			}
		})
	}
}
