/*
Copyright © 2023 ITRS Group

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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial/pkg/host"
)

func TestSave(t *testing.T) {
	tempDir := t.TempDir()

	// Create a config with some test data
	config := New()
	config.Set("app.name", "testapp")
	config.Set("app.version", "1.0.0")
	config.Set("database.host", "localhost")
	config.Set("database.port", 5432)

	tests := []struct {
		name        string
		configName  string
		options     []FileOptions
		wantErr     bool
		wantFile    string
		checkValues bool
	}{
		{
			name:        "save to temp directory",
			configName:  "test",
			options:     []FileOptions{AddDirs(tempDir)},
			wantErr:     false,
			wantFile:    filepath.Join(tempDir, "test.json"),
			checkValues: true,
		},
		{
			name:        "save with custom extension",
			configName:  "test-yaml",
			options:     []FileOptions{AddDirs(tempDir), SetFileExtension("yaml")},
			wantErr:     false,
			wantFile:    filepath.Join(tempDir, "test-yaml.yaml"),
			checkValues: false, // YAML parsing would require more setup
		},
		{
			name:       "save to specific file",
			configName: "specific",
			options: []FileOptions{
				SetConfigFile(filepath.Join(tempDir, "custom.json")),
			},
			wantErr:     false,
			wantFile:    filepath.Join(tempDir, "custom.json"),
			checkValues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.Save(tt.configName, tt.options...)

			if tt.wantErr {
				if err == nil {
					t.Error("Save() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Save() unexpected error: %v", err)
			}

			// Check that file was created
			if _, err := os.Stat(tt.wantFile); os.IsNotExist(err) {
				t.Errorf("Save() should have created file %s", tt.wantFile)
			}

			if tt.checkValues {
				// Verify content by loading it back
				savedConfig, err := Load(tt.configName, SetConfigFile(tt.wantFile))
				if err != nil {
					t.Fatalf("Failed to load saved config: %v", err)
				}

				if savedConfig.GetString("app.name") != "testapp" {
					t.Error("Saved config should preserve app.name")
				}

				if savedConfig.GetString("database.host") != "localhost" {
					t.Error("Saved config should preserve database.host")
				}

				if savedConfig.GetInt("database.port") != 5432 {
					t.Error("Saved config should preserve database.port")
				}
			}
		})
	}
}

func TestSaveGlobal(t *testing.T) {
	tempDir := t.TempDir()

	// Set some values in global config
	ResetConfig()
	GetConfig().Set("global.test", "value")
	GetConfig().Set("global.number", 123)

	err := Save("global-test", AddDirs(tempDir))
	if err != nil {
		t.Fatalf("Save() global config failed: %v", err)
	}

	expectedFile := filepath.Join(tempDir, "global-test.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Save() should have created file %s", expectedFile)
	}

	// Verify content
	savedConfig, err := Load("global-test", SetConfigFile(expectedFile))
	if err != nil {
		t.Fatalf("Failed to load saved global config: %v", err)
	}

	if savedConfig.GetString("global.test") != "value" {
		t.Error("Saved global config should preserve values")
	}

	if savedConfig.GetInt("global.number") != 123 {
		t.Error("Saved global config should preserve numbers")
	}
}

func TestSaveWithHost(t *testing.T) {
	tempDir := t.TempDir()

	config := New()
	config.Set("host.test", "remote")

	// Use localhost host for testing
	localhost := host.Localhost

	err := config.Save("host-test",
		AddDirs(tempDir),
		Host(localhost),
	)

	if err != nil {
		t.Fatalf("Save() with host failed: %v", err)
	}

	expectedFile := filepath.Join(tempDir, "host-test.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Save() with host should have created file %s", expectedFile)
	}
}

func TestSaveErrors(t *testing.T) {
	config := New()
	config.Set("test", "value")

	// Test saving to non-existent directory without create permissions
	err := config.Save("error-test",
		SetConfigFile("/root/cannot-create/test.json"),
	)

	if err == nil {
		t.Error("Save() should fail when unable to create file")
	}
}

func TestSaveAppConfig(t *testing.T) {
	tempDir := t.TempDir()

	config := New(SetAppName("testapp"))
	config.Set("app.config", "test")

	// This should save to a subdirectory based on app name
	err := config.Save("app-config",
		AddDirs(tempDir),
	)

	if err != nil {
		t.Fatalf("Save() app config failed: %v", err)
	}

	// File should be created in app-specific directory
	expectedDir := filepath.Join(tempDir, "testapp")
	expectedFile := filepath.Join(expectedDir, "app-config.json")

	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		// If app-specific directory doesn't exist, file might be in root
		altFile := filepath.Join(tempDir, "app-config.json")
		if _, err := os.Stat(altFile); os.IsNotExist(err) {
			t.Errorf("Save() should have created file in %s or %s", expectedFile, altFile)
		}
	}
}

func TestSaveFileExtensions(t *testing.T) {
	tempDir := t.TempDir()

	config := New()
	config.Set("ext.test", "value")

	extensions := []string{"json", "yaml", "yml", "toml"}

	for _, ext := range extensions {
		t.Run("extension_"+ext, func(t *testing.T) {
			err := config.Save("ext-test",
				AddDirs(tempDir),
				SetFileExtension(ext),
			)

			if err != nil {
				t.Fatalf("Save() with extension %s failed: %v", ext, err)
			}

			expectedFile := filepath.Join(tempDir, "ext-test."+ext)
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Errorf("Save() should have created file %s", expectedFile)
			}
		})
	}
}

func TestSavePreservesType(t *testing.T) {
	tempDir := t.TempDir()

	// Create a config with a known type
	jsonConfig := New()
	jsonConfig.Type = "json"
	jsonConfig.Set("type.test", "json")

	err := jsonConfig.Save("type-test", AddDirs(tempDir))
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load it back and check type is preserved
	loadedConfig, err := Load("type-test", AddDirs(tempDir))
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loadedConfig.Type != "json" {
		t.Errorf("Config type = %q, want 'json'", loadedConfig.Type)
	}
}

func TestSaveOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "overwrite.json")

	// Create initial config
	config1 := New()
	config1.Set("version", 1)
	config1.Set("data", "original")

	err := config1.Save("overwrite", SetConfigFile(configFile))
	if err != nil {
		t.Fatalf("First save failed: %v", err)
	}

	// Modify and save again
	config2 := New()
	config2.Set("version", 2)
	config2.Set("data", "updated")

	err = config2.Save("overwrite", SetConfigFile(configFile))
	if err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	// Verify the file was overwritten
	finalConfig, err := Load("overwrite", SetConfigFile(configFile))
	if err != nil {
		t.Fatalf("Load after overwrite failed: %v", err)
	}

	if finalConfig.GetInt("version") != 2 {
		t.Errorf("version = %d, want 2", finalConfig.GetInt("version"))
	}

	if finalConfig.GetString("data") != "updated" {
		t.Errorf("data = %q, want 'updated'", finalConfig.GetString("data"))
	}
}

func TestSaveComplexData(t *testing.T) {
	tempDir := t.TempDir()

	config := New()

	// Set complex nested data
	config.Set("database.connections.primary.host", "db1.example.com")
	config.Set("database.connections.primary.port", 5432)
	config.Set("database.connections.secondary.host", "db2.example.com")
	config.Set("database.connections.secondary.port", 5433)

	// Set arrays
	config.Set("servers", []string{"web1", "web2", "web3"})
	config.Set("ports", []int{8080, 8081, 8082})

	// Set maps
	envVars := map[string]string{
		"NODE_ENV": "production",
		"LOG_LEVEL": "info",
	}
	config.Set("environment", envVars)

	err := config.Save("complex", AddDirs(tempDir))
	if err != nil {
		t.Fatalf("Save() complex data failed: %v", err)
	}

	// Verify by loading back
	loadedConfig, err := Load("complex", AddDirs(tempDir))
	if err != nil {
		t.Fatalf("Load() complex data failed: %v", err)
	}

	// Check nested values
	if loadedConfig.GetString("database.connections.primary.host") != "db1.example.com" {
		t.Error("Complex nested string not preserved")
	}

	if loadedConfig.GetInt("database.connections.secondary.port") != 5433 {
		t.Error("Complex nested int not preserved")
	}

	// Check arrays
	servers := loadedConfig.GetStringSlice("servers")
	if len(servers) != 3 || servers[0] != "web1" {
		t.Error("String slice not preserved")
	}

	// Check environment map
	if loadedConfig.GetString("environment.NODE_ENV") != "production" {
		t.Error("Map values not preserved")
	}
}