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
	"os"
	"testing"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
)

func TestMainFunction(t *testing.T) {
	// Test that main function exists and can be called
	// We can't directly test main() as it calls cmd.Execute() which may exit
	// Instead, we test that the command structure is properly set up

	if cmd.RootCmd == nil {
		t.Fatal("Expected RootCmd to be initialized by imported packages")
	}

	// Test that the root command has the expected structure
	if cmd.RootCmd.Use != "servicenow2" {
		t.Errorf("Expected root command use to be 'servicenow2', got %q", cmd.RootCmd.Use)
	}

	// Test that subcommands are registered
	commands := cmd.RootCmd.Commands()
	if len(commands) == 0 {
		t.Error("Expected subcommands to be registered")
	}

	// Verify expected commands exist
	expectedCommands := map[string]bool{
		"client": false,
		"proxy":  false,
	}

	for _, command := range commands {
		if _, exists := expectedCommands[command.Name()]; exists {
			expectedCommands[command.Name()] = true
		}
	}

	for cmdName, found := range expectedCommands {
		if !found {
			t.Errorf("Expected command %q to be registered", cmdName)
		}
	}
}

func TestPackageImports(t *testing.T) {
	// Test that the expected packages are imported by checking if their
	// initialization code has run properly

	// The cmd package should be imported and initialized
	if cmd.RootCmd == nil {
		t.Error("cmd package not properly imported/initialized")
	}

	// Check that client and proxy subcommands are available due to blank imports
	foundClient := false
	foundProxy := false

	for _, command := range cmd.RootCmd.Commands() {
		switch command.Name() {
		case "client":
			foundClient = true
		case "proxy":
			foundProxy = true
		}
	}

	if !foundClient {
		t.Error("client package not properly imported (client command not found)")
	}

	if !foundProxy {
		t.Error("proxy package not properly imported (proxy command not found)")
	}
}

func TestCommandExecution(t *testing.T) {
	// Test command execution without actually running main()
	// We'll test the command structure and validate it can be executed

	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test help command execution
	os.Args = []string{"servicenow2", "--help"}

	// We can't actually call cmd.Execute() in tests as it may call os.Exit()
	// Instead, we validate the command structure

	// Test that help flag is available
	helpFlag := cmd.RootCmd.PersistentFlags().Lookup("help")
	if helpFlag == nil {
		t.Error("Expected help flag to be available")
	}

	// Test that version is set
	if cmd.RootCmd.Version == "" {
		t.Error("Expected version to be set")
	}

	// Test command validation
	if err := cmd.RootCmd.ValidateArgs([]string{}); err != nil {
		t.Errorf("Root command validation failed: %v", err)
	}
}

func TestApplicationStructure(t *testing.T) {
	// Test that the application is properly structured
	
	// Root command should exist
	if cmd.RootCmd == nil {
		t.Fatal("Root command should be initialized")
	}

	// Should have the correct application name
	if cmd.RootCmd.Use != "servicenow2" {
		t.Errorf("Expected application name 'servicenow2', got %q", cmd.RootCmd.Use)
	}

	// Should have a description
	if cmd.RootCmd.Short == "" {
		t.Error("Expected application to have a short description")
	}

	// Should have version information
	if cmd.RootCmd.Version == "" {
		t.Error("Expected application to have version information")
	}

	// Should have persistent flags
	if !cmd.RootCmd.PersistentFlags().HasAvailableFlags() {
		t.Error("Expected application to have persistent flags")
	}
}

func TestSubcommandStructure(t *testing.T) {
	// Test the structure of subcommands
	commands := cmd.RootCmd.Commands()

	for _, command := range commands {
		// Each command should have a name
		if command.Name() == "" {
			t.Error("Command should have a name")
		}

		// Each command should have a short description
		if command.Short == "" {
			t.Errorf("Command %q should have a short description", command.Name())
		}

		// Each command should be runnable (have a Run or RunE function)
		if !command.Runnable() {
			t.Errorf("Command %q should be runnable", command.Name())
		}
	}
}

func TestCommandLineCompatibility(t *testing.T) {
	// Test that the application follows expected command line conventions

	// Should support --help
	helpFlag := cmd.RootCmd.PersistentFlags().Lookup("help")
	if helpFlag == nil {
		t.Error("Expected --help flag to be available")
	}

	// Should support --version (through cobra's built-in version flag)
	if cmd.RootCmd.Version == "" {
		t.Error("Expected version to be available")
	}

	// Should support --conf flag for configuration
	confFlag := cmd.RootCmd.PersistentFlags().Lookup("conf")
	if confFlag == nil {
		t.Error("Expected --conf flag to be available")
	}

	// Should support --debug flag
	debugFlag := cmd.RootCmd.PersistentFlags().Lookup("debug")
	if debugFlag == nil {
		t.Error("Expected --debug flag to be available")
	}
}

func TestApplicationMetadata(t *testing.T) {
	// Test application metadata

	// Should have completion disabled
	if !cmd.RootCmd.CompletionOptions.DisableDefaultCmd {
		t.Error("Expected completion to be disabled")
	}

	// Should have auto-gen tag disabled
	if !cmd.RootCmd.DisableAutoGenTag {
		t.Error("Expected auto-gen tag to be disabled")
	}

	// Should silence usage on errors
	if !cmd.RootCmd.SilenceUsage {
		t.Error("Expected usage to be silenced")
	}
}