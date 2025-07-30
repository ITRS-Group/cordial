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
	"net/url"
	"testing"
)

func TestLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		expected string
	}{
		{"zero limit", 0, "0"},
		{"positive limit", 10, "10"},
		{"large limit", 1000, "1000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := evalReqOptions(Limit(tt.limit))
			if opts.limit != tt.expected {
				t.Errorf("Limit(%d) = %q, want %q", tt.limit, opts.limit, tt.expected)
			}
		})
	}
}

func TestFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   string
		expected string
	}{
		{"single field", "sys_id", "sys_id"},
		{"multiple fields", "sys_id,number,state", "sys_id,number,state"},
		{"empty fields", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := evalReqOptions(Fields(tt.fields))
			if opts.fields != tt.expected {
				t.Errorf("Fields(%q) = %q, want %q", tt.fields, opts.fields, tt.expected)
			}
		})
	}
}

func TestOffset(t *testing.T) {
	tests := []struct {
		name     string
		offset   string
		expected string
	}{
		{"zero offset", "0", "0"},
		{"positive offset", "100", "100"},
		{"empty offset", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := evalReqOptions(Offset(tt.offset))
			if opts.offset != tt.expected {
				t.Errorf("Offset(%q) = %q, want %q", tt.offset, opts.offset, tt.expected)
			}
		})
	}
}

func TestQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{"simple query", "state=1", "state=1"},
		{"complex query", "state=1^priority=1", "state=1^priority=1"},
		{"empty query", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := evalReqOptions(Query(tt.query))
			if opts.query != tt.expected {
				t.Errorf("Query(%q) = %q, want %q", tt.query, opts.query, tt.expected)
			}
		})
	}
}

func TestDisplay(t *testing.T) {
	tests := []struct {
		name     string
		display  string
		expected string
	}{
		{"display true", "true", "true"},
		{"display false", "false", "false"},
		{"display all", "all", "all"},
		{"empty display", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := evalReqOptions(Display(tt.display))
			if opts.display != tt.expected {
				t.Errorf("Display(%q) = %q, want %q", tt.display, opts.display, tt.expected)
			}
		})
	}
}

func TestSysID(t *testing.T) {
	tests := []struct {
		name     string
		sysID    string
		expected string
	}{
		{"valid sys_id", "12345678-1234-1234-1234-123456789012", "12345678-1234-1234-1234-123456789012"},
		{"empty sys_id", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := evalReqOptions(SysID(tt.sysID))
			if opts.sysID != tt.expected {
				t.Errorf("SysID(%q) = %q, want %q", tt.sysID, opts.sysID, tt.expected)
			}
		})
	}
}

func TestEvalReqOptions(t *testing.T) {
	t.Run("multiple options", func(t *testing.T) {
		opts := evalReqOptions(
			Limit(10),
			Fields("sys_id,number"),
			Query("state=1"),
			Display("true"),
			SysID("test-id"),
		)

		if opts.limit != "10" {
			t.Errorf("limit = %q, want '10'", opts.limit)
		}
		if opts.fields != "sys_id,number" {
			t.Errorf("fields = %q, want 'sys_id,number'", opts.fields)
		}
		if opts.query != "state=1" {
			t.Errorf("query = %q, want 'state=1'", opts.query)
		}
		if opts.display != "true" {
			t.Errorf("display = %q, want 'true'", opts.display)
		}
		if opts.sysID != "test-id" {
			t.Errorf("sysID = %q, want 'test-id'", opts.sysID)
		}
	})

	t.Run("no options", func(t *testing.T) {
		opts := evalReqOptions()
		
		if opts.limit != "" {
			t.Errorf("limit = %q, want empty", opts.limit)
		}
		if opts.fields != "" {
			t.Errorf("fields = %q, want empty", opts.fields)
		}
		if opts.query != "" {
			t.Errorf("query = %q, want empty", opts.query)
		}
		if opts.display != "" {
			t.Errorf("display = %q, want empty", opts.display)
		}
		if opts.sysID != "" {
			t.Errorf("sysID = %q, want empty", opts.sysID)
		}
	})
}

func TestAssembleURL(t *testing.T) {
	tests := []struct {
		name     string
		table    string
		options  []Options
		expected map[string]string // expected query parameters
		path     string            // expected path
	}{
		{
			name:  "basic table",
			table: "incident",
			options: []Options{},
			expected: map[string]string{
				"sysparm_exclude_reference_link": "true",
			},
			path: "table/incident",
		},
		{
			name:  "with limit",
			table: "incident",
			options: []Options{Limit(10)},
			expected: map[string]string{
				"sysparm_limit": "10",
				"sysparm_exclude_reference_link": "true",
			},
			path: "table/incident",
		},
		{
			name:  "with fields",
			table: "incident",
			options: []Options{Fields("sys_id,number")},
			expected: map[string]string{
				"sysparm_fields": "sys_id,number",
				"sysparm_exclude_reference_link": "true",
			},
			path: "table/incident",
		},
		{
			name:  "with query",
			table: "incident",
			options: []Options{Query("state=1")},
			expected: map[string]string{
				"sysparm_query": "state=1",
				"sysparm_exclude_reference_link": "true",
			},
			path: "table/incident",
		},
		{
			name:  "with sys_id",
			table: "incident",
			options: []Options{SysID("test-id")},
			expected: map[string]string{
				"sysparm_exclude_reference_link": "true",
			},
			path: "table/incident/test-id",
		},
		{
			name:  "all options",
			table: "incident",
			options: []Options{
				Limit(5),
				Fields("sys_id,number,state"),
				Query("state=1^priority=1"),
				Display("true"),
				Offset("10"),
				SysID("test-id"),
			},
			expected: map[string]string{
				"sysparm_limit": "5",
				"sysparm_fields": "sys_id,number,state",
				"sysparm_query": "state=1^priority=1",
				"sysparm_display_value": "true",
				"sysparm_offset": "10",
				"sysparm_exclude_reference_link": "true",
			},
			path: "table/incident/test-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AssembleURL(tt.table, tt.options...)
			
			// Check path
			if result.Path != tt.path {
				t.Errorf("path = %q, want %q", result.Path, tt.path)
			}

			// Parse query parameters
			values, err := url.ParseQuery(result.RawQuery)
			if err != nil {
				t.Fatalf("failed to parse query: %v", err)
			}

			// Check each expected parameter
			for key, expectedValue := range tt.expected {
				if value := values.Get(key); value != expectedValue {
					t.Errorf("query param %q = %q, want %q", key, value, expectedValue)
				}
			}

			// Check no unexpected parameters (except the ones we expect)
			for key := range values {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("unexpected query param %q = %q", key, values.Get(key))
				}
			}
		})
	}
}