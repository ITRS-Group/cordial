/*
Copyright © 2022 ITRS Group

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
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test config files
	jsonConfig := `{
		"database": {
			"host": "localhost",
			"port": 5432
		},
		"app": {
			"name": "testapp",
			"debug": true
		}
	}`

	yamlConfig := `
database:
  host: remotehost
  port: 3306
app:
  name: yamlapp
  timeout: 30
`

	jsonFile := filepath.Join(tempDir, "test.json")
	yamlFile := filepath.Join(tempDir, "test.yaml")

	err := os.WriteFile(jsonFile, []byte(jsonConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}

	err = os.WriteFile(yamlFile, []byte(yamlConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	tests := []struct {
		name        string
		configName  string
		options     []FileOptions
		wantErr     bool
		wantHost    string
		wantPort    int
		wantAppName string
	}{
		{
			name:        "load JSON config",
			configName:  "test",
			options:     []FileOptions{SetConfigFile(jsonFile)},
			wantErr:     false,
			wantHost:    "localhost",
			wantPort:    5432,
			wantAppName: "testapp",
		},
		{
			name:        "load YAML config",
			configName:  "test",
			options:     []FileOptions{SetConfigFile(yamlFile)},
			wantErr:     false,
			wantHost:    "remotehost",
			wantPort:    3306,
			wantAppName: "yamlapp",
		},
		{
			name:       "load from directory",
			configName: "test",
			options:    []FileOptions{AddDirs(tempDir)},
			wantErr:    false,
			// Should find the first config file (order depends on filesystem)
		},
		{
			name:       "non-existent config",
			configName: "nonexistent",
			options:    []FileOptions{AddDirs(tempDir)},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := Load(tt.configName, tt.options...)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Load() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			
			if config == nil {
				t.Fatal("Load() returned nil config")
			}

			// Test loaded values (only for specific test cases)
			if tt.wantHost != "" {
				if got := config.GetString("database.host"); got != tt.wantHost {
					t.Errorf("database.host = %q, want %q", got, tt.wantHost)
				}
			}
			
			if tt.wantPort != 0 {
				if got := config.GetInt("database.port"); got != tt.wantPort {
					t.Errorf("database.port = %d, want %d", got, tt.wantPort)
				}
			}
			
			if tt.wantAppName != "" {
				if got := config.GetString("app.name"); got != tt.wantAppName {
					t.Errorf("app.name = %q, want %q", got, tt.wantAppName)
				}
			}
		})
	}
}

func TestLoadWithDefaults(t *testing.T) {
	defaults := `{
		"database": {
			"host": "defaulthost",
			"port": 5432,
			"timeout": 30
		},
		"app": {
			"name": "defaultapp"
		}
	}`

	// Create a temporary config file with some values
	tempDir := t.TempDir()
	configContent := `{
		"database": {
			"host": "confighost"
		},
		"app": {
			"debug": true
		}
	}`
	
	configFile := filepath.Join(tempDir, "test.json")
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := Load("test",
		WithDefaults([]byte(defaults), "json"),
		SetConfigFile(configFile),
	)
	
	if err != nil {
		t.Fatalf("Load() with defaults failed: %v", err)
	}

	// Should have values from both defaults and config file
	if got := config.GetString("database.host"); got != "confighost" {
		t.Errorf("database.host = %q, want 'confighost' (from config file)", got)
	}
	
	if got := config.GetInt("database.port"); got != 5432 {
		t.Errorf("database.port = %d, want 5432 (from defaults)", got)
	}
	
	if got := config.GetInt("database.timeout"); got != 30 {
		t.Errorf("database.timeout = %d, want 30 (from defaults)", got)
	}
	
	if got := config.GetString("app.name"); got != "defaultapp" {
		t.Errorf("app.name = %q, want 'defaultapp' (from defaults)", got)
	}
	
	if got := config.GetBool("app.debug"); got != true {
		t.Errorf("app.debug = %t, want true (from config file)", got)
	}
}

func TestLoadMergeSettings(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple config files
	config1 := `{"key1": "value1", "shared": "from_config1"}`
	config2 := `{"key2": "value2", "shared": "from_config2"}`

	file1 := filepath.Join(tempDir, "config1.json")
	file2 := filepath.Join(tempDir, "config2.json")

	err := os.WriteFile(file1, []byte(config1), 0644)
	if err != nil {
		t.Fatalf("Failed to create config1: %v", err)
	}

	err = os.WriteFile(file2, []byte(config2), 0644)
	if err != nil {
		t.Fatalf("Failed to create config2: %v", err)
	}

	// Test without merge (should only load first file found)
	config, err := Load("config1",
		AddDirs(tempDir),
	)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if config.GetString("key2") == "value2" {
		t.Error("Without MergeSettings, should not load second config")
	}

	// Test with merge - note: actual merging behavior depends on implementation
	// This is more of a smoke test to ensure the option doesn't break loading
	config, err = Load("config1",
		AddDirs(tempDir),
		MergeSettings(),
	)
	if err != nil {
		t.Fatalf("Load() with MergeSettings failed: %v", err)
	}

	if config == nil {
		t.Fatal("Load() with MergeSettings returned nil")
	}
}

func TestLoadSetGlobal(t *testing.T) {
	tempDir := t.TempDir()
	
	configContent := `{"test": {"global": true}}`
	configFile := filepath.Join(tempDir, "global.json")
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Load with SetGlobal option
	originalGlobal := GetConfig()
	
	config, err := Load("global",
		SetConfigFile(configFile),
		UseGlobal(),
	)
	
	if err != nil {
		t.Fatalf("Load() with UseGlobal failed: %v", err)
	}

	// The returned config should be the same as the global one
	currentGlobal := GetConfig()
	if config != currentGlobal {
		t.Error("Load() with UseGlobal should return the global config")
	}

	// Global config should have the loaded values
	if !GetConfig().GetBool("test.global") {
		t.Error("Global config should have loaded values")
	}

	// Reset global config for other tests
	global = originalGlobal
}

func TestPath(t *testing.T) {
	tempDir := t.TempDir()
	
	configFile := filepath.Join(tempDir, "pathtest.yaml")
	err := os.WriteFile(configFile, []byte("test: value"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	tests := []struct {
		name     string
		options  []FileOptions
		contains string
	}{
		{
			name:     "specific file path",
			options:  []FileOptions{SetConfigFile(configFile)},
			contains: "pathtest.yaml",
		},
		{
			name:     "directory search",
			options:  []FileOptions{AddDirs(tempDir)},
			contains: tempDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := Path("pathtest", tt.options...)
			
			if path == "" {
				t.Error("Path() returned empty string")
			}
			
			if !strings.Contains(path, tt.contains) {
				t.Errorf("Path() = %q, should contain %q", path, tt.contains)
			}
		})
	}
}

func TestLoadWithReader(t *testing.T) {
	configContent := `{
		"reader": {
			"test": true,
			"value": 42
		}
	}`

	config, err := Load("reader",
		SetConfigReader(strings.NewReader(configContent)),
		SetFileExtension("json"),
	)

	if err != nil {
		t.Fatalf("Load() with reader failed: %v", err)
	}

	if !config.GetBool("reader.test") {
		t.Error("Config from reader should have reader.test = true")
	}

	if config.GetInt("reader.value") != 42 {
		t.Errorf("reader.value = %d, want 42", config.GetInt("reader.value"))
	}
}

func TestLoadMustExist(t *testing.T) {
	// Test that MustExist option causes error when file doesn't exist
	_, err := Load("nonexistent",
		AddDirs("/nonexistent/path"),
		MustExist(),
	)

	if err == nil {
		t.Error("Load() with MustExist should fail when config doesn't exist")
	}
}

func TestLoadIgnoreOptions(t *testing.T) {
	tempDir := t.TempDir()
	
	configContent := `{"ignore": {"test": true}}`
	configFile := filepath.Join(tempDir, "ignore.json")
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test various ignore options don't break loading
	config, err := Load("ignore",
		SetConfigFile(configFile),
		IgnoreWorkingDir(),
		IgnoreUserConfDir(),
		IgnoreSystemDir(),
	)

	if err != nil {
		t.Fatalf("Load() with ignore options failed: %v", err)
	}

	if !config.GetBool("ignore.test") {
		t.Error("Config should have loaded despite ignore options")
	}
}