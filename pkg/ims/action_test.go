package ims

import (
	"reflect"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestSelectPrefixKeepsLeadingEnvNameCharacters(t *testing.T) {
	// environment variable names that start with characters found in
	// "select:" must not be altered when the prefix is removed
	t.Setenv("setup_VALUE", "from-env")
	t.Setenv("CORDIAL_TEST_EMPTY", "")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"leading cutset characters", "select:setup_VALUE:default", "from-env"},
		{"first set wins", "select:CORDIAL_TEST_UNSET:setup_VALUE:default", "from-env"},
		{"empty treated as unset", "select:CORDIAL_TEST_EMPTY:default", "default"},
		{"default only", "select:CORDIAL_TEST_UNSET:", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := selectPrefix(nil, test.input, false)
			if err != nil || got != test.want {
				t.Errorf("selectPrefix(%q) = %q, %v, want %q", test.input, got, err, test.want)
			}
		})
	}
}

// the prefixes are passed to each expansion, so a configuration with
// no defaults of its own works and does not change it for anything else
func TestProcessActionGroupNeedsNoDefaultExpandOptions(t *testing.T) {
	t.Setenv("CORDIAL_TEST_SEV", "CRITICAL")

	cf := config.New()
	incident := Values{"short": "disk full"}
	ProcessActionGroup(cf, ActionGroup{
		If:  []string{"${match:CORDIAL_TEST_SEV:^CRIT}", "${nomatch:CORDIAL_TEST_SEV:^WARN}"},
		Set: map[string]string{"a": "${field:short:none}", "b": "${select:CORDIAL_TEST_SEV:x}", "c": "${replace:CORDIAL_TEST_SEV:/CRIT/ok/}"},
		Subgroup: []ActionGroup{
			{Set: map[string]string{"d": "${field:a:none}"}},
		},
	}, incident)

	want := Values{"short": "disk full", "a": "disk full", "b": "CRITICAL", "c": "okICAL", "d": "disk full"}
	if !reflect.DeepEqual(incident, want) {
		t.Errorf("incident = %v, want %v", incident, want)
	}
	if n := len(cf.DefaultExpandOptions()); n != 0 {
		t.Errorf("configuration has %d default expand options, want 0", n)
	}

	// each call has the fields of its own incident
	other := Values{"short": "other"}
	ProcessActionGroup(cf, ActionGroup{Set: map[string]string{"a": "${field:short:none}"}}, other)
	if other["a"] != "other" {
		t.Errorf("a = %q, want %q", other["a"], "other")
	}
}

func TestActionPrefixesDoNotLeakToOtherConfigurations(t *testing.T) {
	ProcessActionGroup(config.New(), ActionGroup{Set: map[string]string{"a": "${field:x:def}"}}, Values{})

	if got := config.Expand[string](config.New(), "${field:x:def}"); got == "def" {
		t.Fatal("a prefix registered for one configuration is visible to another")
	}

	incident := Values{}
	ProcessActionGroup(config.New(), ActionGroup{Set: map[string]string{"a": "${field:x:def}"}}, incident)
	if incident["a"] != "def" {
		t.Errorf("a = %q, want def", incident["a"])
	}
}
