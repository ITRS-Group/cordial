package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestGeneosCommandInitialization(t *testing.T) {
	// Test that GeneosCmd is properly initialized
	if GeneosCmd == nil {
		t.Fatal("GeneosCmd should not be nil")
	}

	if GeneosCmd.Use == "" {
		t.Error("GeneosCmd.Use should not be empty")
	}

	if GeneosCmd.Short == "" {
		t.Error("GeneosCmd.Short should not be empty")
	}

	if GeneosCmd.Long == "" {
		t.Error("GeneosCmd.Long should not be empty")
	}

	if GeneosCmd.Version == "" {
		t.Error("GeneosCmd.Version should not be empty")
	}
}

func TestCommandAnnotations(t *testing.T) {
	// Test that expected annotations are present
	if GeneosCmd.Annotations == nil {
		t.Error("GeneosCmd should have annotations")
		return
	}

	// Check for required home annotation
	if val, ok := GeneosCmd.Annotations[CmdRequireHome]; !ok || val != "true" {
		t.Errorf("GeneosCmd should have %s annotation set to 'true', got %s", CmdRequireHome, val)
	}
}

func TestCommandFlags(t *testing.T) {
	// Test that basic flags are properly set up
	if GeneosCmd.Flags() == nil {
		t.Error("GeneosCmd should have flags initialized")
	}

	// Test that SortFlags is disabled
	if GeneosCmd.Flags().SortFlags {
		t.Error("GeneosCmd flags should not be sorted")
	}
}

func TestCommandOptions(t *testing.T) {
	// Test completion options
	if GeneosCmd.CompletionOptions.DisableDefaultCmd != true {
		t.Error("Default completion command should be disabled")
	}

	// Test disable options
	if !GeneosCmd.DisableAutoGenTag {
		t.Error("Auto generation tag should be disabled")
	}

	if !GeneosCmd.DisableSuggestions {
		t.Error("Suggestions should be disabled")
	}

	if !GeneosCmd.DisableFlagsInUseLine {
		t.Error("Flags in use line should be disabled")
	}
}

func TestCmdKeyType(t *testing.T) {
	// Test CmdKeyType constant
	if CmdKey != "data" {
		t.Errorf("CmdKey = %q, want %q", CmdKey, "data")
	}

	// Test that CmdKeyType is a string type
	var key CmdKeyType = "test"
	if string(key) != "test" {
		t.Error("CmdKeyType should be convertible to string")
	}
}

func TestCmdValType(t *testing.T) {
	// Test CmdValType struct creation
	cmdVal := &CmdValType{
		component: &geneos.RootComponent,
		globals:   true,
		names:     []string{"test1", "test2"},
		params:    []string{"param1"},
	}

	if cmdVal.component != &geneos.RootComponent {
		t.Error("Component should be set correctly")
	}

	if !cmdVal.globals {
		t.Error("Globals should be true")
	}

	if len(cmdVal.names) != 2 {
		t.Errorf("Names length = %d, want 2", len(cmdVal.names))
	}

	if len(cmdVal.params) != 1 {
		t.Errorf("Params length = %d, want 1", len(cmdVal.params))
	}
}

func TestCmddata(t *testing.T) {
	// Test cmddata function with valid context
	cmdVal := &CmdValType{
		component: &geneos.RootComponent,
		globals:   false,
	}

	ctx := context.WithValue(context.Background(), CmdKey, cmdVal)
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	result := cmddata(cmd)
	if result == nil {
		t.Error("cmddata should return the CmdValType from context")
	}

	if result.component != &geneos.RootComponent {
		t.Error("cmddata should return the correct component")
	}

	if result.globals != false {
		t.Error("cmddata should return the correct globals value")
	}
}

func TestCmddataWithNilContext(t *testing.T) {
	// Test cmddata function with nil context
	cmd := &cobra.Command{}
	// Don't set context, should be nil

	result := cmddata(cmd)
	if result != nil {
		t.Error("cmddata should return nil for command without context")
	}
}

func TestCmddataWithInvalidContext(t *testing.T) {
	// Test cmddata function with invalid context value
	ctx := context.WithValue(context.Background(), CmdKey, "invalid")
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	result := cmddata(cmd)
	if result != nil {
		t.Error("cmddata should return nil for invalid context value")
	}
}

func TestAnnotationConstants(t *testing.T) {
	// Test command annotation constants
	constants := map[string]string{
		"CmdWildcardNames": CmdWildcardNames,
		"CmdKeepHosts":     CmdKeepHosts,
		"CmdReplacedBy":    CmdReplacedBy,
		"CmdRequireHome":   CmdRequireHome,
	}

	for name, value := range constants {
		if value == "" {
			t.Errorf("Annotation constant %s should not be empty", name)
		}
	}

	// Test specific expected values
	if CmdWildcardNames != "wildcard" {
		t.Errorf("CmdWildcardNames = %q, want %q", CmdWildcardNames, "wildcard")
	}

	if CmdKeepHosts != "hosts" {
		t.Errorf("CmdKeepHosts = %q, want %q", CmdKeepHosts, "hosts")
	}

	if CmdReplacedBy != "replacedby" {
		t.Errorf("CmdReplacedBy = %q, want %q", CmdReplacedBy, "replacedby")
	}

	if CmdRequireHome != "needshomedir" {
		t.Errorf("CmdRequireHome = %q, want %q", CmdRequireHome, "needshomedir")
	}
}

func TestCommandUsage(t *testing.T) {
	// Test command usage string
	expectedUsage := cordial.ExecutableName() + " COMMAND [flags] [TYPE] [NAME...] [parameters...]"
	if GeneosCmd.Use != expectedUsage {
		t.Errorf("GeneosCmd.Use = %q, want %q", GeneosCmd.Use, expectedUsage)
	}
}

func TestCommandExample(t *testing.T) {
	// Test that command has examples
	if GeneosCmd.Example == "" {
		t.Error("GeneosCmd should have examples")
	}

	// Check for expected example commands
	examples := []string{"init", "ps", "restart"}
	for _, example := range examples {
		if !strings.Contains(GeneosCmd.Example, example) {
			t.Errorf("GeneosCmd.Example should contain %q", example)
		}
	}
}

func TestCommandShort(t *testing.T) {
	// Test command short description
	expected := "Take control of your Geneos environments"
	if GeneosCmd.Short != expected {
		t.Errorf("GeneosCmd.Short = %q, want %q", GeneosCmd.Short, expected)
	}
}

func TestCommandVersion(t *testing.T) {
	// Test that command version is set
	if GeneosCmd.Version != cordial.VERSION {
		t.Errorf("GeneosCmd.Version = %q, want %q", GeneosCmd.Version, cordial.VERSION)
	}
}

func TestCmdValTypeMutex(t *testing.T) {
	// Test that CmdValType includes a mutex for thread safety
	cmdVal := &CmdValType{}

	// Test locking/unlocking
	cmdVal.Lock()
	cmdVal.Unlock()

	// This test mainly ensures the mutex is available and functional
}

func TestGlobalVariables(t *testing.T) {
	// Test package global variables
	if cfgFile != "" {
		t.Log("cfgFile is set to:", cfgFile)
	}

	if Hostname != "" {
		t.Log("Hostname is set to:", Hostname)
	}

	// Test that UserKeyFile is initialized
	if UserKeyFile == nil {
		t.Error("UserKeyFile should be initialized")
	}
}

func TestPackageConstants(t *testing.T) {
	// Test package constants
	if pkgname != "cordial" {
		t.Errorf("pkgname = %q, want %q", pkgname, "cordial")
	}
}

func TestContextOperations(t *testing.T) {
	// Test context operations with command
	cmdVal := &CmdValType{
		globals: true,
		names:   []string{"test"},
	}

	ctx := context.WithValue(context.Background(), CmdKey, cmdVal)
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	// Test retrieving context
	retrievedCtx := cmd.Context()
	if retrievedCtx == nil {
		t.Error("Command context should not be nil")
	}

	// Test retrieving value from context
	value := retrievedCtx.Value(CmdKey)
	if value == nil {
		t.Error("Context should contain CmdKey value")
	}

	if cmdData, ok := value.(*CmdValType); ok {
		if !cmdData.globals {
			t.Error("Retrieved cmdData should have globals=true")
		}
		if len(cmdData.names) != 1 || cmdData.names[0] != "test" {
			t.Error("Retrieved cmdData should have correct names")
		}
	} else {
		t.Error("Context value should be of type *CmdValType")
	}
}