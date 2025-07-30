package commands

import (
	"net/url"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/xpath"
)

func TestCommandStruct(t *testing.T) {
	// Test Command struct
	target := xpath.NewDataviewPath("testDataview")
	args := CommandArgs{"arg1": "value1", "arg2": "value2"}
	scope := Scope{Value: true, Severity: false}

	command := Command{
		Name:   "testCommand",
		Target: target,
		Args:   &args,
		Scope:  scope,
		Limit:  10,
	}

	if command.Name != "testCommand" {
		t.Errorf("Expected command name 'testCommand', got '%s'", command.Name)
	}

	if command.Target == nil {
		t.Error("Target should not be nil")
	}

	if command.Args == nil {
		t.Error("Args should not be nil")
	}

	if len(*command.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(*command.Args))
	}

	if !command.Scope.Value {
		t.Error("Scope.Value should be true")
	}

	if command.Scope.Severity {
		t.Error("Scope.Severity should be false")
	}

	if command.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", command.Limit)
	}
}

func TestCommandArgs(t *testing.T) {
	// Test CommandArgs
	args := CommandArgs{
		"arg1": "value1",
		"arg2": "value2",
		"arg3": "value3",
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}

	if args["arg1"] != "value1" {
		t.Errorf("Expected 'value1' for arg1, got '%s'", args["arg1"])
	}

	if args["arg2"] != "value2" {
		t.Errorf("Expected 'value2' for arg2, got '%s'", args["arg2"])
	}

	if args["arg3"] != "value3" {
		t.Errorf("Expected 'value3' for arg3, got '%s'", args["arg3"])
	}

	// Test setting new value
	args["arg4"] = "value4"
	if args["arg4"] != "value4" {
		t.Errorf("Expected 'value4' for arg4, got '%s'", args["arg4"])
	}
}

func TestScope(t *testing.T) {
	// Test Scope struct
	scope := Scope{
		Value:          true,
		Severity:       false,
		Snooze:         true,
		UserAssignment: false,
	}

	if !scope.Value {
		t.Error("Value should be true")
	}

	if scope.Severity {
		t.Error("Severity should be false")
	}

	if !scope.Snooze {
		t.Error("Snooze should be true")
	}

	if scope.UserAssignment {
		t.Error("UserAssignment should be false")
	}
}

func TestCommandResponseRaw(t *testing.T) {
	// Test CommandResponseRaw struct
	target := xpath.NewDataviewPath("testDataview")
	mimeType := []map[string]string{{"type": "text/plain"}}
	streamData := []map[string]string{{"data": "test"}}

	response := CommandResponseRaw{
		Target:     target,
		MimeType:   mimeType,
		Status:     "success",
		StreamData: streamData,
		XPaths:     []string{"/geneos/gateway/test"},
	}

	if response.Target == nil {
		t.Error("Target should not be nil")
	}

	if len(response.MimeType) != 1 {
		t.Errorf("Expected 1 mime type, got %d", len(response.MimeType))
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	if len(response.StreamData) != 1 {
		t.Errorf("Expected 1 stream data, got %d", len(response.StreamData))
	}

	if len(response.XPaths) != 1 {
		t.Errorf("Expected 1 xpath, got %d", len(response.XPaths))
	}
}

func TestCommandResponse(t *testing.T) {
	// Test CommandResponse struct
	target := xpath.NewDataviewPath("testDataview")
	mimeType := map[string]string{"type": "text/plain"}

	response := CommandResponse{
		Target:         target,
		MimeType:       mimeType,
		Status:         "success",
		Stdout:         "test output",
		StdoutMimeType: "text/plain",
		Stderr:         "test error",
		ExecLog:        "test log",
		XPaths:         []string{"/geneos/gateway/test"},
	}

	if response.Target == nil {
		t.Error("Target should not be nil")
	}

	if len(response.MimeType) != 1 {
		t.Errorf("Expected 1 mime type, got %d", len(response.MimeType))
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	if response.Stdout != "test output" {
		t.Errorf("Expected stdout 'test output', got '%s'", response.Stdout)
	}

	if response.StdoutMimeType != "text/plain" {
		t.Errorf("Expected stdout mime type 'text/plain', got '%s'", response.StdoutMimeType)
	}

	if response.Stderr != "test error" {
		t.Errorf("Expected stderr 'test error', got '%s'", response.Stderr)
	}

	if response.ExecLog != "test log" {
		t.Errorf("Expected exec log 'test log', got '%s'", response.ExecLog)
	}

	if len(response.XPaths) != 1 {
		t.Errorf("Expected 1 xpath, got %d", len(response.XPaths))
	}
}

func TestConnection(t *testing.T) {
	// Test Connection struct
	baseURL, _ := url.Parse("http://localhost:8080")
	password := config.NewPlaintext("testpass")

	conn := &Connection{
		BaseURL:            baseURL,
		AuthType:           1,
		Username:           "testuser",
		Password:           password,
		InsecureSkipVerify: true,
		Timeout:            30 * time.Second,
	}

	if conn.BaseURL == nil {
		t.Error("BaseURL should not be nil")
	}

	if conn.AuthType != 1 {
		t.Errorf("Expected auth type 1, got %d", conn.AuthType)
	}

	if conn.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", conn.Username)
	}

	if conn.Password == nil {
		t.Error("Password should not be nil")
	}

	if !conn.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}

	if conn.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", conn.Timeout)
	}
}

func TestDialGateway(t *testing.T) {
	// Test with valid URL
	validURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse valid URL: %v", err)
	}

	conn, err := DialGateway(validURL)
	if err != nil {
		t.Errorf("DialGateway failed: %v", err)
	}

	if conn == nil {
		t.Fatal("DialGateway should not return nil connection")
	}

	if conn.BaseURL == nil {
		t.Error("Connection BaseURL should not be nil")
	}

	// Test with nil URL
	_, err = DialGateway(nil)
	if err == nil {
		t.Error("Expected error with nil URL")
	}
}

func TestDialGateways(t *testing.T) {
	// Test with valid URLs
	url1, _ := url.Parse("http://localhost:8080")
	url2, _ := url.Parse("http://localhost:8081")
	urls := []*url.URL{url1, url2}

	conn, err := DialGateways(urls)
	if err != nil {
		t.Errorf("DialGateways failed: %v", err)
	}

	if conn == nil {
		t.Fatal("DialGateways should not return nil connection")
	}

	// Test with empty URLs slice
	_, err = DialGateways([]*url.URL{})
	if err == nil {
		t.Error("Expected error with empty URLs slice")
	}

	// Test with nil URLs slice
	_, err = DialGateways(nil)
	if err == nil {
		t.Error("Expected error with nil URLs slice")
	}
}

func TestRunCommand(t *testing.T) {
	// Test RunCommand with valid connection
	baseURL, _ := url.Parse("http://localhost:8080")
	conn := &Connection{
		BaseURL: baseURL,
		Timeout: 30 * time.Second,
	}

	target := xpath.NewDataviewPath("testDataview")
	args := []Args{{"arg1", "value1"}}

	response, err := conn.RunCommand("testCommand", target, args...)
	if err == nil {
		t.Error("Expected error with invalid connection")
	}

	// Test with nil target
	_, err = conn.RunCommand("testCommand", nil, args...)
	if err == nil {
		t.Error("Expected error with nil target")
	}
}

func TestRunCommandAll(t *testing.T) {
	// Test RunCommandAll with valid connection
	baseURL, _ := url.Parse("http://localhost:8080")
	conn := &Connection{
		BaseURL: baseURL,
		Timeout: 30 * time.Second,
	}

	target := xpath.NewDataviewPath("testDataview")
	args := []Args{{"arg1", "value1"}}

	responses, err := conn.RunCommandAll("testCommand", target, args...)
	if err == nil {
		t.Error("Expected error with invalid connection")
	}

	if responses != nil {
		t.Error("Responses should be nil on error")
	}
}

func TestMatch(t *testing.T) {
	// Test Match with valid connection
	baseURL, _ := url.Parse("http://localhost:8080")
	conn := &Connection{
		BaseURL: baseURL,
		Timeout: 30 * time.Second,
	}

	target := xpath.NewDataviewPath("testDataview")

	matches, err := conn.Match(target, 10)
	if err == nil {
		t.Error("Expected error with invalid connection")
	}

	if matches != nil {
		t.Error("Matches should be nil on error")
	}

	// Test with nil target
	_, err = conn.Match(nil, 10)
	if err == nil {
		t.Error("Expected error with nil target")
	}
}

func TestCommandTargets(t *testing.T) {
	// Test CommandTargets with valid connection
	baseURL, _ := url.Parse("http://localhost:8080")
	conn := &Connection{
		BaseURL: baseURL,
		Timeout: 30 * time.Second,
	}

	target := xpath.NewDataviewPath("testDataview")

	matches, err := conn.CommandTargets("testCommand", target)
	if err == nil {
		t.Error("Expected error with invalid connection")
	}

	if matches != nil {
		t.Error("Matches should be nil on error")
	}

	// Test with nil target
	_, err = conn.CommandTargets("testCommand", nil)
	if err == nil {
		t.Error("Expected error with nil target")
	}
}

func TestArgs(t *testing.T) {
	// Test Args type
	args := Args{"arg1", "value1"}

	if len(args) != 2 {
		t.Errorf("Expected 2 elements in args, got %d", len(args))
	}

	if args[0] != "arg1" {
		t.Errorf("Expected first element 'arg1', got '%s'", args[0])
	}

	if args[1] != "value1" {
		t.Errorf("Expected second element 'value1', got '%s'", args[1])
	}
}

func TestDataview(t *testing.T) {
	// Test Dataview struct
	dataview := &Dataview{
		Headlines: []string{"headline1", "headline2"},
		Rows:      [][]string{{"row1col1", "row1col2"}, {"row2col1", "row2col2"}},
	}

	if len(dataview.Headlines) != 2 {
		t.Errorf("Expected 2 headlines, got %d", len(dataview.Headlines))
	}

	if len(dataview.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(dataview.Rows))
	}

	if len(dataview.Rows[0]) != 2 {
		t.Errorf("Expected 2 columns in first row, got %d", len(dataview.Rows[0]))
	}
}

func TestGeneosRESTError(t *testing.T) {
	// Test GeneosRESTError struct
	restError := GeneosRESTError{
		Error: "test error message",
	}

	if restError.Error != "test error message" {
		t.Errorf("Expected error 'test error message', got '%s'", restError.Error)
	}
}

func TestCookResponse(t *testing.T) {
	// Test cookResponse function
	target := xpath.NewDataviewPath("testDataview")
	raw := CommandResponseRaw{
		Target: target,
		Status: "success",
		MimeType: []map[string]string{
			{"type": "text/plain"},
		},
		StreamData: []map[string]string{
			{"data": "test data"},
		},
		XPaths: []string{"/geneos/gateway/test"},
	}

	response := cookResponse(raw)

	if response.Target == nil {
		t.Error("Cooked response target should not be nil")
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	if len(response.MimeType) != 1 {
		t.Errorf("Expected 1 mime type, got %d", len(response.MimeType))
	}

	if len(response.XPaths) != 1 {
		t.Errorf("Expected 1 xpath, got %d", len(response.XPaths))
	}
}