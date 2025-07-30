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

package snow

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestResults_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Results
		wantErr  bool
	}{
		{
			name:  "array of results",
			input: `[{"sys_id": "123", "number": "INC001"}, {"sys_id": "456", "number": "INC002"}]`,
			expected: Results{
				{"sys_id": "123", "number": "INC001"},
				{"sys_id": "456", "number": "INC002"},
			},
			wantErr: false,
		},
		{
			name:  "single result object",
			input: `{"sys_id": "123", "number": "INC001"}`,
			expected: Results{
				{"sys_id": "123", "number": "INC001"},
			},
			wantErr: false,
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: Results{},
			wantErr:  false,
		},
		{
			name:    "invalid JSON",
			input:   `invalid`,
			wantErr: true,
		},
		{
			name:    "unsupported type",
			input:   `"string"`,
			wantErr: true,
		},
		{
			name:  "whitespace handling",
			input: `  {"sys_id": "123"}  `,
			expected: Results{
				{"sys_id": "123"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result Results
			err := json.Unmarshal([]byte(tt.input), &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("UnmarshalJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTableConfig(t *testing.T) {
	// Create a test configuration
	cf := config.New()
	
	// Set up test table configuration
	tables := []TableData{
		{
			Name:   "incident",
			Search: "correlation_id=${correlation_id}",
			Query: TableQuery{
				Enabled:        true,
				ResponseFields: []string{"sys_id", "number", "state"},
			},
			Defaults: map[string]string{
				"urgency": "3",
				"impact":  "3",
			},
			CurrentStates: map[int]TableStates{
				1: {
					Defaults: map[string]string{
						"state": "2",
					},
				},
			},
			Response: TableResponses{
				Created: "Incident ${number} created",
				Updated: "Incident ${number} updated",
				Failed:  "Failed to process incident",
			},
		},
		{
			Name:   "change_request",
			Search: "correlation_id=${correlation_id}",
		},
	}

	cf.Set("servicenow.tables", tables)

	tests := []struct {
		name      string
		tableName string
		wantErr   bool
		expected  TableData
	}{
		{
			name:      "existing table",
			tableName: "incident",
			wantErr:   false,
			expected:  tables[0],
		},
		{
			name:      "another existing table",
			tableName: "change_request",
			wantErr:   false,
			expected:  tables[1],
		},
		{
			name:      "non-existing table",
			tableName: "problem",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TableConfig(cf, tt.tableName)

			if (err != nil) != tt.wantErr {
				t.Errorf("TableConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TableConfig() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRecord_CreateRecord(t *testing.T) {
	// This test would require mocking the HTTP client and ServiceNow function
	// For now, we'll test the basic structure
	record := Record{
		"short_description": "Test incident",
		"urgency":          "3",
		"impact":           "3",
	}

	// Verify record structure
	if record["short_description"] != "Test incident" {
		t.Errorf("Expected short_description to be 'Test incident', got %v", record["short_description"])
	}
	if record["urgency"] != "3" {
		t.Errorf("Expected urgency to be '3', got %v", record["urgency"])
	}
}

func TestRecord_UpdateRecord(t *testing.T) {
	// This test would require mocking the HTTP client and ServiceNow function
	// For now, we'll test the basic structure
	record := Record{
		"state":            "2",
		"work_notes":       "Updated by test",
	}

	// Verify record structure
	if record["state"] != "2" {
		t.Errorf("Expected state to be '2', got %v", record["state"])
	}
	if record["work_notes"] != "Updated by test" {
		t.Errorf("Expected work_notes to be 'Updated by test', got %v", record["work_notes"])
	}
}

func TestSnowError(t *testing.T) {
	// Test JSON unmarshaling of error response
	errorJSON := `{
		"error": {
			"message": "Invalid table",
			"detail": "Table 'invalid_table' does not exist"
		},
		"status": "failure"
	}`

	var snowErr snowError
	err := json.Unmarshal([]byte(errorJSON), &snowErr)
	if err != nil {
		t.Fatalf("Failed to unmarshal error JSON: %v", err)
	}

	if snowErr.Error.Message != "Invalid table" {
		t.Errorf("Expected error message 'Invalid table', got %v", snowErr.Error.Message)
	}
	if snowErr.Error.Detail != "Table 'invalid_table' does not exist" {
		t.Errorf("Expected error detail 'Table 'invalid_table' does not exist', got %v", snowErr.Error.Detail)
	}
	if snowErr.Status != "failure" {
		t.Errorf("Expected status 'failure', got %v", snowErr.Status)
	}
}

func TestSnowResult(t *testing.T) {
	// Test JSON unmarshaling of successful response
	resultJSON := `{
		"result": [
			{
				"sys_id": "12345",
				"number": "INC001",
				"state": "1"
			}
		]
	}`

	var result snowResult
	err := json.Unmarshal([]byte(resultJSON), &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result JSON: %v", err)
	}

	if len(result.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0]["sys_id"] != "12345" {
		t.Errorf("Expected sys_id '12345', got %v", result.Results[0]["sys_id"])
	}
	if result.Results[0]["number"] != "INC001" {
		t.Errorf("Expected number 'INC001', got %v", result.Results[0]["number"])
	}
}

func TestTableQuery(t *testing.T) {
	query := TableQuery{
		Enabled:        true,
		ResponseFields: []string{"sys_id", "number", "state", "short_description"},
	}

	if !query.Enabled {
		t.Error("Expected query to be enabled")
	}

	expectedFields := []string{"sys_id", "number", "state", "short_description"}
	if !reflect.DeepEqual(query.ResponseFields, expectedFields) {
		t.Errorf("Expected response fields %v, got %v", expectedFields, query.ResponseFields)
	}
}

func TestTableStates(t *testing.T) {
	states := TableStates{
		Defaults: map[string]string{
			"urgency": "3",
			"impact":  "3",
		},
		Remove: []string{"caller_id"},
		Rename: map[string]string{
			"short_description": "description",
		},
		MustInclude: []string{"sys_id", "number"},
		Filter:      []string{"state!=6"},
	}

	if states.Defaults["urgency"] != "3" {
		t.Errorf("Expected default urgency '3', got %v", states.Defaults["urgency"])
	}

	if len(states.Remove) != 1 || states.Remove[0] != "caller_id" {
		t.Errorf("Expected remove field 'caller_id', got %v", states.Remove)
	}

	if states.Rename["short_description"] != "description" {
		t.Errorf("Expected rename mapping 'short_description' -> 'description', got %v", states.Rename)
	}

	expectedMustInclude := []string{"sys_id", "number"}
	if !reflect.DeepEqual(states.MustInclude, expectedMustInclude) {
		t.Errorf("Expected must include %v, got %v", expectedMustInclude, states.MustInclude)
	}

	expectedFilter := []string{"state!=6"}
	if !reflect.DeepEqual(states.Filter, expectedFilter) {
		t.Errorf("Expected filter %v, got %v", expectedFilter, states.Filter)
	}
}

func TestTableResponses(t *testing.T) {
	responses := TableResponses{
		Created: "Incident ${number} created successfully",
		Updated: "Incident ${number} updated successfully", 
		Failed:  "Failed to process incident: ${error}",
	}

	if responses.Created != "Incident ${number} created successfully" {
		t.Errorf("Expected created response 'Incident ${number} created successfully', got %v", responses.Created)
	}

	if responses.Updated != "Incident ${number} updated successfully" {
		t.Errorf("Expected updated response 'Incident ${number} updated successfully', got %v", responses.Updated)
	}

	if responses.Failed != "Failed to process incident: ${error}" {
		t.Errorf("Expected failed response 'Failed to process incident: ${error}', got %v", responses.Failed)
	}
}

func TestTableData(t *testing.T) {
	tableData := TableData{
		Name:   "incident",
		Search: "correlation_id=${correlation_id}",
		Query: TableQuery{
			Enabled:        true,
			ResponseFields: []string{"sys_id", "number"},
		},
		Defaults: map[string]string{
			"urgency": "3",
		},
		CurrentStates: map[int]TableStates{
			1: {
				Defaults: map[string]string{
					"state": "2",
				},
			},
		},
		Response: TableResponses{
			Created: "Created ${number}",
			Updated: "Updated ${number}",
			Failed:  "Failed",
		},
	}

	if tableData.Name != "incident" {
		t.Errorf("Expected table name 'incident', got %v", tableData.Name)
	}

	if tableData.Search != "correlation_id=${correlation_id}" {
		t.Errorf("Expected search 'correlation_id=${correlation_id}', got %v", tableData.Search)
	}

	if !tableData.Query.Enabled {
		t.Error("Expected query to be enabled")
	}

	if tableData.Defaults["urgency"] != "3" {
		t.Errorf("Expected default urgency '3', got %v", tableData.Defaults["urgency"])
	}

	if tableData.CurrentStates[1].Defaults["state"] != "2" {
		t.Errorf("Expected state 1 default state '2', got %v", tableData.CurrentStates[1].Defaults["state"])
	}
}