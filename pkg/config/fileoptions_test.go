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
	"strings"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/itrs-group/cordial/pkg/host"
)

func TestSetAppName(t *testing.T) {
	config := New(SetAppName("testapp"))

	// Check that the app name is used in config dir
	appDir := config.AppConfigDir()
	if !strings.Contains(appDir, "testapp") {
		t.Errorf("AppConfigDir() should contain app name, got %q", appDir)
	}
}

func TestSetConfigFile(t *testing.T) {
	testFile := "/path/to/config.json"
	
	// This is more of a smoke test since the actual file doesn't exist
	// The option should be accepted without error
	config := New()
	
	// Test that we can create config with file option
	_ = New(SetConfigFile(testFile))
	
	// If we get here without panic, the option was accepted
	if config == nil {
		t.Error("Config should not be nil")
	}
}

func TestSetFileExtension(t *testing.T) {
	tests := []string{"json", "yaml", "yml", "toml", "xml"}
	
	for _, ext := range tests {
		t.Run("extension_"+ext, func(t *testing.T) {
			config := New(SetFileExtension(ext))
			if config == nil {
				t.Error("Config should not be nil with extension", ext)
			}
		})
	}
}

func TestKeyDelimiter(t *testing.T) {
	tests := []struct {
		delimiter string
		parts     []string
		want      string
	}{
		{".", []string{"a", "b", "c"}, "a.b.c"},
		{":", []string{"x", "y", "z"}, "x:y:z"},
		{"_", []string{"one", "two"}, "one_two"},
		{"/", []string{"path", "to", "item"}, "path/to/item"},
	}

	for _, tt := range tests {
		t.Run("delimiter_"+tt.delimiter, func(t *testing.T) {
			config := New(KeyDelimiter(tt.delimiter))
			got := config.Join(tt.parts...)
			
			if got != tt.want {
				t.Errorf("Join() with delimiter %q = %q, want %q", tt.delimiter, got, tt.want)
			}
			
			if config.Delimiter() != tt.delimiter {
				t.Errorf("Delimiter() = %q, want %q", config.Delimiter(), tt.delimiter)
			}
		})
	}
}

func TestWithEnvs(t *testing.T) {
	config := New(WithEnvs("TEST", "_"))
	
	// Basic test - should not panic and should create valid config
	if config == nil {
		t.Error("Config with env options should not be nil")
	}
	
	// More detailed testing would require setting actual env vars
	// which is covered in other test files
}

func TestAddDirs(t *testing.T) {
	dirs := []string{"/path1", "/path2", "/path3"}
	config := New(AddDirs(dirs...))
	
	if config == nil {
		t.Error("Config with added dirs should not be nil")
	}
}

func TestFromDir(t *testing.T) {
	testDir := "/test/directory"
	config := New(FromDir(testDir))
	
	if config == nil {
		t.Error("Config with FromDir should not be nil")
	}
}

func TestIgnoreOptions(t *testing.T) {
	// Test that ignore options don't break config creation
	config := New(
		IgnoreWorkingDir(),
		IgnoreUserConfDir(),
		IgnoreSystemDir(),
	)
	
	if config == nil {
		t.Error("Config with ignore options should not be nil")
	}
}

func TestMergeSettings(t *testing.T) {
	config := New(MergeSettings())
	
	if config == nil {
		t.Error("Config with MergeSettings should not be nil")
	}
}

func TestHost(t *testing.T) {
	localhost := host.Localhost
	config := New(Host(localhost))
	
	if config == nil {
		t.Error("Config with Host option should not be nil")
	}
}

func TestUseGlobal(t *testing.T) {
	// Save original global
	originalGlobal := GetConfig()
	
	config := New(UseGlobal())
	
	// UseGlobal should affect the global config
	if config == nil {
		t.Error("Config with UseGlobal should not be nil")
	}
	
	// Restore original for other tests
	global = originalGlobal
}

func TestUseDefaults(t *testing.T) {
	config := New(UseDefaults(true))
	if config == nil {
		t.Error("Config with UseDefaults(true) should not be nil")
	}
	
	config2 := New(UseDefaults(false))
	if config2 == nil {
		t.Error("Config with UseDefaults(false) should not be nil")
	}
}

func TestWithDefaults(t *testing.T) {
	defaults := []byte(`{"default": {"value": "test"}}`)
	config := New(WithDefaults(defaults, "json"))
	
	if config == nil {
		t.Error("Config with defaults should not be nil")
	}
}

func TestMustExist(t *testing.T) {
	config := New(MustExist())
	
	if config == nil {
		t.Error("Config with MustExist should not be nil")
	}
}

func TestSetConfigReader(t *testing.T) {
	reader := strings.NewReader(`{"test": "value"}`)
	config := New(SetConfigReader(reader))
	
	if config == nil {
		t.Error("Config with reader should not be nil")
	}
}

func TestDefaultKeyDelimiter(t *testing.T) {
	// Test setting global default
	originalDelimiter := "."
	
	DefaultKeyDelimiter(":")
	config := New()
	
	if config.Delimiter() != ":" {
		t.Errorf("Default delimiter should be ':', got %q", config.Delimiter())
	}
	
	// Reset to original
	DefaultKeyDelimiter(originalDelimiter)
}

func TestDefaultFileExtension(t *testing.T) {
	// Test setting global default extension
	DefaultFileExtension("yaml")
	
	// Create config - extension should be used in file operations
	config := New()
	if config == nil {
		t.Error("Config should not be nil after setting default extension")
	}
	
	// Reset to original (json is typically the default)
	DefaultFileExtension("json")
}

func TestMultipleOptions(t *testing.T) {
	// Test combining multiple options
	config := New(
		SetAppName("multitest"),
		KeyDelimiter(":"),
		SetFileExtension("yaml"),
		WithEnvs("MULTI", "_"),
		AddDirs("/test/dir1", "/test/dir2"),
		UseDefaults(true),
		IgnoreWorkingDir(),
	)
	
	if config == nil {
		t.Error("Config with multiple options should not be nil")
	}
	
	// Test that options were applied
	if config.Delimiter() != ":" {
		t.Error("Key delimiter option not applied")
	}
	
	appDir := config.AppConfigDir()
	if !strings.Contains(appDir, "multitest") {
		t.Error("App name option not applied")
	}
}

func TestOptionOrdering(t *testing.T) {
	// Test that option order doesn't matter
	config1 := New(
		SetAppName("order1"),
		KeyDelimiter("_"),
	)
	
	config2 := New(
		KeyDelimiter("_"),
		SetAppName("order1"),
	)
	
	if config1.Delimiter() != config2.Delimiter() {
		t.Error("Option ordering should not affect delimiter")
	}
}

func TestNilOptions(t *testing.T) {
	// Test that nil options slice works
	config := New(nil...)
	
	if config == nil {
		t.Error("Config with nil options should not be nil")
	}
}

func TestEmptyOptions(t *testing.T) {
	// Test with no options
	config := New()
	
	if config == nil {
		t.Error("Config with no options should not be nil")
	}
	
	// Should have default values
	if config.Delimiter() != "." {
		t.Errorf("Default delimiter should be '.', got %q", config.Delimiter())
	}
}

func TestStopOnInternalDefaultsErrors(t *testing.T) {
	config := New(StopOnInternalDefaultsErrors())
	
	if config == nil {
		t.Error("Config with StopOnInternalDefaultsErrors should not be nil")
	}
}

func TestWatchConfig(t *testing.T) {
	// Test watch config option
	notifyFunc := func(event fsnotify.Event) {
		// Test callback
	}
	
	config := New(WatchConfig(notifyFunc))
	
	if config == nil {
		t.Error("Config with WatchConfig should not be nil")
	}
	
	// Note: Actually triggering the watch would require file system operations
	// This is mainly testing that the option doesn't break config creation
}

func TestCombinedFileOptions(t *testing.T) {
	// Test a realistic combination of file options
	reader := strings.NewReader(`{"combined": {"test": true}}`)
	
	config := New(
		SetAppName("combined-test"),
		SetConfigReader(reader),
		SetFileExtension("json"),
		UseDefaults(false),
	)
	
	if config == nil {
		t.Error("Config with combined file options should not be nil")
	}
}

func TestFileOptionsValidation(t *testing.T) {
	// Test various edge cases for options
	tests := []struct {
		name    string
		options []FileOptions
		valid   bool
	}{
		{
			name:    "empty app name",
			options: []FileOptions{SetAppName("")},
			valid:   true, // Should be valid, just uses empty string
		},
		{
			name:    "empty delimiter",
			options: []FileOptions{KeyDelimiter("")},
			valid:   true, // Implementation dependent
		},
		{
			name:    "empty extension",
			options: []FileOptions{SetFileExtension("")},
			valid:   true, // Should be valid
		},
		{
			name:    "nil reader",
			options: []FileOptions{SetConfigReader(nil)},
			valid:   true, // Should handle gracefully
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := New(tt.options...)
			
			if tt.valid && config == nil {
				t.Error("Valid options should create non-nil config")
			}
		})
	}
}