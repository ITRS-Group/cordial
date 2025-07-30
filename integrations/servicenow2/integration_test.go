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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/itrs-group/cordial/integrations/servicenow2/internal/snow"
	"github.com/itrs-group/cordial/pkg/config"
)

// MockServiceNowServer creates a mock ServiceNow server for testing
func MockServiceNowServer() *httptest.Server {
	e := echo.New()

	// Mock incidents table endpoint
	e.GET("/api/now/v2/table/incident", func(c echo.Context) error {
		query := c.QueryParam("sysparm_query")
		// Note: limit and fields are available but not used in this mock
		// limit := c.QueryParam("sysparm_limit")
		// fields := c.QueryParam("sysparm_fields")

		// Mock response based on query
		var results []map[string]string

		if strings.Contains(query, "correlation_id=test-123") {
			results = append(results, map[string]string{
				"sys_id": "existing-incident-id",
				"number": "INC0000001",
				"state":  "1",
			})
		}

		response := map[string]interface{}{
			"result": results,
		}

		return c.JSON(http.StatusOK, response)
	})

	// Mock incident creation endpoint
	e.POST("/api/now/v2/table/incident", func(c echo.Context) error {
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		}

		// Mock successful creation
		result := map[string]string{
			"sys_id":           "new-incident-id",
			"number":           "INC0000002",
			"short_description": fmt.Sprintf("%v", body["short_description"]),
			"state":            "1",
		}

		response := map[string]interface{}{
			"result": []map[string]string{result},
		}

		return c.JSON(http.StatusCreated, response)
	})

	// Mock incident update endpoint
	e.PUT("/api/now/v2/table/incident/:sys_id", func(c echo.Context) error {
		sysID := c.Param("sys_id")
		
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		}

		// Mock successful update
		result := map[string]string{
			"sys_id": sysID,
			"number": "INC0000001",
			"state":  "2", // Updated state
		}

		response := map[string]interface{}{
			"result": []map[string]string{result},
		}

		return c.JSON(http.StatusOK, response)
	})

	// Mock OAuth token endpoint
	e.POST("/oauth_token.do", func(c echo.Context) error {
		response := map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		return c.JSON(http.StatusOK, response)
	})

	return httptest.NewServer(e)
}

func TestServiceNowClientIntegration(t *testing.T) {
	// Start mock ServiceNow server
	server := MockServiceNowServer()
	defer server.Close()

	// Create test configuration
	cf := config.New()
	cf.Set("servicenow.username", "testuser")
	cf.Set("servicenow.password", "testpass")
	cf.Set("servicenow.url", server.URL)
	cf.Set("servicenow.path", "/api/now/v2/table")

	// Test ServiceNow client creation and basic functionality
	client := snow.ServiceNow(cf.Sub("servicenow"))
	if client == nil {
		t.Fatal("Failed to create ServiceNow client")
	}

	// Test making a request to get incidents
	ctx := &snow.Context{
		Conf: cf,
	}

	// Create a mock echo context for testing
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)
	echoCtx.SetParamNames("table")
	echoCtx.SetParamValues("incident")
	ctx.Context = echoCtx

	// Test GetRecords
	results, err := snow.GetRecords(ctx, "incident", snow.Query("correlation_id=test-123"))
	if err != nil {
		t.Fatalf("Failed to get records: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0]["number"] != "INC0000001" {
		t.Errorf("Expected incident number INC0000001, got %s", results[0]["number"])
	}
}

func TestServiceNowOAuthIntegration(t *testing.T) {
	// Start mock ServiceNow server
	server := MockServiceNowServer()
	defer server.Close()

	// Create test configuration with OAuth
	cf := config.New()
	cf.Set("servicenow.username", "testuser")
	cf.Set("servicenow.password", "testpass")
	cf.Set("servicenow.client-id", "test-client-id")
	cf.Set("servicenow.client-secret", "test-client-secret")
	cf.Set("servicenow.url", server.URL)
	cf.Set("servicenow.path", "/api/now/v2/table")

	// Test ServiceNow client creation with OAuth
	client := snow.ServiceNow(cf.Sub("servicenow"))
	if client == nil {
		t.Fatal("Failed to create ServiceNow client with OAuth")
	}

	// Reset the global connection for cleanup
	// Note: In a real integration test, we might want to test the actual OAuth flow
	// but for this mock test, we're just verifying the client can be created
}

func TestIncidentCreationFlow(t *testing.T) {
	// Start mock ServiceNow server
	server := MockServiceNowServer()
	defer server.Close()

	// Create test configuration
	cf := config.New()
	cf.Set("servicenow.username", "testuser")
	cf.Set("servicenow.password", "testpass")
	cf.Set("servicenow.url", server.URL)
	cf.Set("servicenow.path", "/api/now/v2/table")

	// Create a test record
	record := snow.Record{
		"short_description": "Test incident from integration test",
		"urgency":          "3",
		"impact":           "3",
		"correlation_id":   "test-integration-123",
	}

	// Create echo context for testing
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)
	echoCtx.SetParamNames("table")
	echoCtx.SetParamValues("incident")

	ctx := &snow.Context{
		Context: echoCtx,
		Conf:    cf,
	}

	// Test creating a record
	number, err := record.CreateRecord(ctx)
	if err != nil {
		t.Fatalf("Failed to create record: %v", err)
	}

	if number != "INC0000002" {
		t.Errorf("Expected incident number INC0000002, got %s", number)
	}
}

func TestIncidentUpdateFlow(t *testing.T) {
	// Start mock ServiceNow server
	server := MockServiceNowServer()
	defer server.Close()

	// Create test configuration
	cf := config.New()
	cf.Set("servicenow.username", "testuser")
	cf.Set("servicenow.password", "testpass")
	cf.Set("servicenow.url", server.URL)
	cf.Set("servicenow.path", "/api/now/v2/table")

	// Create a test record for update
	record := snow.Record{
		"work_notes": "Updated from integration test",
		"state":     "2",
	}

	// Create echo context for testing
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/test", nil)
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)
	echoCtx.SetParamNames("table")
	echoCtx.SetParamValues("incident")

	ctx := &snow.Context{
		Context: echoCtx,
		Conf:    cf,
	}

	// Test updating a record
	number, err := record.UpdateRecord(ctx, "existing-incident-id")
	if err != nil {
		t.Fatalf("Failed to update record: %v", err)
	}

	if number != "INC0000001" {
		t.Errorf("Expected incident number INC0000001, got %s", number)
	}
}

func TestLookupRecordFlow(t *testing.T) {
	// Start mock ServiceNow server
	server := MockServiceNowServer()
	defer server.Close()

	// Create test configuration with table configuration
	cf := config.New()
	cf.Set("servicenow.username", "testuser")
	cf.Set("servicenow.password", "testpass")
	cf.Set("servicenow.url", server.URL)
	cf.Set("servicenow.path", "/api/now/v2/table")
	
	// Set up table configuration
	tables := []snow.TableData{
		{
			Name:   "incident",
			Search: "correlation_id=${correlation_id}",
		},
	}
	cf.Set("servicenow.tables", tables)

	// Create echo context for testing
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)
	echoCtx.SetParamNames("table")
	echoCtx.SetParamValues("incident")

	ctx := &snow.Context{
		Context: echoCtx,
		Conf:    cf,
	}

	// Test looking up a record with correlation_id expansion
	// Use LookupTable to provide the correlation_id value
	sysID, state, err := snow.LookupRecord(ctx, config.LookupTable(map[string]string{
		"correlation_id": "test-123",
	}))
	if err != nil {
		t.Fatalf("Failed to lookup record: %v", err)
	}

	if sysID != "existing-incident-id" {
		t.Errorf("Expected sys_id 'existing-incident-id', got %s", sysID)
	}

	if state != 1 {
		t.Errorf("Expected state 1, got %d", state)
	}
}

func TestErrorHandling(t *testing.T) {
	// Test with invalid server URL
	cf := config.New()
	cf.Set("username", "testuser")
	cf.Set("password", "testpass")
	cf.Set("url", "http://invalid-url-that-does-not-exist.local")
	cf.Set("path", "/api/now/v2/table")

	// Create echo context for testing
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)
	echoCtx.SetParamNames("table")
	echoCtx.SetParamValues("incident")

	ctx := &snow.Context{
		Context: echoCtx,
		Conf:    cf.Sub("servicenow"),
	}

	// This should fail due to invalid URL
	_, err := snow.GetRecords(ctx, "incident")
	if err == nil {
		t.Error("Expected error when connecting to invalid URL, got nil")
	}
}

func TestTableConfiguration(t *testing.T) {
	// Test table configuration parsing
	cf := config.New()
	
	tables := []snow.TableData{
		{
			Name:   "incident",
			Search: "correlation_id=${correlation_id}",
			Query: snow.TableQuery{
				Enabled:        true,
				ResponseFields: []string{"sys_id", "number", "state"},
			},
			Defaults: map[string]string{
				"urgency": "3",
				"impact":  "3",
			},
			CurrentStates: map[int]snow.TableStates{
				1: {
					Defaults: map[string]string{
						"state": "2",
					},
				},
			},
			Response: snow.TableResponses{
				Created: "Incident ${number} created",
				Updated: "Incident ${number} updated",
				Failed:  "Failed to process incident",
			},
		},
	}

	cf.Set("servicenow.tables", tables)

	// Test table lookup
	tableData, err := snow.TableConfig(cf, "incident")
	if err != nil {
		t.Fatalf("Failed to get table config: %v", err)
	}

	if tableData.Name != "incident" {
		t.Errorf("Expected table name 'incident', got %s", tableData.Name)
	}

	if tableData.Search != "correlation_id=${correlation_id}" {
		t.Errorf("Expected search 'correlation_id=${correlation_id}', got %s", tableData.Search)
	}

	// Test non-existent table
	_, err = snow.TableConfig(cf, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent table, got nil")
	}
}

func TestCommandLineIntegration(t *testing.T) {
	// Test command line parsing and initialization
	
	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test root command
	if cmd.RootCmd == nil {
		t.Fatal("Root command not initialized")
	}

	// Test that commands are properly registered
	commands := cmd.RootCmd.Commands()
	commandNames := make([]string, len(commands))
	for i, c := range commands {
		commandNames[i] = c.Name()
	}

	expectedCommands := []string{"client", "proxy"}
	for _, expected := range expectedCommands {
		found := false
		for _, name := range commandNames {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command %s to be registered, but it's missing", expected)
		}
	}
}

func TestFullWorkflow(t *testing.T) {
	// Test a complete workflow: lookup -> create or update
	server := MockServiceNowServer()
	defer server.Close()

	// Create test configuration
	cf := config.New()
	cf.Set("servicenow.username", "testuser")
	cf.Set("servicenow.password", "testpass")
	cf.Set("servicenow.url", server.URL)
	cf.Set("servicenow.path", "/api/now/v2/table")
	
	// Set up table configuration
	tables := []snow.TableData{
		{
			Name:   "incident",
			Search: "correlation_id=${correlation_id}",
		},
	}
	cf.Set("servicenow.tables", tables)

	// Create echo context for testing
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)
	echoCtx.SetParamNames("table")
	echoCtx.SetParamValues("incident")

	ctx := &snow.Context{
		Context: echoCtx,
		Conf:    cf,
	}

	// Step 1: Lookup existing incident
	sysID, state, err := snow.LookupRecord(ctx, config.LookupTable(map[string]string{
		"correlation_id": "test-123",
	}))
	if err != nil {
		t.Fatalf("Failed to lookup record: %v", err)
	}

	if sysID != "" {
		// Step 2a: Update existing incident
		record := snow.Record{
			"work_notes": "Updated by integration test",
			"state":     "2",
		}

		number, err := record.UpdateRecord(ctx, sysID)
		if err != nil {
			t.Fatalf("Failed to update existing record: %v", err)
		}

		t.Logf("Updated incident %s (sys_id: %s, state: %d)", number, sysID, state)
	} else {
		// Step 2b: Create new incident
		record := snow.Record{
			"short_description": "New incident from integration test",
			"correlation_id":   "test-new-123",
			"urgency":          "3",
			"impact":           "3",
		}

		number, err := record.CreateRecord(ctx)
		if err != nil {
			t.Fatalf("Failed to create new record: %v", err)
		}

		t.Logf("Created incident %s", number)
	}
}

func TestJSONSerialization(t *testing.T) {
	// Test JSON serialization of various data structures
	
	// Test Record
	record := snow.Record{
		"short_description": "Test incident",
		"urgency":          "3",
		"correlation_id":   "test-123",
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Failed to marshal Record: %v", err)
	}

	var unmarshaledRecord snow.Record
	err = json.Unmarshal(jsonData, &unmarshaledRecord)
	if err != nil {
		t.Fatalf("Failed to unmarshal Record: %v", err)
	}

	if unmarshaledRecord["short_description"] != "Test incident" {
		t.Errorf("Record serialization failed")
	}

	// Test Results
	results := snow.Results{
		{"sys_id": "123", "number": "INC001"},
		{"sys_id": "456", "number": "INC002"},
	}

	jsonData, err = json.Marshal(results)
	if err != nil {
		t.Fatalf("Failed to marshal Results: %v", err)
	}

	var unmarshaledResults snow.Results
	err = json.Unmarshal(jsonData, &unmarshaledResults)
	if err != nil {
		t.Fatalf("Failed to unmarshal Results: %v", err)
	}

	if len(unmarshaledResults) != 2 {
		t.Errorf("Expected 2 results, got %d", len(unmarshaledResults))
	}
}