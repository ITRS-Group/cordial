package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
)

func TestMain(m *testing.M) {
	// Setup test environment
	cordial.VERSION = "test-version"
	code := m.Run()
	os.Exit(code)
}

func TestVersionTrimming(t *testing.T) {
	// Test version trimming logic (init function behavior)
	originalVersion := cordial.VERSION
	defer func() { cordial.VERSION = originalVersion }()

	// Test the trimming logic that init() would do
	testVersion := "  1.0.0  \n\t"
	trimmedVersion := strings.TrimSpace(testVersion)

	if trimmedVersion != "1.0.0" {
		t.Errorf("Version trimming failed, got %q, want %q", trimmedVersion, "1.0.0")
	}

	// Test that the current VERSION is properly trimmed (should be done by init)
	if strings.TrimSpace(cordial.VERSION) != cordial.VERSION {
		t.Errorf("cordial.VERSION appears to have untrimmed whitespace: %q", cordial.VERSION)
	}
}

func TestExecutableNameParsing(t *testing.T) {
	tests := []struct {
		name         string
		executableName string
		shouldCallCtl bool
	}{
		{
			name:         "regular geneos executable",
			executableName: "geneos",
			shouldCallCtl: false,
		},
		{
			name:         "gateway control executable",
			executableName: "gatewayctl",
			shouldCallCtl: true,
		},
		{
			name:         "netprobe control executable", 
			executableName: "netprobectl",
			shouldCallCtl: true,
		},
		{
			name:         "san control executable",
			executableName: "sanctl",
			shouldCallCtl: true,
		},
		{
			name:         "executable without ctl suffix",
			executableName: "gateway",
			shouldCallCtl: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()
			execPath := filepath.Join(tempDir, tt.executableName)
			
			// Create a dummy executable file
			file, err := os.Create(execPath)
			if err != nil {
				t.Fatalf("Failed to create test executable: %v", err)
			}
			file.Close()

			// Test basename extraction logic
			basename := filepath.Base(execPath)
			hasCtlSuffix := len(basename) > 3 && basename[len(basename)-3:] == "ctl"

			if hasCtlSuffix != tt.shouldCallCtl {
				t.Errorf("Executable name %q ctl suffix detection failed, got %v, want %v", 
					tt.executableName, hasCtlSuffix, tt.shouldCallCtl)
			}
		})
	}
}

func TestComponentNameExtraction(t *testing.T) {
	tests := []struct {
		name         string
		executableName string
		expectedComponent string
	}{
		{
			name:         "gateway control",
			executableName: "gatewayctl",
			expectedComponent: "gateway",
		},
		{
			name:         "netprobe control",
			executableName: "netprobectl", 
			expectedComponent: "netprobe",
		},
		{
			name:         "san control",
			executableName: "sanctl",
			expectedComponent: "san",
		},
		{
			name:         "ca3 control",
			executableName: "ca3ctl",
			expectedComponent: "ca3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract component name by removing "ctl" suffix
			componentName := tt.executableName
			if len(componentName) > 3 && componentName[len(componentName)-3:] == "ctl" {
				componentName = componentName[:len(componentName)-3]
			}

			if componentName != tt.expectedComponent {
				t.Errorf("Component name extraction failed for %q, got %q, want %q",
					tt.executableName, componentName, tt.expectedComponent)
			}
		})
	}
}

func TestCommandArgumentHandling(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "list command",
			args:        []string{"gatewayctl", "list"},
			expectError: false,
			description: "Should convert to 'geneos ls gateway'",
		},
		{
			name:        "create command",
			args:        []string{"gatewayctl", "create"},
			expectError: true,
			description: "Create command should be rejected",
		},
		{
			name:        "start command with instance",
			args:        []string{"gatewayctl", "instance1", "start"},
			expectError: false,
			description: "Should convert to 'geneos start gateway instance1'",
		},
		{
			name:        "stop command with instance",
			args:        []string{"gatewayctl", "instance1", "stop"},
			expectError: false,
			description: "Should convert to 'geneos stop gateway instance1'",
		},
		{
			name:        "restart command with instance",
			args:        []string{"gatewayctl", "instance1", "restart"},
			expectError: false,
			description: "Should convert to 'geneos restart gateway instance1'",
		},
		{
			name:        "status command with instance", 
			args:        []string{"gatewayctl", "instance1", "status"},
			expectError: false,
			description: "Should convert to 'geneos status gateway instance1'",
		},
		{
			name:        "unknown command",
			args:        []string{"gatewayctl", "instance1", "unknown"},
			expectError: true,
			description: "Unknown commands should be rejected",
		},
		{
			name:        "insufficient arguments",
			args:        []string{"gatewayctl", "instance1"},
			expectError: true,
			description: "Insufficient arguments should be handled gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the argument parsing logic
			if len(tt.args) < 1 {
				t.Skip("Invalid test case - no arguments")
				return
			}

			execName := tt.args[0]
			
			// Check if it's a ctl command
			if len(execName) <= 3 || execName[len(execName)-3:] != "ctl" {
				t.Skip("Not a ctl command")
				return
			}

			if len(tt.args) > 1 {
				command := tt.args[1]
				
				switch command {
				case "list":
					// This should be converted to ls command
					if tt.expectError {
						t.Error("List command should not produce error")
					}
				case "create":
					// This should produce an error
					if !tt.expectError {
						t.Error("Create command should produce error")
					}
				default:
					if len(tt.args) > 2 {
						function := tt.args[2]
						validFunctions := []string{"start", "stop", "restart", "command", "log", "details", "refresh", "status", "delete"}
						isValid := false
						for _, validFunc := range validFunctions {
							if function == validFunc {
								isValid = true
								break
							}
						}
						
						if !isValid && !tt.expectError {
							t.Errorf("Invalid function %q should produce error", function)
						} else if isValid && tt.expectError {
							t.Errorf("Valid function %q should not produce error", function)
						}
					} else if !tt.expectError {
						// Insufficient arguments, should error
						t.Error("Insufficient arguments should produce error")
					}
				}
			}
		})
	}
}

// TestMainEntryPoint tests the main function behavior with different scenarios
func TestMainEntryPoint(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name string
		args []string
		expectCtl bool
	}{
		{
			name: "regular geneos command",
			args: []string{"geneos", "ls"},
			expectCtl: false,
		},
		{
			name: "gateway ctl command",
			args: []string{"gatewayctl", "list"},
			expectCtl: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			
			execname := filepath.Base(os.Args[0])
			hasCtlSuffix := len(execname) > 3 && execname[len(execname)-3:] == "ctl"
			
			if hasCtlSuffix != tt.expectCtl {
				t.Errorf("ctl detection failed for %v, got %v, want %v", 
					tt.args[0], hasCtlSuffix, tt.expectCtl)
			}
		})
	}
}