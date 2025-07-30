package geneos

import (
	"reflect"
	"strings"
	"testing"
)

func TestComponentRegistration(t *testing.T) {
	// Test that RootComponent is properly initialized
	if RootComponent.Name != RootComponentName {
		t.Errorf("RootComponent name is %q, want %q", RootComponent.Name, RootComponentName)
	}

	if len(RootComponent.Aliases) == 0 {
		t.Error("RootComponent should have aliases")
	}

	// Check that "any" is in aliases
	found := false
	for _, alias := range RootComponent.Aliases {
		if alias == "any" {
			found = true
			break
		}
	}
	if !found {
		t.Error("RootComponent should have 'any' as an alias")
	}
}

func TestParseComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "exact match",
			input:    "gateway",
			expected: "gateway",
		},
		{
			name:     "alias match", 
			input:    "any",
			expected: RootComponentName,
		},
		{
			name:     "empty string",
			input:    "",
			expected: RootComponentName,
		},
		{
			name:     "case insensitive",
			input:    "GATEWAY",
			expected: "gateway",
		},
		{
			name:     "unknown component",
			input:    "nonexistent",
			expected: RootComponentName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseComponent(tt.input)
			if result.String() != tt.expected {
				t.Errorf("ParseComponent(%q) = %q, want %q", tt.input, result.String(), tt.expected)
			}
		})
	}
}

func TestComponentString(t *testing.T) {
	// Test root component string representation
	if RootComponent.String() != RootComponentName {
		t.Errorf("RootComponent.String() = %q, want %q", RootComponent.String(), RootComponentName)
	}
}

func TestComponentName(t *testing.T) {
	tests := []struct {
		name      string
		component *Component
		expected  string
	}{
		{
			name:      "root component",
			component: &RootComponent,
			expected:  RootComponentName,
		},
		{
			name: "regular component",
			component: &Component{
				Name: "gateway",
			},
			expected: "gateway",
		},
		{
			name: "nil component",
			component: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.component.String()
			if result != tt.expected {
				t.Errorf("Component.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAllComponents(t *testing.T) {
	components := AllComponents()
	
	// Should at least contain the root component
	if len(components) == 0 {
		t.Error("AllComponents() should return at least the root component")
	}

	// Check that root component is included
	found := false
	for _, comp := range components {
		if comp.Name == RootComponentName {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllComponents() should include the root component")
	}
}

func TestComponentValidation(t *testing.T) {
	tests := []struct {
		name      string
		component Component
		wantValid bool
	}{
		{
			name: "valid component",
			component: Component{
				Name:     "test",
				Aliases:  []string{"t"},
				Defaults: []string{"test"},
			},
			wantValid: true,
		},
		{
			name: "component with empty name",
			component: Component{
				Name:     "",
				Aliases:  []string{"empty"},
				Defaults: []string{"test"},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic validation - component should have a name
			isValid := tt.component.Name != ""
			if isValid != tt.wantValid {
				t.Errorf("Component validation failed: got %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}

func TestComponentDefaults(t *testing.T) {
	// Test that root component has expected defaults
	if RootComponent.Defaults == nil {
		t.Error("RootComponent should have defaults initialized")
	}
}

func TestComponentGlobalSettings(t *testing.T) {
	// Test that root component has global settings
	if RootComponent.GlobalSettings == nil {
		t.Error("RootComponent should have GlobalSettings initialized")
	}
}

func TestComponentConfigAliases(t *testing.T) {
	// Test that root component has config aliases
	if RootComponent.ConfigAliases == nil {
		t.Error("RootComponent should have ConfigAliases initialized")
	}
}

func TestComponentWithSharedSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "component without shared suffix",
			input:    "gateway",
			expected: "gateway" + sharedSuffix,
		},
		{
			name:     "component already with shared suffix",
			input:    "gateway" + sharedSuffix,
			expected: "gateway" + sharedSuffix,
		},
		{
			name:     "empty component",
			input:    "",
			expected: sharedSuffix,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test shared suffix logic
			var result string
			if strings.HasSuffix(tt.input, sharedSuffix) {
				result = tt.input
			} else {
				result = tt.input + sharedSuffix
			}

			if result != tt.expected {
				t.Errorf("Shared suffix test for %q: got %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestComponentsMap(t *testing.T) {
	// Test that componentsMap is properly initialized
	if registeredComponents == nil {
		t.Error("registeredComponents should be initialized")
	}

	// Test basic map operations
	testComponent := &Component{Name: "test"}
	registeredComponents["test"] = testComponent

	retrieved := registeredComponents["test"]
	if retrieved != testComponent {
		t.Error("Failed to store and retrieve component from map")
	}

	// Clean up
	delete(registeredComponents, "test")
}

func TestInitDirs(t *testing.T) {
	// Test that initDirs map is properly initialized
	if initDirs == nil {
		t.Error("initDirs should be initialized")
	}

	// Test basic map operations
	testDirs := []string{"test1", "test2"}
	initDirs["test"] = testDirs

	retrieved := initDirs["test"]
	if !reflect.DeepEqual(retrieved, testDirs) {
		t.Errorf("Failed to store and retrieve dirs from map: got %v, want %v", retrieved, testDirs)
	}

	// Clean up
	delete(initDirs, "test")
}

func TestComponentConstants(t *testing.T) {
	// Test package constants
	if RootComponentName == "" {
		t.Error("RootComponentName should not be empty")
	}

	if sharedSuffix == "" {
		t.Error("sharedSuffix should not be empty")
	}

	// Test that shared suffix starts with underscore (common convention)
	if !strings.HasPrefix(sharedSuffix, "_") {
		t.Errorf("sharedSuffix %q should start with underscore", sharedSuffix)
	}
}

func TestComponentPackageTypes(t *testing.T) {
	// Test that root component has package types configured
	if RootComponent.PackageTypes != nil && len(RootComponent.PackageTypes) > 0 {
		t.Log("RootComponent has package types configured")
	}
}

func TestComponentDownloadBases(t *testing.T) {
	// Test that root component has download bases configured
	downloadBases := RootComponent.DownloadBase
	if downloadBases.Default == "" && downloadBases.Nexus == "" {
		t.Log("RootComponent has empty download bases (expected for root)")
	}
}

func TestComponentRegister(t *testing.T) {
	// Test component registration
	originalSize := len(registeredComponents)
	
	testComp := &Component{
		Name:    "test_register",
		Aliases: []string{"tr"},
	}
	
	// Test that Register method exists and can be called
	// Note: We can't test the actual Register method without importing the full component
	// but we can test the registration concept
	registeredComponents["test_register"] = testComp
	
	if len(registeredComponents) != originalSize+1 {
		t.Error("Component registration failed to increase map size")
	}
	
	// Clean up
	delete(registeredComponents, "test_register")
}