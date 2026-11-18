package geneos

import (
	"slices"
	"strings"
	"testing"
)

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

func TestUnrollProbesInheritsGroupDefaults(t *testing.T) {
	port, other := 7036, 9999
	secure, timezone := "secure", "UTC"
	maxLog, cache := 50, true
	enabled := false

	probes := &Probes{ProbeGroups: []ProbeGroup{{
		Name: "outer",
		ProbeInfo: ProbeInfo{
			Port:               &port,
			ConnectionSecurity: &secure,
			Comment:            "group comment must not be inherited",
			ProbeInfoWithoutPort: ProbeInfoWithoutPort{
				MaxLogFileSize:    &maxLog,
				Timezone:          &timezone,
				CacheProcessNames: &cache,
				Comment:           "nor this one",
			},
		},
		Probes: []Probe{
			{Name: "inherits", Hostname: "h1"},
			{Name: "overrides", Hostname: "h2", ProbeInfo: ProbeInfo{Port: &other}},
			{Name: "disabled", Hostname: "h3", Disabled: &enabled},
		},
		FloatingProbes: []FloatingProbe{{Name: "floating"}},
		ProbeGroups: []ProbeGroup{{
			Name:   "inner",
			Probes: []Probe{{Name: "nested", Hostname: "h4"}},
		}},
	}}}

	got := UnrollProbes(probes)

	for _, name := range []string{"inherits", "nested"} {
		p, ok := got[name]
		if !ok {
			t.Fatalf("probe %q missing from %v", name, got)
		}
		if p.Port == nil || *p.Port != port {
			t.Errorf("%s: Port = %v, want %d", name, p.Port, port)
		}
		if p.ConnectionSecurity == nil || *p.ConnectionSecurity != secure {
			t.Errorf("%s: ConnectionSecurity = %v, want %q", name, p.ConnectionSecurity, secure)
		}
		if p.MaxLogFileSize == nil || *p.MaxLogFileSize != maxLog {
			t.Errorf("%s: MaxLogFileSize = %v, want %d", name, p.MaxLogFileSize, maxLog)
		}
		if p.Timezone == nil || *p.Timezone != timezone {
			t.Errorf("%s: Timezone = %v, want %q", name, p.Timezone, timezone)
		}
		if p.CacheProcessNames == nil || !*p.CacheProcessNames {
			t.Errorf("%s: CacheProcessNames = %v, want true", name, p.CacheProcessNames)
		}
		if p.Comment != "" || p.ProbeInfo.Comment != "" || p.ProbeInfoWithoutPort.Comment != "" {
			t.Errorf("%s: comments were inherited: %+v", name, p)
		}
	}

	if p := got["overrides"]; p.Port == nil || *p.Port != other {
		t.Errorf("overrides: Port = %v, want %d", p.Port, other)
	}
	if p := got["floating"]; p.MaxLogFileSize == nil || *p.MaxLogFileSize != maxLog {
		t.Errorf("floating: MaxLogFileSize = %v, want %d", p.MaxLogFileSize, maxLog)
	}

	// the inherited values are copies, not shared with the group
	*got["inherits"].Port = 1
	if port != 7036 || *got["nested"].Port != 7036 {
		t.Error("inherited pointer is shared")
	}
}

func TestUnrollEntitiesInheritsFromGroup(t *testing.T) {
	entities := UnrollEntities(&ManagedEntities{ManagedEntityGroups: []ManagedEntityGroup{{
		Name: "group",
		ManagedEntityInfo: ManagedEntityInfo{
			Comment:    "not inherited",
			Attributes: []Attribute{{Name: "ENV", Value: "prod"}},
			Vars:       []Vars{{Name: "v", String: "s"}},
		},
		Entities: []ManagedEntity{{Name: "e1", Attributes: nil}},
	}}}, nil)

	e, ok := entities["e1"]
	if !ok {
		t.Fatalf("entity missing: %v", entities)
	}
	if len(e.Attributes) != 1 || e.Attributes[0].Name != "ENV" || len(e.Vars) != 1 {
		t.Errorf("attributes/vars not inherited: %+v", e.ManagedEntityInfo)
	}
	if e.ManagedEntityInfo.Comment != "" {
		t.Errorf("comment inherited: %q", e.ManagedEntityInfo.Comment)
	}
}

func TestUnrollRulesNestedKeys(t *testing.T) {
	disabled := true
	rules := &Rules{
		Rules: []Rule{{Name: "top"}, {Name: "off", Disabled: &disabled}},
		RuleGroups: []RuleGroup{
			{
				Name:  "a",
				Rules: []Rule{{Name: "r1"}, {Name: "off", Disabled: &disabled}},
				RuleGroups: []RuleGroup{{
					Name:  "b",
					Rules: []Rule{{Name: "r2"}},
					RuleGroups: []RuleGroup{{
						Name:  "c",
						Rules: []Rule{{Name: "r3"}},
					}},
				}, {
					Name:     "disabled",
					Disabled: &disabled,
					Rules:    []Rule{{Name: "hidden"}},
				}},
			},
			{Name: "off", Disabled: &disabled, Rules: []Rule{{Name: "hidden"}}},
		},
	}

	got := UnrollRules(rules)
	var keys []string
	for k, r := range got {
		keys = append(keys, k)
		if !strings.HasSuffix(k, r.Name) {
			t.Errorf("key %q does not end with rule name %q", k, r.Name)
		}
	}
	slices.Sort(keys)
	want := []string{"a > b > c > r3", "a > b > r2", "a > r1", "top"}
	if !slices.Equal(keys, want) {
		t.Errorf("keys = %q, want %q", keys, want)
	}
	if got := UnrollRules(nil); got == nil || len(got) != 0 {
		t.Errorf("UnrollRules(nil) = %v", got)
	}
}

func TestGetPlugin(t *testing.T) {
	if got := GetPlugin(nil); got != nil {
		t.Errorf("GetPlugin(nil) = %v", got)
	}
	// an empty plugin has no plugin, it must not return the comment
	if got := GetPlugin(&Plugin{}); got != nil {
		t.Errorf("GetPlugin(empty) = %#v, want nil", got)
	}
	if got := GetPlugin(&Plugin{Comment: "comment"}); got != nil {
		t.Errorf("GetPlugin(comment only) = %#v, want nil", got)
	}
	cpu := &CPUPlugin{}
	if got := GetPlugin(&Plugin{Comment: "comment", CPU: cpu}); got != any(cpu) {
		t.Errorf("GetPlugin(cpu) = %#v, want the CPU plugin", got)
	}
}
