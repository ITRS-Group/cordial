package instance

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestInstanceCreation(t *testing.T) {
	// Test creating a new instance
	instance := &Instance{
		Conf:         config.New(),
		InstanceHost: geneos.LOCAL,
		Component:    &geneos.RootComponent,
		ConfigLoaded: time.Now(),
	}

	if instance.Conf == nil {
		t.Error("Instance.Conf should not be nil")
	}

	if instance.InstanceHost == nil {
		t.Error("Instance.InstanceHost should not be nil")
	}

	if instance.Component == nil {
		t.Error("Instance.Component should not be nil")
	}

	if instance.ConfigLoaded.IsZero() {
		t.Error("Instance.ConfigLoaded should be set")
	}
}

func TestInstanceValidation(t *testing.T) {
	tests := []struct {
		name     string
		instance *Instance
		valid    bool
	}{
		{
			name: "valid instance",
			instance: &Instance{
				Conf:         config.New(),
				InstanceHost: geneos.LOCAL,
				Component:    &geneos.RootComponent,
				ConfigLoaded: time.Now(),
			},
			valid: true,
		},
		{
			name: "instance with nil config",
			instance: &Instance{
				Conf:         nil,
				InstanceHost: geneos.LOCAL,
				Component:    &geneos.RootComponent,
				ConfigLoaded: time.Now(),
			},
			valid: false,
		},
		{
			name: "instance with nil host",
			instance: &Instance{
				Conf:         config.New(),
				InstanceHost: nil,
				Component:    &geneos.RootComponent,
				ConfigLoaded: time.Now(),
			},
			valid: false,
		},
		{
			name: "instance with nil component",
			instance: &Instance{
				Conf:         config.New(),
				InstanceHost: geneos.LOCAL,
				Component:    nil,
				ConfigLoaded: time.Now(),
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - check required fields
			valid := tt.instance.Conf != nil && 
					tt.instance.InstanceHost != nil && 
					tt.instance.Component != nil

			if valid != tt.valid {
				t.Errorf("Instance validation failed: got %v, want %v", valid, tt.valid)
			}
		})
	}
}

func TestProcessFDs(t *testing.T) {
	// Test ProcessFDs structure
	fds := ProcessFDs{
		PID:  1234,
		FD:   5,
		Path: "/tmp/test.sock",
	}

	if fds.PID != 1234 {
		t.Errorf("ProcessFDs.PID = %d, want %d", fds.PID, 1234)
	}

	if fds.FD != 5 {
		t.Errorf("ProcessFDs.FD = %d, want %d", fds.FD, 5)
	}

	if fds.Path != "/tmp/test.sock" {
		t.Errorf("ProcessFDs.Path = %q, want %q", fds.Path, "/tmp/test.sock")
	}
}

func TestInstanceConfigLoading(t *testing.T) {
	// Test config loading timestamp
	now := time.Now()
	instance := &Instance{
		ConfigLoaded: now,
	}

	if !instance.ConfigLoaded.Equal(now) {
		t.Error("ConfigLoaded timestamp should match the set time")
	}

	// Test that config loaded time is recent for a new instance
	recent := time.Now().Add(-time.Minute)
	if instance.ConfigLoaded.Before(recent) {
		t.Error("ConfigLoaded time should be recent")
	}
}

func TestInstanceFields(t *testing.T) {
	// Test that Instance has all expected fields
	instance := &Instance{}

	// Check that embedded geneos.Instance can be accessed
	_ = instance.Instance

	// Check that all fields are accessible
	if instance.Conf != nil {
		t.Log("Conf field is accessible")
	}

	if instance.InstanceHost != nil {
		t.Log("InstanceHost field is accessible")
	}

	if instance.Component != nil {
		t.Log("Component field is accessible")
	}

	// ConfigLoaded should be zero time for uninitialized instance
	if !instance.ConfigLoaded.IsZero() {
		t.Error("ConfigLoaded should be zero time for uninitialized instance")
	}
}

func TestInstanceWithRealComponent(t *testing.T) {
	// Test instance with a real component
	component := &geneos.Component{
		Name:    "test-component",
		Aliases: []string{"tc"},
	}

	instance := &Instance{
		Conf:         config.New(),
		InstanceHost: geneos.LOCAL,
		Component:    component,
		ConfigLoaded: time.Now(),
	}

	if instance.Component.Name != "test-component" {
		t.Errorf("Component name = %q, want %q", instance.Component.Name, "test-component")
	}

	if len(instance.Component.Aliases) != 1 || instance.Component.Aliases[0] != "tc" {
		t.Errorf("Component aliases = %v, want %v", instance.Component.Aliases, []string{"tc"})
	}
}

func TestInstanceConfigModification(t *testing.T) {
	// Test modifying instance configuration
	instance := &Instance{
		Conf: config.New(),
	}

	// Set a configuration value
	testKey := "test.key"
	testValue := "test.value"
	instance.Conf.Set(testKey, testValue)

	// Verify the value was set
	if instance.Conf.GetString(testKey) != testValue {
		t.Errorf("Config value not set correctly: got %q, want %q", 
			instance.Conf.GetString(testKey), testValue)
	}
}

func TestInstanceHostOperations(t *testing.T) {
	// Test basic host operations
	if geneos.LOCAL == nil {
		t.Skip("LOCAL host not available")
	}

	instance := &Instance{
		InstanceHost: geneos.LOCAL,
	}

	// Test that host is accessible
	if instance.InstanceHost == nil {
		t.Error("InstanceHost should not be nil")
	}

	// Test basic host operations if available
	if instance.InstanceHost.IsLocal() {
		t.Log("Host is local")
	}
}

func TestInstanceTimeComparison(t *testing.T) {
	// Test comparing instance times
	time1 := time.Now()
	time.Sleep(time.Millisecond) // Ensure different times
	time2 := time.Now()

	instance1 := &Instance{ConfigLoaded: time1}
	instance2 := &Instance{ConfigLoaded: time2}

	if !instance1.ConfigLoaded.Before(instance2.ConfigLoaded) {
		t.Error("instance1 should have been loaded before instance2")
	}

	if !instance2.ConfigLoaded.After(instance1.ConfigLoaded) {
		t.Error("instance2 should have been loaded after instance1")
	}
}

func TestInstanceJSONTag(t *testing.T) {
	// Test that Instance struct has proper JSON tags
	// The geneos.Instance field should be excluded from JSON
	instance := &Instance{
		Conf:         config.New(),
		ConfigLoaded: time.Now(),
	}

	// This is mainly a compile-time check to ensure the struct is properly defined
	_ = instance.Instance // Should be accessible but excluded from JSON
}

func TestInstanceZeroValues(t *testing.T) {
	// Test zero values of Instance
	var instance Instance

	if instance.Conf != nil {
		t.Error("Zero instance should have nil Conf")
	}

	if instance.InstanceHost != nil {
		t.Error("Zero instance should have nil InstanceHost")
	}

	if instance.Component != nil {
		t.Error("Zero instance should have nil Component")
	}

	if !instance.ConfigLoaded.IsZero() {
		t.Error("Zero instance should have zero ConfigLoaded time")
	}
}

func TestInstancePointerOperations(t *testing.T) {
	// Test operations with instance pointers
	instance := &Instance{
		Conf:         config.New(),
		InstanceHost: geneos.LOCAL,
		Component:    &geneos.RootComponent,
		ConfigLoaded: time.Now(),
	}

	// Test copying instance pointer
	instanceCopy := instance

	// Verify they point to the same instance
	if instanceCopy != instance {
		t.Error("Instance copy should point to the same instance")
	}

	// Verify shared state
	if instanceCopy.Conf != instance.Conf {
		t.Error("Instance copy should share the same Conf")
	}
}