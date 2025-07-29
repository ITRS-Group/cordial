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
	"testing"
)

func TestExpandString(t *testing.T) {
	config := New()
	config.Set("database.host", "localhost")
	config.Set("database.port", "5432")
	config.Set("app.name", "testapp")

	tests := []struct {
		name     string
		input    string
		want     string
		options  []ExpandOptions
	}{
		{
			name:  "no expansion",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "config expansion",
			input: "Host: ${database.host}",
			want:  "Host: localhost",
		},
		{
			name:  "config prefix expansion",
			input: "Port: ${config:database.port}",
			want:  "Port: 5432",
		},
		{
			name:  "multiple expansions",
			input: "${app.name} runs on ${database.host}:${database.port}",
			want:  "testapp runs on localhost:5432",
		},
		{
			name:  "non-existent config key",
			input: "Missing: ${nonexistent.key}",
			want:  "Missing: ",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "nested braces",
			input: "Value: ${{nested}}",
			want:  "Value: ${{nested}}", // Should not expand invalid syntax
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ExpandString(tt.input, tt.options...)
			if got != tt.want {
				t.Errorf("ExpandString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandStringEnv(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("TEST_NUMBER", "42")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("TEST_NUMBER")

	config := New()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "env expansion",
			input: "Value: ${env:TEST_VAR}",
			want:  "Value: test_value",
		},
		{
			name:  "env number",
			input: "Number: ${env:TEST_NUMBER}",
			want:  "Number: 42",
		},
		{
			name:  "non-existent env var",
			input: "Missing: ${env:NONEXISTENT}",
			want:  "Missing: ",
		},
		{
			name:  "mixed env and config",
			input: "Env: ${env:TEST_VAR}, Config: ${database.host}",
			want:  "Env: test_value, Config: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ExpandString(tt.input)
			if got != tt.want {
				t.Errorf("ExpandString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandStringWithLookupTable(t *testing.T) {
	config := New()

	lookupTable := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"override": "from_table",
	}

	// Also set a config value with the same name
	config.Set("override", "from_config")

	tests := []struct {
		name    string
		input   string
		want    string
		options []ExpandOptions
	}{
		{
			name:    "lookup table expansion",
			input:   "Value: ${key1}",
			want:    "Value: value1",
			options: []ExpandOptions{LookupTable(lookupTable)},
		},
		{
			name:    "lookup table override",
			input:   "Override: ${override}",
			want:    "Override: from_table",
			options: []ExpandOptions{LookupTable(lookupTable)},
		},
		{
			name:  "no lookup table",
			input: "Value: ${key1}",
			want:  "Value: ", // Should fallback to env (which doesn't exist)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ExpandString(tt.input, tt.options...)
			if got != tt.want {
				t.Errorf("ExpandString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandStringFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	fileContent := "file content here"
	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := New()

	tests := []struct {
		name    string
		input   string
		want    string
		options []ExpandOptions
	}{
		{
			name:  "file expansion",
			input: "Content: ${" + testFile + "}",
			want:  "Content: " + fileContent,
		},
		{
			name:  "file:// prefix",
			input: "Content: ${file://" + testFile + "}",
			want:  "Content: " + fileContent,
		},
		{
			name:  "non-existent file",
			input: "Content: ${/nonexistent/file.txt}",
			want:  "Content: ", // Should return empty on error
		},
		{
			name:    "disabled external lookups",
			input:   "Content: ${" + testFile + "}",
			want:    "Content: ", // Should not read file
			options: []ExpandOptions{ExternalLookups(false)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ExpandString(tt.input, tt.options...)
			if got != tt.want {
				t.Errorf("ExpandString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandStringWithDefault(t *testing.T) {
	config := New()

	tests := []struct {
		name    string
		input   string
		want    string
		options []ExpandOptions
	}{
		{
			name:    "empty with default",
			input:   "",
			want:    "default_value",
			options: []ExpandOptions{Default("default_value")},
		},
		{
			name:    "non-empty with default",
			input:   "actual_value",
			want:    "actual_value",
			options: []ExpandOptions{Default("default_value")},
		},
		{
			name:    "expansion with default",
			input:   "${nonexistent}",
			want:    "",
			options: []ExpandOptions{Default("default_value")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ExpandString(tt.input, tt.options...)
			if got != tt.want {
				t.Errorf("ExpandString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandStringWithTrimSpace(t *testing.T) {
	config := New()
	config.Set("test.value", "  trimmed  ")

	tests := []struct {
		name    string
		input   string
		want    string
		options []ExpandOptions
	}{
		{
			name:    "trim space enabled",
			input:   "${test.value}",
			want:    "trimmed",
			options: []ExpandOptions{TrimSpace(true)},
		},
		{
			name:  "trim space disabled (default)",
			input: "${test.value}",
			want:  "  trimmed  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ExpandString(tt.input, tt.options...)
			if got != tt.want {
				t.Errorf("ExpandString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandStringGlobal(t *testing.T) {
	// Test global ExpandString function
	ResetConfig()
	GetConfig().Set("global.test", "global_value")

	input := "Global: ${global.test}"
	want := "Global: global_value"

	got := ExpandString(input)
	if got != want {
		t.Errorf("ExpandString() global = %q, want %q", got, want)
	}
}

func TestExpandStringSlice(t *testing.T) {
	config := New()
	config.Set("server.host", "localhost")
	config.Set("server.port", "8080")

	input := []string{
		"http://${server.host}:${server.port}",
		"Database: ${database.url}",
		"Plain text",
	}

	want := []string{
		"http://localhost:8080",
		"Database: ",
		"Plain text",
	}

	got := config.ExpandStringSlice(input)
	if len(got) != len(want) {
		t.Fatalf("ExpandStringSlice() length = %d, want %d", len(got), len(want))
	}

	for i, v := range got {
		if v != want[i] {
			t.Errorf("ExpandStringSlice()[%d] = %q, want %q", i, v, want[i])
		}
	}
}

func TestExpandStringSliceGlobal(t *testing.T) {
	ResetConfig()
	GetConfig().Set("global.value", "test")

	input := []string{"${global.value}", "static"}
	want := []string{"test", "static"}

	got := ExpandStringSlice(input)
	for i, v := range got {
		if v != want[i] {
			t.Errorf("ExpandStringSlice() global[%d] = %q, want %q", i, v, want[i])
		}
	}
}

func TestExpand(t *testing.T) {
	config := New()
	config.Set("test.key", "value")

	input := "Test: ${test.key}"
	want := []byte("Test: value")

	got := config.Expand(input)
	if string(got) != string(want) {
		t.Errorf("Expand() = %q, want %q", string(got), string(want))
	}
}

func TestExpandGlobal(t *testing.T) {
	ResetConfig()
	GetConfig().Set("global.key", "global")

	input := "Global: ${global.key}"
	want := []byte("Global: global")

	got := Expand(input)
	if string(got) != string(want) {
		t.Errorf("Expand() global = %q, want %q", string(got), string(want))
	}
}

func TestExpandWithNoExpand(t *testing.T) {
	config := New()
	config.Set("test.key", "value")

	input := "Test: ${test.key}"
	want := "Test: ${test.key}" // Should not expand

	got := config.ExpandString(input, NoExpand())
	if got != want {
		t.Errorf("ExpandString() with NoExpand = %q, want %q", got, want)
	}
}

func TestExpandComplexScenarios(t *testing.T) {
	config := New()
	config.Set("database.host", "db.example.com")
	config.Set("database.port", 5432)
	config.Set("app.name", "myapp")
	config.Set("template", "Welcome to ${app.name}")

	// Set environment variable
	os.Setenv("STAGE", "production")
	defer os.Unsetenv("STAGE")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "nested reference",
			input: "Message: ${template}",
			want:  "Message: Welcome to myapp",
		},
		{
			name:  "mixed sources",
			input: "${app.name} on ${database.host}:${database.port} (${env:STAGE})",
			want:  "myapp on db.example.com:5432 (production)",
		},
		{
			name:  "URL-like expansion",
			input: "postgresql://${database.host}:${database.port}/mydb",
			want:  "postgresql://db.example.com:5432/mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ExpandString(tt.input)
			if got != tt.want {
				t.Errorf("ExpandString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandToPassword(t *testing.T) {
	config := New()
	config.Set("password", "secret123")

	plaintext := config.ExpandToPassword("${password}")
	if plaintext == nil {
		t.Fatal("ExpandToPassword() returned nil")
	}

	// Note: We can't directly test the content without exposing it
	// This is a basic test to ensure the function doesn't panic
	if plaintext.IsNil() {
		t.Error("ExpandToPassword() returned nil plaintext")
	}
}

func TestExpandToPasswordGlobal(t *testing.T) {
	ResetConfig()
	GetConfig().Set("global.password", "global_secret")

	plaintext := ExpandToPassword("${global.password}")
	if plaintext == nil {
		t.Fatal("ExpandToPassword() global returned nil")
	}

	if plaintext.IsNil() {
		t.Error("ExpandToPassword() global returned nil plaintext")
	}
}

func TestExpandToEnclave(t *testing.T) {
	config := New()
	config.Set("secret", "enclave_data")

	enclave := config.ExpandToEnclave("${secret}")
	if enclave == nil {
		t.Fatal("ExpandToEnclave() returned nil")
	}

	// Basic test to ensure the enclave is valid
	if enclave.Size() == 0 {
		t.Error("ExpandToEnclave() returned empty enclave")
	}
}

func TestExpandToEnclaveGlobal(t *testing.T) {
	ResetConfig()
	GetConfig().Set("global.secret", "global_enclave")

	enclave := ExpandToEnclave("${global.secret}")
	if enclave == nil {
		t.Fatal("ExpandToEnclave() global returned nil")
	}

	if enclave.Size() == 0 {
		t.Error("ExpandToEnclave() global returned empty enclave")
	}
}

func TestExpandToLockedBuffer(t *testing.T) {
	config := New()
	config.Set("buffer", "locked_data")

	buffer := config.ExpandToLockedBuffer("${buffer}")
	if buffer == nil {
		t.Fatal("ExpandToLockedBuffer() returned nil")
	}

	if buffer.Size() == 0 {
		t.Error("ExpandToLockedBuffer() returned empty buffer")
	}
}

func TestExpandToLockedBufferGlobal(t *testing.T) {
	ResetConfig()
	GetConfig().Set("global.buffer", "global_locked")

	buffer := ExpandToLockedBuffer("${global.buffer}")
	if buffer == nil {
		t.Fatal("ExpandToLockedBuffer() global returned nil")
	}

	if buffer.Size() == 0 {
		t.Error("ExpandToLockedBuffer() global returned empty buffer")
	}
}

func TestExpandWithCustomPrefix(t *testing.T) {
	config := New()

	// Custom prefix function that reverses the string
	reverseFunc := func(c *Config, s string, trim bool) (string, error) {
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), nil
	}

	input := "Reversed: ${reverse:hello}"
	want := "Reversed: olleh"

	got := config.ExpandString(input, Prefix("reverse", reverseFunc))
	if got != want {
		t.Errorf("ExpandString() with custom prefix = %q, want %q", got, want)
	}
}

func TestExpandError(t *testing.T) {
	config := New()

	// Test various error conditions that should result in empty strings
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "malformed syntax",
			input: "Bad: ${unclosed",
			want:  "Bad: ${unclosed", // Should not expand malformed syntax
		},
		{
			name:  "empty variable name",
			input: "Empty: ${}",
			want:  "Empty: ", // Should expand to empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ExpandString(tt.input)
			if got != tt.want {
				t.Errorf("ExpandString() error case = %q, want %q", got, tt.want)
			}
		})
	}
}