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

	"github.com/itrs-group/cordial/pkg/host"
)

func TestPromoteFile(t *testing.T) {
	tempDir := t.TempDir()
	localhost := host.Localhost

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	file3 := filepath.Join(tempDir, "file3.txt")

	content1 := "content1"
	content2 := "content2"

	err := os.WriteFile(file2, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file2: %v", err)
	}

	err = os.WriteFile(file3, []byte("content3"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file3: %v", err)
	}

	tests := []struct {
		name       string
		paths      []string
		want       string
		wantExists bool
		setup      func()
	}{
		{
			name:       "promote second file to first",
			paths:      []string{file1, file2, file3},
			want:       file1,
			wantExists: true,
			setup:      func() {},
		},
		{
			name:       "first file already exists",
			paths:      []string{file1, file2, file3},
			want:       file1,
			wantExists: true,
			setup: func() {
				os.WriteFile(file1, []byte(content1), 0644)
			},
		},
		{
			name:       "empty first path",
			paths:      []string{"", file2, file3},
			want:       file2,
			wantExists: true,
			setup:      func() {},
		},
		{
			name:       "no files exist",
			paths:      []string{filepath.Join(tempDir, "none1"), filepath.Join(tempDir, "none2")},
			want:       "",
			wantExists: false,
			setup:      func() {},
		},
		{
			name:       "empty paths",
			paths:      []string{},
			want:       "",
			wantExists: false,
			setup:      func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up from previous test
			os.Remove(file1)

			tt.setup()

			got := PromoteFile(localhost, tt.paths...)

			if got != tt.want {
				t.Errorf("PromoteFile() = %q, want %q", got, tt.want)
			}

			if tt.wantExists {
				if got == "" {
					return // Skip existence check if no file expected
				}
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("PromoteFile() result file %q should exist", got)
				}
			}
		})
	}
}

func TestPromoteFileWithDirectory(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	localhost := host.Localhost

	// Create test file
	srcFile := filepath.Join(tempDir, "source.txt")
	err = os.WriteFile(srcFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Test promoting to directory
	result := PromoteFile(localhost, subDir, srcFile)

	// Should move file into the directory
	if !strings.Contains(result, subDir) {
		t.Errorf("PromoteFile() with directory should return path in directory, got %q", result)
	}

	if result != "" {
		if _, err := os.Stat(result); os.IsNotExist(err) {
			t.Errorf("Promoted file should exist at %q", result)
		}
	}
}

func TestAbbreviateHome(t *testing.T) {
	// Get the current user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "path in home",
			path: filepath.Join(homeDir, "test", "file.txt"),
			want: "~/test/file.txt",
		},
		{
			name: "home directory itself",
			path: homeDir,
			want: "~",
		},
		{
			name: "path not in home",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "relative path",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AbbreviateHome(tt.path)
			if got != tt.want {
				t.Errorf("AbbreviateHome() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "tilde path",
			path: "~/test/file.txt",
			want: filepath.Join(homeDir, "test", "file.txt"),
		},
		{
			name: "tilde with separator",
			path: "~/",
			want: homeDir,
		},
		{
			name: "absolute path",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "relative path",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "tilde in middle (should not expand)",
			path: "/path/~/file",
			want: "/path/~/file",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandHome(tt.path)
			if got != tt.want {
				t.Errorf("ExpandHome() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandHomeBytes(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	tests := []struct {
		name string
		path []byte
		want []byte
	}{
		{
			name: "tilde path",
			path: []byte("~/test/file.txt"),
			want: []byte(filepath.Join(homeDir, "test", "file.txt")),
		},
		{
			name: "tilde",
			path: []byte("~/"),
			want: []byte(homeDir),
		},
		{
			name: "absolute path",
			path: []byte("/usr/local/bin"),
			want: []byte("/usr/local/bin"),
		},
		{
			name: "empty path",
			path: []byte(""),
			want: []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandHomeBytes(tt.path)
			if string(got) != string(tt.want) {
				t.Errorf("ExpandHomeBytes() = %q, want %q", string(got), string(tt.want))
			}
		})
	}
}

func TestHomeExpansionRoundTrip(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	// Test round trip: expand then abbreviate
	originalPath := "~/documents/config.json"
	expandedPath := ExpandHome(originalPath)
	abbreviatedPath := AbbreviateHome(expandedPath)

	if abbreviatedPath != originalPath {
		t.Errorf("Round trip failed: %q -> %q -> %q", originalPath, expandedPath, abbreviatedPath)
	}

	// Test the other direction: abbreviate then expand
	fullPath := filepath.Join(homeDir, "test", "file.txt")
	abbreviated := AbbreviateHome(fullPath)
	expanded := ExpandHome(abbreviated)

	if expanded != fullPath {
		t.Errorf("Reverse round trip failed: %q -> %q -> %q", fullPath, abbreviated, expanded)
	}
}

func TestPromoteFileErrors(t *testing.T) {
	// Create a host that might fail operations
	localhost := host.Localhost

	// Test with non-existent directory (should handle gracefully)
	result := PromoteFile(localhost, "/nonexistent/path/file1", "/nonexistent/path/file2")
	if result != "" {
		t.Errorf("PromoteFile() with non-existent files should return empty string, got %q", result)
	}
}

func TestFilePathEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		function string
		input    string
		expected func(string) bool // function to validate the result
	}{
		{
			name:     "AbbreviateHome with trailing slash",
			function: "AbbreviateHome",
			input:    "/some/path/",
			expected: func(result string) bool { return result == "/some/path" },
		},
		{
			name:     "ExpandHome with multiple tildes",
			function: "ExpandHome",
			input:    "~/~/test",
			expected: func(result string) bool {
				homeDir, _ := os.UserHomeDir()
				return strings.HasPrefix(result, homeDir)
			},
		},
		{
			name:     "ExpandHome with tilde not at start",
			function: "ExpandHome",
			input:    "prefix~/test",
			expected: func(result string) bool { return result == "prefix~/test" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			switch tt.function {
			case "AbbreviateHome":
				result = AbbreviateHome(tt.input)
			case "ExpandHome":
				result = ExpandHome(tt.input)
			}

			if !tt.expected(result) {
				t.Errorf("%s(%q) = %q, validation failed", tt.function, tt.input, result)
			}
		})
	}
}

func TestPromoteFileWithPermissions(t *testing.T) {
	tempDir := t.TempDir()
	localhost := host.Localhost

	// Create a test file with specific permissions
	testFile := filepath.Join(tempDir, "perm_test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	targetFile := filepath.Join(tempDir, "target.txt")

	// Promote the file
	result := PromoteFile(localhost, targetFile, testFile)

	if result != targetFile {
		t.Errorf("PromoteFile() = %q, want %q", result, targetFile)
	}

	// Verify the file exists at the target location
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Promoted file should exist at target location")
	}
}

func TestExpandHomeBytesEdgeCases(t *testing.T) {
	// Test with nil input
	result := ExpandHomeBytes(nil)
	if result != nil {
		t.Errorf("ExpandHomeBytes(nil) = %v, want nil", result)
	}

	// Test with binary data that contains tilde-like sequences
	binaryData := []byte{0x7e, 0x2f, 0x00, 0x01, 0x02} // starts with ~/
	result = ExpandHomeBytes(binaryData)
	// Should handle binary data gracefully (behavior may vary)
	if result == nil {
		t.Error("ExpandHomeBytes() should not return nil for valid input")
	}
}
