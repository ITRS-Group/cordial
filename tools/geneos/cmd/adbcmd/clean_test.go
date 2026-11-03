package adbcmd

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/xpath"
)

func TestUpdateDataitemXpathTypes(t *testing.T) {
	typeValue := func(value string) *string { return &value }
	tests := []struct {
		name     string
		original *string
		actions  optionSlice
		want     *string
	}{
		{name: "default preserves non-empty", original: typeValue("System"), actions: optionSlice{"remove-empty"}, want: typeValue("System")},
		{name: "default removes empty", original: typeValue(""), actions: optionSlice{"remove-empty"}},
		{name: "default preserves absent", actions: optionSlice{"remove-empty"}},
		{name: "keep preserves non-empty", original: typeValue("System"), actions: optionSlice{"keep"}, want: typeValue("System")},
		{name: "keep preserves empty", original: typeValue(""), actions: optionSlice{"keep"}, want: typeValue("")},
		{name: "keep preserves absent", actions: optionSlice{"keep"}},
		{name: "no actions preserves non-empty", original: typeValue("System"), want: typeValue("System")},
		{name: "no actions preserves empty", original: typeValue(""), want: typeValue("")},
		{name: "non-matching removal preserves", original: typeValue("System"), actions: optionSlice{"remove:Application*"}, want: typeValue("System")},
		{name: "matching removal removes", original: typeValue("System"), actions: optionSlice{"remove:Sys*"}},
		{name: "remove-all removes", original: typeValue("System"), actions: optionSlice{"remove-all"}},
		{name: "set creates absent", actions: optionSlice{"set:System"}, want: typeValue("System")},
		{name: "set-empty creates empty", actions: optionSlice{"set-empty"}, want: typeValue("")},
		{name: "replacement preserves absence", actions: optionSlice{"replace:/^/Prefix/"}},
		{name: "unknown action preserves", original: typeValue("System"), actions: optionSlice{"unknown"}, want: typeValue("System")},
		{name: "set then replace", original: typeValue("System"), actions: optionSlice{"set:Demo-Application", "replace:/^Demo-/Production-/"}, want: typeValue("Production-Application")},
		{name: "chained replacements", original: typeValue("Demo-System"), actions: optionSlice{"replace:/^Demo-/Production-/", "replace:/Production/Live/"}, want: typeValue("Live-System")},
		{name: "set then matching removal", original: typeValue("System"), actions: optionSlice{"set:Application", "remove:App*"}},
		{name: "set-empty then remove-empty", original: typeValue("System"), actions: optionSlice{"set-empty", "remove-empty"}},
		{name: "remove then keep stays absent", original: typeValue("System"), actions: optionSlice{"remove-all", "keep"}},
		{name: "remove then set", original: typeValue("System"), actions: optionSlice{"remove-all", "set:Application"}, want: typeValue("Application")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var originalValue string
			if test.original != nil {
				originalValue = *test.original
			}
			input := &xpath.XPath{Sampler: &xpath.Sampler{Name: "CPU", Type: test.original}}
			got := updateDataitemXpath(input, cleaningOptions{Types: test.actions})
			if got.Sampler == nil {
				t.Fatal("Sampler was removed")
			}
			actual := got.Sampler.Type
			if (actual == nil) != (test.want == nil) {
				t.Fatalf("Type present = %t, want %t", actual != nil, test.want != nil)
			}
			if actual != nil && *actual != *test.want {
				t.Fatalf("Type = %q, want %q", *actual, *test.want)
			}
			if test.original != nil && *test.original != originalValue {
				t.Fatalf("original Type was modified: %q, want %q", *test.original, originalValue)
			}
		})
	}

	t.Run("no Type action does not create Sampler", func(t *testing.T) {
		got := updateDataitemXpath(&xpath.XPath{}, cleaningOptions{})
		if got.Sampler != nil {
			t.Fatal("Sampler was created without an action")
		}
	})
	t.Run("set Type creates Sampler", func(t *testing.T) {
		got := updateDataitemXpath(&xpath.XPath{}, cleaningOptions{Types: optionSlice{"set:System"}})
		if got.Sampler == nil || got.Sampler.Type == nil || *got.Sampler.Type != "System" {
			t.Fatal("Sampler with Type System was not created")
		}
	})
}

func TestUpdateDataitemXpathAttributes(t *testing.T) {
	t.Run("add initializes nil Attribute map", func(t *testing.T) {
		input := &xpath.XPath{Entity: &xpath.Entity{Name: "server"}}
		got := updateDataitemXpath(input, cleaningOptions{
			Attributes: optionSlice{"add:Location=London"},
		})
		if got.Entity == nil {
			t.Fatal("Managed Entity was removed")
		}
		if got.Entity.Attributes["Location"] != "London" {
			t.Fatalf("Attributes = %v, want Location=London", got.Entity.Attributes)
		}
	})

	t.Run("Attribute action does not create missing Managed Entity", func(t *testing.T) {
		got := updateDataitemXpath(&xpath.XPath{}, cleaningOptions{
			Attributes: optionSlice{"add:Location=London"},
		})
		if got.Entity != nil {
			t.Fatalf("Managed Entity was created: %#v", got.Entity)
		}
	})

	t.Run("explicit Managed Entity action allows Attribute addition", func(t *testing.T) {
		got := updateDataitemXpath(&xpath.XPath{}, cleaningOptions{
			Entities:   optionSlice{"set:server"},
			Attributes: optionSlice{"add:Location=London"},
		})
		if got.Entity == nil {
			t.Fatal("Managed Entity was not created")
		}
		if got.Entity.Name != "server" || got.Entity.Attributes["Location"] != "London" {
			t.Fatalf("Managed Entity = %#v, want server with Location=London", got.Entity)
		}
	})
}

func TestCleanActivedashboardModifierAlias(t *testing.T) {
	originalFilter := cleanFilterXPath
	originalOptions := cleanOptions
	originalSkipModifiers := cleanCmdSkipModifiers
	originalLog := log
	defer func() {
		cleanFilterXPath = originalFilter
		cleanOptions = originalOptions
		cleanCmdSkipModifiers = originalSkipModifiers
		log = originalLog
	}()

	cleanCmdSkipModifiers = false
	log = slog.Default()
	dataitemPath := `/geneos/gateway[(@name="Other")]/directory/probe[(@name="Probe")]/managedEntity[(@name="Entity")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]`
	tests := []struct {
		name          string
		filterGateway string
		gatewayAction optionSlice
		paths         []string
		wantPaths     []string
		wantAlias     string
	}{
		{
			name:          "filtered-out path preserves alias",
			filterGateway: "Target",
			gatewayAction: optionSlice{"set:Updated"},
			paths:         []string{dataitemPath},
			wantPaths:     []string{dataitemPath},
			wantAlias:     "saved-alias",
		},
		{
			name:          "changed path clears alias",
			filterGateway: "Other",
			gatewayAction: optionSlice{"set:Updated"},
			paths:         []string{dataitemPath},
			wantPaths:     []string{`/geneos/gateway[(@name="Updated")]/directory/probe[(@name="Probe")]/managedEntity[(@name="Entity")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]`},
			wantAlias:     EMPTY_STRING,
		},
		{
			name:          "unchanged matched path preserves alias",
			filterGateway: "Other",
			gatewayAction: optionSlice{"keep"},
			paths:         []string{dataitemPath},
			wantPaths:     []string{dataitemPath},
			wantAlias:     "saved-alias",
		},
		{
			name:          "changed path clears shared alias when another path is filtered out",
			filterGateway: "Other",
			gatewayAction: optionSlice{"set:Updated"},
			paths: []string{
				dataitemPath,
				strings.Replace(dataitemPath, `name="Other"`, `name="Target"`, 1),
			},
			wantPaths: []string{
				`/geneos/gateway[(@name="Updated")]/directory/probe[(@name="Probe")]/managedEntity[(@name="Entity")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]`,
				strings.Replace(dataitemPath, `name="Other"`, `name="Target"`, 1),
			},
			wantAlias: EMPTY_STRING,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanFilterXPath = &xpath.XPath{Gateway: &xpath.Gateway{Name: test.filterGateway}}
			cleanOptions = cleaningOptions{Gateways: test.gatewayAction}
			dataitems := make([]string, len(test.paths))
			wantDataitems := make([]string, len(test.wantPaths))
			for i, path := range test.paths {
				dataitems[i] = "dataitem(" + path + ")"
			}
			for i, path := range test.wantPaths {
				wantDataitems[i] = "dataitem(" + path + ")"
			}
			line := strings.Join([]string{
				dataItemExprNodeClass,
				"7.0",
				"$0$",
				"Value",
				strings.Join(dataitems, ""),
				"saved-alias",
				"1",
			}, string(fieldSeparator))

			var output strings.Builder
			if err := cleanActivedashboard(strings.NewReader(line), &output); err != nil {
				t.Fatalf("cleanActivedashboard() error = %v", err)
			}
			parts := strings.Split(strings.TrimSuffix(output.String(), "\n"), string(fieldSeparator))
			wantURLTarget := strings.Join(wantDataitems, "")
			if parts[4] != wantURLTarget {
				t.Errorf("URLTarget = %q, want %q", parts[4], wantURLTarget)
			}
			if parts[5] != test.wantAlias {
				t.Errorf("URLAlias = %q, want %q", parts[5], test.wantAlias)
			}
		})
	}
}
