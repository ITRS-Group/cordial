package client

import (
	"testing"

	"github.com/itrs-group/cordial/integrations/servicenow2/internal/snow"
	"github.com/itrs-group/cordial/pkg/config"
)

func TestProcessActionGroupAppliesMutationsAndBreak(t *testing.T) {
	incident := snow.Record{"remove": "old", "keep": "value"}
	ag := ActionGroup{
		If:    []string{"true"},
		Set:   map[string]string{"added": "new"},
		Unset: []string{"remove"},
		Break: []string{"true"},
	}

	if !processActionGroup(config.New(), ag, incident) {
		t.Fatal("processActionGroup() = false, want break")
	}
	if incident["added"] != "new" || incident["keep"] != "value" {
		t.Errorf("incident after action = %#v", incident)
	}
	if _, ok := incident["remove"]; ok {
		t.Errorf("unset field remains in incident: %#v", incident)
	}
}

func TestProcessActionGroupFalseConditionSkipsMutations(t *testing.T) {
	incident := snow.Record{"keep": "value"}
	stop := processActionGroup(config.New(), ActionGroup{
		If:  []string{"false"},
		Set: map[string]string{"added": "new"},
	}, incident)
	if stop {
		t.Fatal("processActionGroup() stopped for a false condition")
	}
	if _, ok := incident["added"]; ok {
		t.Errorf("actions ran after a false condition: %#v", incident)
	}
}

func TestProcessActionGroupSubgroupBreakStopsParent(t *testing.T) {
	incident := snow.Record{}
	stop := processActionGroup(config.New(), ActionGroup{
		Subgroup: []ActionGroup{{
			Set:   map[string]string{"child": "ran"},
			Break: []string{"true"},
		}, {
			Set: map[string]string{"later": "must not run"},
		}},
	}, incident)
	if stop {
		t.Fatal("a subgroup break should terminate the subgroup, not the parent")
	}
	if incident["child"] != "ran" {
		t.Errorf("subgroup action did not run: %#v", incident)
	}
	if _, ok := incident["later"]; ok {
		t.Errorf("later subgroup ran after break: %#v", incident)
	}
}
