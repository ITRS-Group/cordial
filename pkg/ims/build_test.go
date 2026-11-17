package ims

import (
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestBuildIncident(t *testing.T) {
	cf := config.New()
	config.Set(cf, "defaults", []any{map[string]any{"set": map[string]any{"a": "default", "b": "default", "__profile": "special"}}})
	config.Set(cf, "profiles.special", []any{map[string]any{"set": map[string]any{"b": "profile"}}})
	config.Set(cf, "profiles.other", []any{map[string]any{"set": map[string]any{"b": "other"}}})

	tests := []struct {
		name    string
		profile string
		want    string
	}{
		{"profile from defaults", "", "profile"},
		{"explicit profile wins", "other", "other"},
		{"undefined profile only warns", "missing", "default"},
	}
	for _, test := range tests {
		incident := Values{}
		if err := BuildIncident(cf, incident, test.profile, "__profile"); err != nil {
			t.Errorf("%s: %v", test.name, err)
			continue
		}
		if incident["a"] != "default" || incident["b"] != test.want {
			t.Errorf("%s: incident = %v, want b=%q", test.name, incident, test.want)
		}
	}

	t.Run("no defaults or profiles", func(t *testing.T) {
		incident := Values{}
		if err := BuildIncident(config.New(), incident, "", "__profile"); err != nil || len(incident) != 0 {
			t.Errorf("err = %v, incident = %v", err, incident)
		}
	})

	t.Run("invalid configuration is an error", func(t *testing.T) {
		bad := config.New()
		config.Set(bad, "defaults", "not a list of groups")
		if err := BuildIncident(bad, Values{}, "", "__profile"); err == nil || !strings.Contains(err.Error(), "defaults") {
			t.Errorf("defaults: err = %v", err)
		}
		bad = config.New()
		config.Set(bad, "profiles.broken", "not a list of groups")
		if err := BuildIncident(bad, Values{}, "broken", "__profile"); err == nil || !strings.Contains(err.Error(), "broken") {
			t.Errorf("profile: err = %v", err)
		}
	})
}

func TestExpansionFunctions(t *testing.T) {
	t.Setenv("CORDIAL_TEST_SEVERITY", "CRITICAL")
	t.Setenv("CORDIAL_TEST_HOST", "  host1.example.com ")

	t.Run("match and nomatch", func(t *testing.T) {
		tests := []struct {
			fn      func(map[string]any, string, bool, ...any) (string, error)
			input   string
			want    string
			wantErr bool
		}{
			{matchPrefix, "match:CORDIAL_TEST_SEVERITY:^CRIT", "true", false},
			{matchPrefix, "match:CORDIAL_TEST_SEVERITY:OK", "false", false},
			{matchPrefix, "match:CORDIAL_TEST_SEVERITY:a:b|CRITICAL", "true", false},
			{matchPrefix, "match:CORDIAL_TEST_UNSET:.*", "false", false},
			{noMatchPrefix, "nomatch:CORDIAL_TEST_SEVERITY:OK", "true", false},
			{noMatchPrefix, "nomatch:CORDIAL_TEST_SEVERITY:CRIT", "false", false},
			{noMatchPrefix, "nomatch:CORDIAL_TEST_UNSET:.*", "false", false},
			{matchPrefix, "match:CORDIAL_TEST_SEVERITY:(", "false", true},
			{matchPrefix, "match:CORDIAL_TEST_SEVERITY", "false", true},
			{noMatchPrefix, "nomatch::x", "false", true},
		}
		for _, test := range tests {
			got, err := test.fn(nil, test.input, false)
			if got != test.want || (err != nil) != test.wantErr {
				t.Errorf("%s = %q, %v, want %q (error %t)", test.input, got, err, test.want, test.wantErr)
			}
		}
	})

	t.Run("replace", func(t *testing.T) {
		tests := []struct {
			input   string
			want    string
			wantErr bool
		}{
			{"replace:CORDIAL_TEST_HOST:/\\.example\\.com//", "  host1 ", false},
			{"replace:CORDIAL_TEST_UNSET:/a/b/", "", false},
			{"replace:CORDIAL_TEST_HOST:/(", "  host1.example.com ", true},
			{"replace:CORDIAL_TEST_HOST:/a/b/c", "  host1.example.com ", true},
			{"replace:CORDIAL_TEST_HOST:/[/x/", "  host1.example.com ", true},
			{"replace:", "", true},
		}
		for _, test := range tests {
			got, err := replacePrefix(nil, test.input, false)
			if got != test.want || (err != nil) != test.wantErr {
				t.Errorf("%s = %q, %v, want %q (error %t)", test.input, got, err, test.want, test.wantErr)
			}
		}
		if got, _ := replacePrefix(nil, "replace:CORDIAL_TEST_HOST:/\\.example\\.com//", true); got != "host1" {
			t.Errorf("trimmed result = %q", got)
		}
	})

	t.Run("select", func(t *testing.T) {
		t.Setenv("CORDIAL_TEST_A", "a")
		t.Setenv("CORDIAL_TEST_B", "b")
		for input, want := range map[string]string{
			"select:CORDIAL_TEST_A+CORDIAL_TEST_B:default":  "ab",
			"select:CORDIAL_TEST_A-CORDIAL_TEST_B:default":  "a-b",
			"select:CORDIAL_TEST_A++CORDIAL_TEST_B:default": "a+b",
			"select:CORDIAL_TEST_UNSET:":                    "",
		} {
			if got, err := selectPrefix(nil, input, false); err != nil || got != want {
				t.Errorf("%s = %q, %v, want %q", input, got, err, want)
			}
		}
	})

	t.Run("field", func(t *testing.T) {
		incident := Values{"a": " x ", "b": "y"}
		for input, want := range map[string]string{
			"field:a:default":   " x ",
			"field:c:b:default": "y",
			"field:c:default":   "default",
			"field:c:":          "",
		} {
			if got, err := fieldPrefix(nil, input, false, incident); err != nil || got != want {
				t.Errorf("%s = %q, %v, want %q", input, got, err, want)
			}
		}
		if got, _ := fieldPrefix(nil, "field:a:d", true, incident); got != "x" {
			t.Errorf("trimmed = %q", got)
		}
		if got, _ := fieldPrefix(nil, "field:a:d", false); got != "d" {
			t.Errorf("without incident = %q", got)
		}
	})
}
