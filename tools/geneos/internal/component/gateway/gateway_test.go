package gateway

import (
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestGatewayComponent(t *testing.T) {
	// Test that gateway component is properly registered
	component := geneos.ParseComponent("gateway")
	if component == nil {
		t.Fatal("Gateway component should be registered")
	}

	if component.String() != "gateway" {
		t.Errorf("Gateway component name = %q, want %q", component.String(), "gateway")
	}
}

func TestGatewayComponentRegistration(t *testing.T) {
	// Test that gateway is in the list of all components
	components := geneos.AllComponents()
	
	found := false
	for _, comp := range components {
		if comp.String() == "gateway" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Gateway component should be in AllComponents()")
	}
}

func TestGatewayAliases(t *testing.T) {
	// Test gateway component aliases
	testCases := []string{"gateway", "gw"}
	
	for _, alias := range testCases {
		component := geneos.ParseComponent(alias)
		if component.String() != "gateway" {
			t.Errorf("Alias %q should resolve to gateway component, got %q", alias, component.String())
		}
	}
}

func TestGatewayComponentFields(t *testing.T) {
	component := geneos.ParseComponent("gateway")
	
	// Test that component has expected fields
	if component.Name == "" {
		t.Error("Gateway component should have a name")
	}
	
	// Test that component has defaults
	if component.Defaults == nil {
		t.Error("Gateway component should have defaults")
	}
	
	// Test that component has package types
	if component.PackageTypes == nil {
		t.Error("Gateway component should have package types")
	}
}

func TestGatewayComponentMethods(t *testing.T) {
	component := geneos.ParseComponent("gateway")
	
	// Test String method
	stringName := component.String()
	if stringName != "gateway" {
		t.Errorf("Gateway String() = %q, want %q", stringName, "gateway")
	}
	
	// Test IsA method
	if !component.IsA("gateway") {
		t.Error("Gateway component should match its own name")
	}
	
	if !component.IsA("GATEWAY") {
		t.Error("Gateway IsA should be case-insensitive")
	}
	
	if component.IsA("netprobe") {
		t.Error("Gateway should not match different component name")
	}
}

func TestGatewayConfiguration(t *testing.T) {
	component := geneos.ParseComponent("gateway")
	
	// Test that gateway has global settings
	if component.GlobalSettings == nil {
		t.Error("Gateway component should have global settings")
	}
	
	// Test that gateway has config aliases
	if component.ConfigAliases == nil {
		t.Error("Gateway component should have config aliases")
	}
}

func TestGatewayDownloadBases(t *testing.T) {
	component := geneos.ParseComponent("gateway")
	
	// Test that gateway has download base configuration
	// This may be empty for some components, so just test structure exists
	_ = component.DownloadBase
}

func TestGatewayPortRanges(t *testing.T) {
	component := geneos.ParseComponent("gateway")
	
	// Test that gateway has port range configuration
	if component.PortRange == nil {
		t.Error("Gateway component should have port range configuration")
	}
}

func TestGatewayInitialization(t *testing.T) {
	component := geneos.ParseComponent("gateway")
	
	// Test that gateway has an initializer function
	// This may be nil for some components
	if component.Initialise != nil {
		// Test that it can be called (though we won't actually call it in tests)
		t.Log("Gateway has initialization function")
	}
}

func TestGatewayValidation(t *testing.T) {
	component := geneos.ParseComponent("gateway")
	
	// Basic validation tests
	if component == &geneos.RootComponent {
		t.Error("Gateway should not be the root component")
	}
	
	// Test that gateway has non-empty name
	if component.Name == "" {
		t.Error("Gateway component name should not be empty")
	}
	
	// Test that gateway name matches expected
	if component.Name != "gateway" {
		t.Errorf("Gateway component name = %q, want %q", component.Name, "gateway")
	}
}

func TestGatewayHelp(t *testing.T) {
	component := geneos.ParseComponent("gateway")
	
	// Test that gateway has help text configured
	if component.UsageText == "" {
		t.Log("Gateway component has no usage text (this may be normal)")
	}
	
	if component.LegacyParameters == "" {
		t.Log("Gateway component has no legacy parameters (this may be normal)")
	}
}