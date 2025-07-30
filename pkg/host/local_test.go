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

package host

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestNewLocal(t *testing.T) {
	h := NewLocal()
	if h == nil {
		t.Fatal("NewLocal() returned nil")
	}

	local, ok := h.(*Local)
	if !ok {
		t.Fatal("NewLocal() did not return *Local")
	}

	if local == nil {
		t.Error("Local struct should not be nil")
	}
}

func TestLocalhost(t *testing.T) {
	if Localhost == nil {
		t.Fatal("Localhost global variable is nil")
	}

	if !Localhost.IsLocal() {
		t.Error("Localhost should be local")
	}
}

func TestLocal_Username(t *testing.T) {
	h := NewLocal()
	username := h.Username()
	if username == "" {
		t.Error("Username() should return non-empty string")
	}
}

func TestLocal_Hostname(t *testing.T) {
	h := NewLocal()
	hostname := h.Hostname()
	if hostname == "" {
		t.Error("Hostname() should return non-empty string")
	}
}

func TestLocal_IsLocal(t *testing.T) {
	h := NewLocal()
	if !h.IsLocal() {
		t.Error("Local host should return true for IsLocal()")
	}
}

func TestLocal_IsAvailable(t *testing.T) {
	h := NewLocal()
	available, err := h.IsAvailable()
	if err != nil {
		t.Errorf("IsAvailable() returned error: %v", err)
	}
	if !available {
		t.Error("Local host should always be available")
	}
}

func TestLocal_String(t *testing.T) {
	h := NewLocal()
	s := h.String()
	if s != "localhost" {
		t.Errorf("String() returned %q, expected %q", s, "localhost")
	}
}

func TestLocal_ServerVersion(t *testing.T) {
	h := NewLocal()
	version := h.ServerVersion()
	if version != runtime.GOOS {
		t.Errorf("ServerVersion() returned %q, expected %q", version, runtime.GOOS)
	}
}

func TestLocal_LastError(t *testing.T) {
	h := NewLocal()
	err := h.LastError()
	if err != nil {
		t.Errorf("LastError() should return nil for Local, got: %v", err)
	}
}

func TestLocal_TempDir(t *testing.T) {
	h := NewLocal()
	tempDir := h.TempDir()
	if tempDir == "" {
		t.Error("TempDir() should return non-empty string")
	}
	if tempDir != os.TempDir() {
		t.Errorf("TempDir() returned %q, expected %q", tempDir, os.TempDir())
	}
}

func TestLocal_Getwd(t *testing.T) {
	h := NewLocal()
	wd, err := h.Getwd()
	if err != nil {
		t.Errorf("Getwd() failed: %v", err)
	}
	if wd == "" {
		t.Error("Getwd() should return non-empty path")
	}

	// Compare with os.Getwd()
	expectedWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() failed: %v", err)
	}
	if wd != expectedWd {
		t.Errorf("Getwd() returned %q, expected %q", wd, expectedWd)
	}
}

func TestLocal_Abs(t *testing.T) {
	h := NewLocal()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{"Current directory", ".", false},
		{"Relative path", "test", false},
		{"Already absolute", "/tmp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs, err := h.Abs(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Abs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if abs == "" {
					t.Error("Abs() should return non-empty path")
				}
				if !h.IsAbs(abs) {
					t.Errorf("Abs() returned %q which is not absolute", abs)
				}
			}
		})
	}
}

func TestLocal_IsAbs(t *testing.T) {
	h := NewLocal()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Absolute Unix path", "/tmp", true},
		{"Relative path", "test", false},
		{"Current directory", ".", false},
		{"Parent directory", "..", false},
	}

	// Add Windows-specific test if running on Windows
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name     string
			path     string
			expected bool
		}{"Windows absolute path", "C:\\temp", true})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.IsAbs(tt.path)
			if result != tt.expected {
				t.Errorf("IsAbs(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestLocal_FileOperations(t *testing.T) {
	h := NewLocal()

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "local_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("WriteFile and ReadFile", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		testContent := []byte("test content")

		// Write file
		err := h.WriteFile(testFile, testContent, 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		// Read file
		content, err := h.ReadFile(testFile)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}

		if !bytes.Equal(content, testContent) {
			t.Errorf("Content mismatch: got %q, want %q", content, testContent)
		}
	})

	t.Run("Create and Open", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "create_test.txt")
		testContent := []byte("create test content")

		// Create file
		w, err := h.Create(testFile, 0644)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		_, err = w.Write(testContent)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		w.Close()

		// Open file
		r, err := h.Open(testFile)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer r.Close()

		content, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}

		if !bytes.Equal(content, testContent) {
			t.Errorf("Content mismatch: got %q, want %q", content, testContent)
		}
	})

	t.Run("Stat and Lstat", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "stat_test.txt")
		testContent := []byte("stat test")

		// Create test file
		err := h.WriteFile(testFile, testContent, 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		// Test Stat
		fi, err := h.Stat(testFile)
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}

		if fi.Size() != int64(len(testContent)) {
			t.Errorf("File size mismatch: got %d, want %d", fi.Size(), len(testContent))
		}

		// Test Lstat
		fi2, err := h.Lstat(testFile)
		if err != nil {
			t.Fatalf("Lstat failed: %v", err)
		}

		if fi.Size() != fi2.Size() {
			t.Error("Stat and Lstat should return same size for regular file")
		}
	})

	t.Run("MkdirAll", func(t *testing.T) {
		nestedDir := filepath.Join(tempDir, "nested", "dir", "structure")

		err := h.MkdirAll(nestedDir, 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Verify directory exists
		fi, err := h.Stat(nestedDir)
		if err != nil {
			t.Fatalf("Stat failed on created directory: %v", err)
		}

		if !fi.IsDir() {
			t.Error("Created path should be a directory")
		}
	})

	t.Run("ReadDir", func(t *testing.T) {
		// Create some test files and directories
		testDir := filepath.Join(tempDir, "readdir_test")
		err := h.MkdirAll(testDir, 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Create files
		err = h.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		err = h.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("content2"), 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		// Create subdirectory
		err = h.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Read directory
		entries, err := h.ReadDir(testDir)
		if err != nil {
			t.Fatalf("ReadDir failed: %v", err)
		}

		if len(entries) != 3 {
			t.Errorf("Expected 3 entries, got %d", len(entries))
		}

		// Check for expected entries
		names := make(map[string]bool)
		for _, entry := range entries {
			names[entry.Name()] = entry.IsDir()
		}

		expectedEntries := map[string]bool{
			"file1.txt": false,
			"file2.txt": false,
			"subdir":    true,
		}

		for name, isDir := range expectedEntries {
			if actualIsDir, found := names[name]; !found {
				t.Errorf("Expected entry %q not found", name)
			} else if actualIsDir != isDir {
				t.Errorf("Entry %q: expected isDir=%v, got %v", name, isDir, actualIsDir)
			}
		}
	})

	t.Run("Remove and RemoveAll", func(t *testing.T) {
		// Test Remove
		testFile := filepath.Join(tempDir, "remove_test.txt")
		err := h.WriteFile(testFile, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		err = h.Remove(testFile)
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		// Verify file is gone
		_, err = h.Stat(testFile)
		if err == nil {
			t.Error("File should have been removed")
		}

		// Test RemoveAll
		testDir := filepath.Join(tempDir, "removeall_test")
		err = h.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		err = h.WriteFile(filepath.Join(testDir, "file.txt"), []byte("content"), 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		err = h.RemoveAll(testDir)
		if err != nil {
			t.Fatalf("RemoveAll failed: %v", err)
		}

		// Verify directory is gone
		_, err = h.Stat(testDir)
		if err == nil {
			t.Error("Directory should have been removed")
		}
	})

	t.Run("Rename", func(t *testing.T) {
		oldPath := filepath.Join(tempDir, "old.txt")
		newPath := filepath.Join(tempDir, "new.txt")
		testContent := []byte("rename test")

		// Create source file
		err := h.WriteFile(oldPath, testContent, 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		// Rename
		err = h.Rename(oldPath, newPath)
		if err != nil {
			t.Fatalf("Rename failed: %v", err)
		}

		// Verify old file is gone
		_, err = h.Stat(oldPath)
		if err == nil {
			t.Error("Old file should not exist after rename")
		}

		// Verify new file exists with correct content
		content, err := h.ReadFile(newPath)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}

		if !bytes.Equal(content, testContent) {
			t.Errorf("Content mismatch after rename: got %q, want %q", content, testContent)
		}
	})

	t.Run("Glob", func(t *testing.T) {
		globDir := filepath.Join(tempDir, "glob_test")
		err := h.MkdirAll(globDir, 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Create test files
		files := []string{"test1.txt", "test2.txt", "other.log"}
		for _, file := range files {
			err := h.WriteFile(filepath.Join(globDir, file), []byte("content"), 0644)
			if err != nil {
				t.Fatalf("WriteFile failed for %s: %v", file, err)
			}
		}

		// Test glob pattern
		pattern := filepath.Join(globDir, "*.txt")
		matches, err := h.Glob(pattern)
		if err != nil {
			t.Fatalf("Glob failed: %v", err)
		}

		if len(matches) != 2 {
			t.Errorf("Expected 2 matches, got %d", len(matches))
		}

		// Convert to unix-style paths for comparison
		for i, match := range matches {
			matches[i] = filepath.ToSlash(match)
		}

		expectedPrefix := filepath.ToSlash(globDir)
		for _, match := range matches {
			if !strings.HasPrefix(match, expectedPrefix) {
				t.Errorf("Match %q should have prefix %q", match, expectedPrefix)
			}
			if !strings.HasSuffix(match, ".txt") {
				t.Errorf("Match %q should have .txt suffix", match)
			}
		}
	})
}

func TestLocal_Chtimes(t *testing.T) {
	h := NewLocal()

	// Create temporary file
	tempDir, err := os.MkdirTemp("", "local_chtimes_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "chtimes_test.txt")
	err = h.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Set specific times
	atime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)

	err = h.Chtimes(testFile, atime, mtime)
	if err != nil {
		t.Fatalf("Chtimes failed: %v", err)
	}

	// Verify times were set
	fi, err := h.Stat(testFile)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// Note: access time might not be preserved on all filesystems
	if !fi.ModTime().Equal(mtime) {
		t.Errorf("ModTime mismatch: got %v, want %v", fi.ModTime(), mtime)
	}
}

func TestLocal_HostPath(t *testing.T) {
	h := NewLocal()

	tests := []string{
		"/tmp/test",
		"relative/path",
		".",
	}

	for _, path := range tests {
		result := h.HostPath(path)
		if result != path {
			t.Errorf("HostPath(%q) = %q, expected %q", path, result, path)
		}
	}
}

func TestLocal_GetFs(t *testing.T) {
	h := NewLocal()
	fs := h.GetFs()
	if fs == nil {
		t.Error("GetFs() should return non-nil filesystem")
	}

	// Test that the filesystem works
	tempFile := "/tmp/test_getfs.txt"
	testContent := []byte("test content")

	file, err := fs.Create(tempFile)
	if err != nil {
		t.Fatalf("Create via GetFs() failed: %v", err)
	}
	_, err = file.Write(testContent)
	file.Close()
	if err != nil {
		t.Fatalf("Write via GetFs() failed: %v", err)
	}
	defer fs.Remove(tempFile)

	file, err = fs.Open(tempFile)
	if err != nil {
		t.Fatalf("Open via GetFs() failed: %v", err)
	}
	content, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		t.Fatalf("ReadAll via GetFs() failed: %v", err)
	}

	if !bytes.Equal(content, testContent) {
		t.Errorf("Content mismatch via GetFs(): got %q, want %q", content, testContent)
	}
}

func TestLocal_Uname(t *testing.T) {
	h := NewLocal()
	goos, goarch, err := h.Uname()
	if err != nil {
		t.Fatalf("Uname() failed: %v", err)
	}

	if goos != runtime.GOOS {
		t.Errorf("Uname() OS = %q, expected %q", goos, runtime.GOOS)
	}

	if goarch != runtime.GOARCH {
		t.Errorf("Uname() Arch = %q, expected %q", goarch, runtime.GOARCH)
	}
}

func TestLocal_WalkDir(t *testing.T) {
	h := NewLocal()

	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "local_walkdir_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test structure
	subDir := filepath.Join(tempDir, "subdir")
	err = h.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	files := []string{
		filepath.Join(tempDir, "file1.txt"),
		filepath.Join(subDir, "file2.txt"),
	}

	for _, file := range files {
		err := h.WriteFile(file, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("WriteFile failed for %s: %v", file, err)
		}
	}

	// Walk directory
	var walkEntries []string
	err = h.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		walkEntries = append(walkEntries, path)
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}

	// Should have at least: ., file1.txt, subdir, subdir/file2.txt
	if len(walkEntries) < 4 {
		t.Errorf("Expected at least 4 entries, got %d: %v", len(walkEntries), walkEntries)
	}

	// Check that root entry is "."
	if walkEntries[0] != "." {
		t.Errorf("First entry should be '.', got %q", walkEntries[0])
	}
}

func TestLocal_ProcessOperations(t *testing.T) {
	h := NewLocal()

	// Test with a simple command that should exist on most systems
	t.Run("Run command", func(t *testing.T) {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", "echo", "test")
		} else {
			cmd = exec.Command("echo", "test")
		}

		output, err := h.Run(cmd, "")
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		outputStr := strings.TrimSpace(string(output))
		if outputStr != "test" {
			t.Errorf("Expected output 'test', got %q", outputStr)
		}
	})

	t.Run("Start command", func(t *testing.T) {
		// Create a temporary script that will exit quickly
		tempDir, err := os.MkdirTemp("", "local_start_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			// Create a batch file
			scriptFile := filepath.Join(tempDir, "test.bat")
			err = os.WriteFile(scriptFile, []byte("@echo test"), 0755)
			if err != nil {
				t.Fatalf("Failed to create script: %v", err)
			}
			cmd = exec.Command(scriptFile)
		} else {
			// Create a shell script
			scriptFile := filepath.Join(tempDir, "test.sh")
			err = os.WriteFile(scriptFile, []byte("#!/bin/sh\necho test\n"), 0755)
			if err != nil {
				t.Fatalf("Failed to create script: %v", err)
			}
			cmd = exec.Command("/bin/sh", scriptFile)
		}

		cmd.Dir = tempDir
		err = h.Start(cmd, "")
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// Note: We don't wait for the process as Start() is meant to detach
	})
}

// Skip signal test on Windows as it's different
func TestLocal_Signal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Signal test skipped on Windows")
	}

	h := NewLocal()

	// Test with current process (safe signal)
	pid := os.Getpid()
	err := h.Signal(pid, syscall.Signal(0)) // Signal 0 is used to check if process exists
	if err != nil {
		t.Errorf("Signal(0) failed: %v", err)
	}

	// Test with non-existent PID
	err = h.Signal(999999, syscall.SIGTERM)
	// This should either succeed (if PID exists) or fail gracefully
	// We don't assert on the result as it depends on system state
}

// Test ownership functions (skip on Windows as they're not meaningful there)
func TestLocal_Ownership(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Ownership tests skipped on Windows")
	}

	h := NewLocal()

	// Create temporary file
	tempDir, err := os.MkdirTemp("", "local_ownership_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "ownership_test.txt")
	err = h.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Get current file info
	fi, err := h.Stat(testFile)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// Test Chown (changing to same owner should work)
	stat := fi.Sys().(*syscall.Stat_t)
	err = h.Chown(testFile, int(stat.Uid), int(stat.Gid))
	if err != nil {
		t.Errorf("Chown failed: %v", err)
	}

	// Test Lchown
	err = h.Lchown(testFile, int(stat.Uid), int(stat.Gid))
	if err != nil {
		t.Errorf("Lchown failed: %v", err)
	}
}

func TestLocal_Symlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink tests skipped on Windows")
	}

	h := NewLocal()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "local_symlink_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create target file
	targetFile := filepath.Join(tempDir, "target.txt")
	err = h.WriteFile(targetFile, []byte("target content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create symlink
	linkFile := filepath.Join(tempDir, "link.txt")
	err = h.Symlink(targetFile, linkFile)
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	// Test Readlink
	linkTarget, err := h.Readlink(linkFile)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}

	if linkTarget != targetFile {
		t.Errorf("Readlink returned %q, expected %q", linkTarget, targetFile)
	}

	// Test Lstat vs Stat
	lfi, err := h.Lstat(linkFile)
	if err != nil {
		t.Fatalf("Lstat failed: %v", err)
	}

	if lfi.Mode()&fs.ModeSymlink == 0 {
		t.Error("Lstat should show symlink mode")
	}

	sfi, err := h.Stat(linkFile)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if sfi.Mode()&fs.ModeSymlink != 0 {
		t.Error("Stat should not show symlink mode (should follow link)")
	}
}