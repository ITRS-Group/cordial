package geneos

import "testing"

func TestUnrollProbesOptionalDisabled(t *testing.T) {
	falseValue := false
	trueValue := true
	got := UnrollProbes(&Probes{Probes: []Probe{
		{Name: "unset"},
		{Name: "enabled", Disabled: &falseValue},
		{Name: "disabled", Disabled: &trueValue},
	}})

	for _, name := range []string{"unset", "enabled"} {
		if _, ok := got[name]; !ok {
			t.Errorf("probe %q with unset or false disabled flag was omitted", name)
		}
	}
	if _, ok := got["disabled"]; ok {
		t.Error("probe with true disabled flag was included")
	}
}

func TestUnrollTypesOptionalDisabled(t *testing.T) {
	falseValue := false
	trueValue := true
	got := UnrollTypes(&Types{Types: []Type{
		{Name: "unset"},
		{Name: "enabled", Disabled: &falseValue},
		{Name: "disabled", Disabled: &trueValue},
	}})

	for _, name := range []string{"unset", "enabled"} {
		if _, ok := got[name]; !ok {
			t.Errorf("type %q with unset or false disabled flag was omitted", name)
		}
	}
	if _, ok := got["disabled"]; ok {
		t.Error("type with true disabled flag was included")
	}
}

func TestUnrollSamplersOptionalDisabled(t *testing.T) {
	falseValue := false
	trueValue := true
	got := UnrollSamplers(&Samplers{Samplers: []Sampler{
		{Name: "unset"},
		{Name: "enabled", Disabled: &falseValue},
		{Name: "disabled", Disabled: &trueValue},
	}})

	for _, name := range []string{"unset", "enabled"} {
		if _, ok := got[name]; !ok {
			t.Errorf("sampler %q with unset or false disabled flag was omitted", name)
		}
	}
	if _, ok := got["disabled"]; ok {
		t.Error("sampler with true disabled flag was included")
	}
}
