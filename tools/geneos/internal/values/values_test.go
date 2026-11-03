package values

import (
	"reflect"
	"testing"
)

func TestVariablesSetParsesAndUpdatesByName(t *testing.T) {
	var got Variables
	for _, value := range []string{
		"name=value",
		"integer:count=2",
		"name=updated",
		"gateway/boolean:flag=true",
	} {
		if err := got.Set(value); err != nil {
			t.Fatalf("Set(%q) error = %v", value, err)
		}
	}
	want := Variables{
		{Type: "string", Name: "name", Value: "updated"},
		{Type: "integer", Name: "count", Value: "2"},
		{Type: "boolean", Name: "gateway/flag", Value: "true"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Variables = %#v, want %#v", got, want)
	}
	if got := got.String(); got != "string:name=updated,integer:count=2,boolean:gateway/flag=true" {
		t.Errorf("Variables.String() = %q", got)
	}
}

func TestNormaliseVarsFromLegacyMap(t *testing.T) {
	got, changed := NormaliseVars(map[string]string{
		"count": "integer:2",
		"label": "ready",
	})
	if !changed {
		t.Fatal("NormaliseVars() did not report converting the legacy map")
	}
	want := map[string]Variable{
		"count": {Type: "integer", Name: "count", Value: "2"},
		"label": {Type: "string", Name: "label", Value: "ready"},
	}
	if len(got) != len(want) {
		t.Fatalf("NormaliseVars() returned %d variables, want %d", len(got), len(want))
	}
	for _, variable := range got {
		if variable != want[variable.Name] {
			t.Errorf("normalized variable = %#v, want %#v", variable, want[variable.Name])
		}
	}
}

func TestNormaliseVarsFromLegacySlice(t *testing.T) {
	got, changed := NormaliseVars([]any{
		map[string]any{"type": "", "name": "state", "value": "open"},
		map[string]any{"type": "boolean", "name": "enabled", "value": "true"},
		map[string]any{"type": "string", "name": "", "value": "skip"},
		"not a variable",
	})
	if !changed {
		t.Fatal("NormaliseVars() did not report converting the legacy slice")
	}
	want := []Variable{
		{Type: "string", Name: "state", Value: "open"},
		{Type: "boolean", Name: "enabled", Value: "true"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NormaliseVars() = %#v, want %#v", got, want)
	}
}

func TestNormaliseVarsLeavesCurrentValuesUnchanged(t *testing.T) {
	input := []Variable{{Type: "string", Name: "state", Value: "open"}}
	got, changed := NormaliseVars(input)
	if changed {
		t.Fatal("NormaliseVars() reported changing current values")
	}
	if !reflect.DeepEqual(got, input) {
		t.Errorf("NormaliseVars() = %#v, want %#v", got, input)
	}
}

func TestNameValuesIncludesAndGateways(t *testing.T) {
	var pairs NameValues
	_ = pairs.Set("team=ops")
	_ = pairs.Set("region=west")
	if got := pairs.String(); got != "team=ops,region=west" {
		t.Errorf("NameValues.String() = %q", got)
	}

	var includes Includes
	_ = includes.Set("common.xml")
	_ = includes.Set("20:site.xml")
	if includes["100"] != "common.xml" || includes["20"] != "site.xml" {
		t.Errorf("Includes.Set() = %#v", includes)
	}

	var gateways Gateways
	_ = gateways.Set("gateway-a:7039")
	_ = gateways.Set("gateway-b:7039")
	if got := gateways.String(); got != "gateway-a:7039,gateway-b:7039" {
		t.Errorf("Gateways.String() = %q", got)
	}
}
