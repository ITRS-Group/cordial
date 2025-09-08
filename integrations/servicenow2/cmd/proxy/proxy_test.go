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

package proxy

import (
	"os"
	"testing"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/spf13/cobra"
)

func TestProxyCommand(t *testing.T) {
	// Test that the proxy command is properly initialized
	if routerCmd == nil {
		t.Fatal("Expected routerCmd to be initialized, got nil")
	}

	if routerCmd.Use != "proxy" {
		t.Errorf("Expected routerCmd.Use to be 'proxy', got %q", routerCmd.Use)
	}

	if routerCmd.Short != "Run a ServiceNow integration proxy" {
		t.Errorf("Expected routerCmd.Short to be 'Run a ServiceNow integration proxy', got %q", routerCmd.Short)
	}

	// Test that the command has a Run function (not RunE)
	if routerCmd.Run == nil {
		t.Error("Expected routerCmd to have a Run function")
	}

	// Test that SilenceUsage is set
	if !routerCmd.SilenceUsage {
		t.Error("Expected routerCmd.SilenceUsage to be true")
	}
}

func TestProxyFlags(t *testing.T) {
	// Test that proxy command flags are properly set up
	flags := routerCmd.Flags()

	// Check for daemon flag
	daemonFlag := flags.Lookup("daemon")
	if daemonFlag == nil {
		t.Error("Expected 'daemon' flag to be defined")
	} else {
		if daemonFlag.Shorthand != "D" {
			t.Errorf("Expected daemon flag shorthand to be 'D', got %q", daemonFlag.Shorthand)
		}
		if daemonFlag.Usage != "Daemonise the proxy process" {
			t.Errorf("Expected daemon flag usage to be 'Daemonise the proxy process', got %q", daemonFlag.Usage)
		}
	}

	// Check for logfile flag (persistent flag)
	persistentFlags := routerCmd.PersistentFlags()
	logfileFlag := persistentFlags.Lookup("logfile")
	if logfileFlag == nil {
		t.Error("Expected 'logfile' persistent flag to be defined")
	} else {
		if logfileFlag.Shorthand != "l" {
			t.Errorf("Expected logfile flag shorthand to be 'l', got %q", logfileFlag.Shorthand)
		}
		expectedUsage := "Write logs to `file`. Use '-' for console or " + os.DevNull + " for none"
		if logfileFlag.Usage != expectedUsage {
			t.Errorf("Expected logfile flag usage to be %q, got %q", expectedUsage, logfileFlag.Usage)
		}
	}

	// Test that flags are not sorted
	if routerCmd.Flags().SortFlags {
		t.Error("Expected proxy command flags to not be sorted")
	}
}

func TestProxyCommandAddedToRoot(t *testing.T) {
	// Test that the proxy command is added to the root command
	found := false
	for _, subCmd := range cmd.RootCmd.Commands() {
		if subCmd.Name() == "proxy" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected proxy command to be added to root command")
	}
}

func TestGlobalVariables(t *testing.T) {
	// Test initial values of global variables
	if daemon {
		t.Error("Expected daemon to be false initially")
	}

	// Note: logFile is initialized to "-" by default in the flag definition
	if logFile != "-" {
		t.Errorf("Expected logFile to be '-' initially, got %q", logFile)
	}
}

func TestLogFileDefaultValue(t *testing.T) {
	// Test that logfile flag has correct default value
	persistentFlags := routerCmd.PersistentFlags()
	logfileFlag := persistentFlags.Lookup("logfile")
	if logfileFlag == nil {
		t.Fatal("Expected 'logfile' flag to be defined")
	}

	// The default value should be "-" (console)
	if logfileFlag.DefValue != "-" {
		t.Errorf("Expected logfile flag default value to be '-', got %q", logfileFlag.DefValue)
	}
}

func TestDaemonFlagDefaultValue(t *testing.T) {
	// Test that daemon flag has correct default value
	flags := routerCmd.Flags()
	daemonFlag := flags.Lookup("daemon")
	if daemonFlag == nil {
		t.Fatal("Expected 'daemon' flag to be defined")
	}

	// The default value should be "false"
	if daemonFlag.DefValue != "false" {
		t.Errorf("Expected daemon flag default value to be 'false', got %q", daemonFlag.DefValue)
	}
}

func TestCommandStructure(t *testing.T) {
	// Test command properties
	if len(routerCmd.Aliases) > 0 {
		t.Errorf("Expected proxy command to have no aliases, got %v", routerCmd.Aliases)
	}

	// Test that the command is runnable
	if !routerCmd.Runnable() {
		t.Error("Expected proxy command to be runnable")
	}
}

func TestCommandLongDescription(t *testing.T) {
	// Test that the command has a long description
	if routerCmd.Long == "" {
		t.Error("Expected proxy command to have a long description")
	}

	// The long description should contain information about the proxy
	if !contains(routerCmd.Long, "proxy") {
		t.Error("Expected long description to mention 'proxy'")
	}
}

func TestCommandHelpTemplate(t *testing.T) {
	// Test that help can be generated without errors
	help := routerCmd.UsageString()
	if help == "" {
		t.Error("Expected proxy command to generate help text")
	}

	// Debug: log the actual help content
	t.Logf("Help text: %s", help)

	// Should contain basic command information
	if !contains(help, "proxy") {
		t.Error("Expected help text to contain 'proxy'")
	}

	if !contains(help, "Daemonise the proxy process") {
		t.Error("Expected help text to contain flag descriptions")
	}
}

func TestFlagOrder(t *testing.T) {
	// Test that SortFlags is false to preserve flag order
	if routerCmd.Flags().SortFlags {
		t.Error("Expected flags to not be sorted (SortFlags should be false)")
	}
}

func TestCommandName(t *testing.T) {
	// Test that the command name is correct
	if routerCmd.Name() != "proxy" {
		t.Errorf("Expected command name to be 'proxy', got %q", routerCmd.Name())
	}
}

func TestCommandParent(t *testing.T) {
	// Test that the command is properly attached to root
	if routerCmd.Parent() != cmd.RootCmd {
		t.Error("Expected proxy command to be child of root command")
	}
}

func TestSubcommands(t *testing.T) {
	// Test if the proxy command has any subcommands
	subcommands := routerCmd.Commands()

	// For now, the proxy command shouldn't have subcommands based on the current structure
	// This test documents the current state and will need updating if subcommands are added
	if len(subcommands) > 0 {
		t.Logf("Proxy command has %d subcommands: %v", len(subcommands), getCommandNames(subcommands))
	}
}

func TestCommandValidation(t *testing.T) {
	// Test that the command structure is valid
	if err := routerCmd.ValidateArgs([]string{}); err != nil {
		t.Errorf("Expected proxy command to validate with no args, got error: %v", err)
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

// Helper function to get command names from a slice of commands
func getCommandNames(commands []*cobra.Command) []string {
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	return names
}
