/*
Copyright © 2025 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"encoding/json"
	"testing"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
)

func TestClientCommand(t *testing.T) {
	// Test that the client command is properly initialized
	if clientCmd == nil {
		t.Fatal("Expected clientCmd to be initialized, got nil")
	}

	if clientCmd.Use != "client [FLAGS] [field=value ...]" {
		t.Errorf("Expected clientCmd.Use to be 'client [FLAGS] [field=value ...]', got %q", clientCmd.Use)
	}

	if clientCmd.Short != "Create or update a ServiceNow incident" {
		t.Errorf("Expected clientCmd.Short to be 'Create or update a ServiceNow incident', got %q", clientCmd.Short)
	}

	// Test that the command has a RunE function
	if clientCmd.RunE == nil {
		t.Error("Expected clientCmd to have a RunE function")
	}

	// Test that SilenceUsage is set
	if !clientCmd.SilenceUsage {
		t.Error("Expected clientCmd.SilenceUsage to be true")
	}
}

func TestClientFlags(t *testing.T) {
	// Test that client command flags are properly set up
	flags := clientCmd.Flags()

	// Check for profile flag
	profileFlag := flags.Lookup("profile")
	if profileFlag == nil {
		t.Error("Expected 'profile' flag to be defined")
	} else {
		if profileFlag.Shorthand != "p" {
			t.Errorf("Expected profile flag shorthand to be 'p', got %q", profileFlag.Shorthand)
		}
		if profileFlag.Usage != "profile to use for field creation" {
			t.Errorf("Expected profile flag usage to be 'profile to use for field creation', got %q", profileFlag.Usage)
		}
	}

	// Check for table flag
	tableFlag := flags.Lookup("table")
	if tableFlag == nil {
		t.Error("Expected 'table' flag to be defined")
	} else {
		if tableFlag.Shorthand != "t" {
			t.Errorf("Expected table flag shorthand to be 't', got %q", tableFlag.Shorthand)
		}
		if tableFlag.Usage != "servicenow table, defaults typically to incident" {
			t.Errorf("Expected table flag usage to be 'servicenow table, defaults typically to incident', got %q", tableFlag.Usage)
		}
	}

	// Check for quiet flag
	quietFlag := flags.Lookup("quiet")
	if quietFlag == nil {
		t.Error("Expected 'quiet' flag to be defined")
	} else {
		if quietFlag.Shorthand != "q" {
			t.Errorf("Expected quiet flag shorthand to be 'q', got %q", quietFlag.Shorthand)
		}
		if quietFlag.Usage != "quiet mode. supress all non-error messages" {
			t.Errorf("Expected quiet flag usage to be 'quiet mode. supress all non-error messages', got %q", quietFlag.Usage)
		}
	}

	// Test that flags are not sorted
	if clientCmd.Flags().SortFlags {
		t.Error("Expected client command flags to not be sorted")
	}
}

func TestClientCommandAddedToRoot(t *testing.T) {
	// Test that the client command is added to the root command
	found := false
	for _, subCmd := range cmd.RootCmd.Commands() {
		if subCmd.Name() == "client" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected client command to be added to root command")
	}
}

func TestGlobalFlags(t *testing.T) {
	// Test initial values of global flag variables
	if clientCmdProfile != "" {
		t.Errorf("Expected clientCmdProfile to be empty initially, got %q", clientCmdProfile)
	}

	if clientCmdTable != "" {
		t.Errorf("Expected clientCmdTable to be empty initially, got %q", clientCmdTable)
	}

	if clientCmdQuiet {
		t.Error("Expected clientCmdQuiet to be false initially")
	}
}

func TestActionGroup(t *testing.T) {
	// Test the ActionGroup struct
	actionGroup := ActionGroup{
		If:   []string{"state=1"},
		Set:  map[string]string{"urgency": "2", "impact": "2"},
		Unset: []string{"caller_id"},
		Subgroup: []ActionGroup{
			{
				If:  []string{"priority=1"},
				Set: map[string]string{"state": "2"},
			},
		},
		Break: []string{"state=6"},
	}

	// Test JSON marshaling/unmarshaling
	jsonData, err := json.Marshal(actionGroup)
	if err != nil {
		t.Fatalf("Failed to marshal ActionGroup to JSON: %v", err)
	}

	var unmarshaledGroup ActionGroup
	err = json.Unmarshal(jsonData, &unmarshaledGroup)
	if err != nil {
		t.Fatalf("Failed to unmarshal ActionGroup from JSON: %v", err)
	}

	// Test that data was preserved
	if len(unmarshaledGroup.If) != 1 || unmarshaledGroup.If[0] != "state=1" {
		t.Errorf("Expected If to be ['state=1'], got %v", unmarshaledGroup.If)
	}

	if unmarshaledGroup.Set["urgency"] != "2" {
		t.Errorf("Expected Set[urgency] to be '2', got %v", unmarshaledGroup.Set["urgency"])
	}

	if len(unmarshaledGroup.Unset) != 1 || unmarshaledGroup.Unset[0] != "caller_id" {
		t.Errorf("Expected Unset to be ['caller_id'], got %v", unmarshaledGroup.Unset)
	}

	if len(unmarshaledGroup.Subgroup) != 1 {
		t.Errorf("Expected 1 subgroup, got %d", len(unmarshaledGroup.Subgroup))
	}

	if len(unmarshaledGroup.Break) != 1 || unmarshaledGroup.Break[0] != "state=6" {
		t.Errorf("Expected Break to be ['state=6'], got %v", unmarshaledGroup.Break)
	}
}

func TestActionGroupEmpty(t *testing.T) {
	// Test empty ActionGroup
	actionGroup := ActionGroup{}

	// Test JSON marshaling/unmarshaling of empty struct
	jsonData, err := json.Marshal(actionGroup)
	if err != nil {
		t.Fatalf("Failed to marshal empty ActionGroup to JSON: %v", err)
	}

	var unmarshaledGroup ActionGroup
	err = json.Unmarshal(jsonData, &unmarshaledGroup)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty ActionGroup from JSON: %v", err)
	}

	// All fields should be zero values
	if len(unmarshaledGroup.If) != 0 {
		t.Errorf("Expected empty If slice, got %v", unmarshaledGroup.If)
	}

	if len(unmarshaledGroup.Set) != 0 {
		t.Errorf("Expected empty Set map, got %v", unmarshaledGroup.Set)
	}

	if len(unmarshaledGroup.Unset) != 0 {
		t.Errorf("Expected empty Unset slice, got %v", unmarshaledGroup.Unset)
	}

	if len(unmarshaledGroup.Subgroup) != 0 {
		t.Errorf("Expected empty Subgroup slice, got %v", unmarshaledGroup.Subgroup)
	}

	if len(unmarshaledGroup.Break) != 0 {
		t.Errorf("Expected empty Break slice, got %v", unmarshaledGroup.Break)
	}
}

func TestActionGroupJSONOmitEmpty(t *testing.T) {
	// Test that empty fields are omitted from JSON output
	actionGroup := ActionGroup{
		Set: map[string]string{"urgency": "3"},
		// All other fields are empty and should be omitted
	}

	jsonData, err := json.Marshal(actionGroup)
	if err != nil {
		t.Fatalf("Failed to marshal ActionGroup to JSON: %v", err)
	}

	jsonStr := string(jsonData)

	// Should only contain "set" field
	if !contains(jsonStr, `"set"`) {
		t.Error("Expected JSON to contain 'set' field")
	}

	// Should not contain empty fields due to omitempty tags
	if contains(jsonStr, `"if"`) {
		t.Error("Expected JSON to not contain empty 'if' field")
	}

	if contains(jsonStr, `"unset"`) {
		t.Error("Expected JSON to not contain empty 'unset' field")
	}

	if contains(jsonStr, `"subgroup"`) {
		t.Error("Expected JSON to not contain empty 'subgroup' field")
	}

	if contains(jsonStr, `"break"`) {
		t.Error("Expected JSON to not contain empty 'break' field")
	}
}

func TestComplexActionGroup(t *testing.T) {
	// Test a more complex ActionGroup with nested subgroups
	actionGroup := ActionGroup{
		If: []string{"state=1", "priority=1"},
		Set: map[string]string{
			"urgency": "1",
			"impact":  "1",
			"state":   "2",
		},
		Unset: []string{"caller_id", "description"},
		Subgroup: []ActionGroup{
			{
				If:  []string{"urgency=1"},
				Set: map[string]string{"priority": "1"},
				Subgroup: []ActionGroup{
					{
						If:  []string{"priority=1"},
						Set: map[string]string{"escalation": "true"},
					},
				},
			},
			{
				If:    []string{"state=6"},
				Break: []string{"stop_processing"},
			},
		},
		Break: []string{"complete"},
	}

	// Test JSON round trip
	jsonData, err := json.Marshal(actionGroup)
	if err != nil {
		t.Fatalf("Failed to marshal complex ActionGroup to JSON: %v", err)
	}

	var unmarshaledGroup ActionGroup
	err = json.Unmarshal(jsonData, &unmarshaledGroup)
	if err != nil {
		t.Fatalf("Failed to unmarshal complex ActionGroup from JSON: %v", err)
	}

	// Verify structure is preserved
	if len(unmarshaledGroup.If) != 2 {
		t.Errorf("Expected 2 If conditions, got %d", len(unmarshaledGroup.If))
	}

	if len(unmarshaledGroup.Set) != 3 {
		t.Errorf("Expected 3 Set fields, got %d", len(unmarshaledGroup.Set))
	}

	if len(unmarshaledGroup.Unset) != 2 {
		t.Errorf("Expected 2 Unset fields, got %d", len(unmarshaledGroup.Unset))
	}

	if len(unmarshaledGroup.Subgroup) != 2 {
		t.Errorf("Expected 2 subgroups, got %d", len(unmarshaledGroup.Subgroup))
	}

	// Test nested subgroup
	if len(unmarshaledGroup.Subgroup[0].Subgroup) != 1 {
		t.Errorf("Expected 1 nested subgroup, got %d", len(unmarshaledGroup.Subgroup[0].Subgroup))
	}

	if unmarshaledGroup.Subgroup[0].Subgroup[0].Set["escalation"] != "true" {
		t.Errorf("Expected nested subgroup escalation to be 'true', got %v", unmarshaledGroup.Subgroup[0].Subgroup[0].Set["escalation"])
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}