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

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		options []FileOptions
		want    *Config
	}{
		{
			name:    "default config",
			options: nil,
		},
		{
			name:    "with custom delimiter",
			options: []FileOptions{KeyDelimiter(":")},
		},
		{
			name:    "with app name",
			options: []FileOptions{SetAppName("testapp")},
		},
		{
			name:    "with env prefix",
			options: []FileOptions{WithEnvs("TEST", "_")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.options...)
			if got == nil {
				t.Fatal("New() returned nil")
			}
			if got.Viper == nil {
				t.Error("New() returned Config with nil Viper")
			}
			if got.mutex == nil {
				t.Error("New() returned Config with nil mutex")
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	// Reset global config for testing
	ResetConfig()

	got := GetConfig()
	if got == nil {
		t.Fatal("GetConfig() returned nil")
	}
	if got.Viper == nil {
		t.Error("GetConfig() returned Config with nil Viper")
	}

	// Test that it returns the same instance
	got2 := GetConfig()
	if got != got2 {
		t.Error("GetConfig() should return the same instance")
	}
}

func TestResetConfig(t *testing.T) {
	// Set some values in global config
	global.Set("test.key", "test.value")
	original := GetConfig()

	// Reset config
	ResetConfig()

	// Should be a different instance but preserve settings
	newConfig := GetConfig()
	if original == newConfig {
		t.Error("ResetConfig() should create a new instance")
	}

	if newConfig.GetString("test.key") != "test.value" {
		t.Error("ResetConfig() should preserve existing settings")
	}
}

func TestAppConfigDir(t *testing.T) {
	config := New(SetAppName("testapp"))
	dir := config.AppConfigDir()

	if dir == "" {
		t.Error("AppConfigDir() returned empty string")
	}

	if !strings.Contains(dir, "testapp") {
		t.Errorf("AppConfigDir() = %q, should contain 'testapp'", dir)
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		name      string
		delimiter string
		parts     []string
		want      string
	}{
		{
			name:      "default delimiter",
			delimiter: ".",
			parts:     []string{"a", "b", "c"},
			want:      "a.b.c",
		},
		{
			name:      "custom delimiter",
			delimiter: ":",
			parts:     []string{"x", "y", "z"},
			want:      "x:y:z",
		},
		{
			name:      "single part",
			delimiter: ".",
			parts:     []string{"single"},
			want:      "single",
		},
		{
			name:      "empty parts",
			delimiter: ".",
			parts:     []string{},
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := New(KeyDelimiter(tt.delimiter))
			got := config.Join(tt.parts...)
			if got != tt.want {
				t.Errorf("Join() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		delimiter string
	}{
		{"default", "."},
		{"colon", ":"},
		{"underscore", "_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config *Config
			if tt.delimiter == "." {
				config = New()
			} else {
				config = New(KeyDelimiter(tt.delimiter))
			}

			got := config.Delimiter()
			if got != tt.delimiter {
				t.Errorf("Delimiter() = %q, want %q", got, tt.delimiter)
			}
		})
	}
}

func TestSub(t *testing.T) {
	config := New()
	config.Set("database.host", "localhost")
	config.Set("database.port", 5432)
	config.Set("database.ssl.enabled", true)

	sub := config.Sub("database")
	if sub == nil {
		t.Fatal("Sub() returned nil")
	}

	if sub.GetString("host") != "localhost" {
		t.Errorf("Sub config host = %q, want 'localhost'", sub.GetString("host"))
	}

	if sub.GetInt("port") != 5432 {
		t.Errorf("Sub config port = %d, want 5432", sub.GetInt("port"))
	}

	// Test sub of sub
	sslSub := sub.Sub("ssl")
	if !sslSub.GetBool("enabled") {
		t.Error("Sub of sub should return correct value")
	}

	// Test non-existent key returns empty config, not nil
	emptySub := config.Sub("nonexistent")
	if emptySub == nil {
		t.Error("Sub() with non-existent key should return empty config, not nil")
	}
}

func TestConfigSetGet(t *testing.T) {
	config := New()

	// Test string
	config.Set("string.key", "test value")
	if got := config.GetString("string.key"); got != "test value" {
		t.Errorf("GetString() = %q, want 'test value'", got)
	}

	// Test int
	config.Set("int.key", 42)
	if got := config.GetInt("int.key"); got != 42 {
		t.Errorf("GetInt() = %d, want 42", got)
	}

	// Test bool
	config.Set("bool.key", true)
	if got := config.GetBool("bool.key"); got != true {
		t.Errorf("GetBool() = %t, want true", got)
	}

	// Test slice
	testSlice := []string{"a", "b", "c"}
	config.Set("slice.key", testSlice)
	if got := config.GetStringSlice("slice.key"); len(got) != 3 || got[0] != "a" {
		t.Errorf("GetStringSlice() = %v, want %v", got, testSlice)
	}
}

func TestConfigMerge(t *testing.T) {
	config1 := New()
	config1.Set("key1", "value1")
	config1.Set("shared", "from_config1")

	config2 := New()
	config2.Set("key2", "value2")
	config2.Set("shared", "from_config2")

	// Test merge
	config1.MergeConfigMap(config2.AllSettings())

	if config1.GetString("key1") != "value1" {
		t.Error("Merge should preserve original values")
	}

	if config1.GetString("key2") != "value2" {
		t.Error("Merge should add new values")
	}

	if config1.GetString("shared") != "from_config2" {
		t.Error("Merge should override with new values")
	}
}

func TestUserConfigDir(t *testing.T) {
	// Test with current user
	dir, err := UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() failed: %v", err)
	}

	if dir == "" {
		t.Error("UserConfigDir() returned empty string")
	}

	// Test that the directory is absolute
	if !filepath.IsAbs(dir) {
		t.Errorf("UserConfigDir() = %q, should be absolute path", dir)
	}
}

func TestConfigWithEnv(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_CONFIG_VAR", "env_value")
	defer os.Unsetenv("TEST_CONFIG_VAR")

	config := New(WithEnvs("TEST", "_"))

	// Should be able to get env var through config
	if got := config.GetString("CONFIG.VAR"); got != "env_value" {
		t.Errorf("GetString() from env = %q, want 'env_value'", got)
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Test that global functions work
	ResetConfig()

	// Test Join
	joined := Join("a", "b", "c")
	expected := "a.b.c"
	if joined != expected {
		t.Errorf("Join() = %q, want %q", joined, expected)
	}

	// Test Delimiter
	delimiter := Delimiter()
	if delimiter != "." {
		t.Errorf("Delimiter() = %q, want '.'", delimiter)
	}

	// Test AppConfigDir
	dir := AppConfigDir()
	// Should not be empty unless there's an error
	if dir == "" {
		t.Log("AppConfigDir() returned empty string - might be expected in test environment")
	}
}
