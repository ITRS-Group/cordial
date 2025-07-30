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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	// Create temporary directories for testing
	srcDir, err := os.MkdirTemp("", "host_test_src")
	if err != nil {
		t.Fatalf("Failed to create temp src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "host_test_dst")
	if err != nil {
		t.Fatalf("Failed to create temp dst dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Test copying a file
	t.Run("Copy file to file", func(t *testing.T) {
		srcFile := filepath.Join(srcDir, "test.txt")
		dstFile := filepath.Join(dstDir, "copied.txt")
		testContent := []byte("test content")

		// Create source file
		if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
			t.Fatalf("Failed to create src file: %v", err)
		}

		// Copy file
		srcHost := NewLocal()
		dstHost := NewLocal()
		if err := CopyFile(srcHost, srcFile, dstHost, dstFile); err != nil {
			t.Fatalf("CopyFile failed: %v", err)
		}

		// Verify copy
		copiedContent, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}

		if string(copiedContent) != string(testContent) {
			t.Errorf("Content mismatch: got %q, want %q", copiedContent, testContent)
		}
	})

	t.Run("Copy file to directory", func(t *testing.T) {
		srcFile := filepath.Join(srcDir, "test2.txt")
		testContent := []byte("test content 2")

		// Create source file
		if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
			t.Fatalf("Failed to create src file: %v", err)
		}

		// Copy file to directory
		srcHost := NewLocal()
		dstHost := NewLocal()
		if err := CopyFile(srcHost, srcFile, dstHost, dstDir); err != nil {
			t.Fatalf("CopyFile failed: %v", err)
		}

		// Verify copy in destination directory
		expectedDstFile := filepath.Join(dstDir, "test2.txt")
		copiedContent, err := os.ReadFile(expectedDstFile)
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}

		if string(copiedContent) != string(testContent) {
			t.Errorf("Content mismatch: got %q, want %q", copiedContent, testContent)
		}
	})

	t.Run("Copy non-existent file", func(t *testing.T) {
		srcFile := filepath.Join(srcDir, "nonexistent.txt")
		dstFile := filepath.Join(dstDir, "should_not_exist.txt")

		srcHost := NewLocal()
		dstHost := NewLocal()
		err := CopyFile(srcHost, srcFile, dstHost, dstFile)
		if err == nil {
			t.Error("Expected error when copying non-existent file")
		}
	})

	t.Run("Copy directory should fail", func(t *testing.T) {
		srcHost := NewLocal()
		dstHost := NewLocal()
		err := CopyFile(srcHost, srcDir, dstHost, dstDir)
		if err != fs.ErrInvalid {
			t.Errorf("Expected fs.ErrInvalid when copying directory, got: %v", err)
		}
	})
}

func TestCopyAll(t *testing.T) {
	// Create temporary directories for testing
	srcDir, err := os.MkdirTemp("", "host_test_copyall_src")
	if err != nil {
		t.Fatalf("Failed to create temp src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "host_test_copyall_dst")
	if err != nil {
		t.Fatalf("Failed to create temp dst dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	t.Run("Copy directory structure", func(t *testing.T) {
		// Create source directory structure
		subDir := filepath.Join(srcDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}

		// Create files
		file1 := filepath.Join(srcDir, "file1.txt")
		file2 := filepath.Join(subDir, "file2.txt")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		// Copy all
		srcHost := NewLocal()
		dstHost := NewLocal()
		if err := CopyAll(srcHost, srcDir, dstHost, dstDir); err != nil {
			t.Fatalf("CopyAll failed: %v", err)
		}

		// Verify files were copied
		copiedFile1 := filepath.Join(dstDir, "file1.txt")
		copiedFile2 := filepath.Join(dstDir, "subdir", "file2.txt")

		content1, err := os.ReadFile(copiedFile1)
		if err != nil {
			t.Errorf("Failed to read copied file1: %v", err)
		} else if string(content1) != "content1" {
			t.Errorf("File1 content mismatch: got %q, want %q", content1, "content1")
		}

		content2, err := os.ReadFile(copiedFile2)
		if err != nil {
			t.Errorf("Failed to read copied file2: %v", err)
		} else if string(content2) != "content2" {
			t.Errorf("File2 content mismatch: got %q, want %q", content2, "content2")
		}
	})
}

func TestProcessDirEntry(t *testing.T) {
	// Create temporary directories for testing
	srcDir, err := os.MkdirTemp("", "host_test_process_src")
	if err != nil {
		t.Fatalf("Failed to create temp src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "host_test_process_dst")
	if err != nil {
		t.Fatalf("Failed to create temp dst dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	srcHost := NewLocal()
	dstHost := NewLocal()

	t.Run("Process directory", func(t *testing.T) {
		subDir := filepath.Join(srcDir, "testdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}

		fi, err := os.Stat(subDir)
		if err != nil {
			t.Fatalf("Failed to stat test dir: %v", err)
		}

		dstPath := filepath.Join(dstDir, "testdir")
		if err := processDirEntry(fi, srcHost, subDir, dstHost, dstPath); err != nil {
			t.Fatalf("processDirEntry failed for directory: %v", err)
		}

		// Verify directory was created
		if _, err := os.Stat(dstPath); err != nil {
			t.Errorf("Destination directory was not created: %v", err)
		}
	})

	t.Run("Process regular file", func(t *testing.T) {
		srcFile := filepath.Join(srcDir, "regular.txt")
		testContent := []byte("regular file content")
		if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		fi, err := os.Stat(srcFile)
		if err != nil {
			t.Fatalf("Failed to stat test file: %v", err)
		}

		dstFile := filepath.Join(dstDir, "regular.txt")
		if err := processDirEntry(fi, srcHost, srcFile, dstHost, dstFile); err != nil {
			t.Fatalf("processDirEntry failed for regular file: %v", err)
		}

		// Verify file was copied
		copiedContent, err := os.ReadFile(dstFile)
		if err != nil {
			t.Errorf("Failed to read copied file: %v", err)
		} else if string(copiedContent) != string(testContent) {
			t.Errorf("Content mismatch: got %q, want %q", copiedContent, testContent)
		}
	})

	// Skip symlink test on Windows as it requires special privileges
	if !isWindows() {
		t.Run("Process symlink", func(t *testing.T) {
			// Create target file
			targetFile := filepath.Join(srcDir, "target.txt")
			if err := os.WriteFile(targetFile, []byte("target content"), 0644); err != nil {
				t.Fatalf("Failed to create target file: %v", err)
			}

			// Create symlink
			linkFile := filepath.Join(srcDir, "link.txt")
			if err := os.Symlink(targetFile, linkFile); err != nil {
				t.Skipf("Failed to create symlink (may not be supported): %v", err)
			}

			fi, err := os.Lstat(linkFile)
			if err != nil {
				t.Fatalf("Failed to lstat symlink: %v", err)
			}

			dstLink := filepath.Join(dstDir, "link.txt")
			if err := processDirEntry(fi, srcHost, linkFile, dstHost, dstLink); err != nil {
				t.Fatalf("processDirEntry failed for symlink: %v", err)
			}

			// Verify symlink was created
			fi, err = os.Lstat(dstLink)
			if err != nil {
				t.Errorf("Failed to stat destination symlink: %v", err)
			} else if fi.Mode()&fs.ModeSymlink == 0 {
				t.Error("Destination is not a symlink")
			}
		})
	}
}

// Helper function to check if running on Windows
func isWindows() bool {
	return filepath.Separator == '\\'
}

func TestErrorConstants(t *testing.T) {
	errors := []error{
		ErrInvalidArgs,
		ErrNotSupported,
		ErrNotAvailable,
		ErrExist,
		ErrNotExist,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Error constant should not be nil")
		}
		if err.Error() == "" {
			t.Error("Error should have a message")
		}
	}
}

// Benchmark tests
func BenchmarkCopyFile(b *testing.B) {
	// Create temporary directories
	srcDir, err := os.MkdirTemp("", "host_bench_src")
	if err != nil {
		b.Fatalf("Failed to create temp src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "host_bench_dst")
	if err != nil {
		b.Fatalf("Failed to create temp dst dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Create test file
	srcFile := filepath.Join(srcDir, "bench.txt")
	testContent := make([]byte, 1024) // 1KB file
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	srcHost := NewLocal()
	dstHost := NewLocal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dstFile := filepath.Join(dstDir, fmt.Sprintf("bench_%d.txt", i))
		if err := CopyFile(srcHost, srcFile, dstHost, dstFile); err != nil {
			b.Fatalf("CopyFile failed: %v", err)
		}
	}
}

// Test that demonstrates proper interface usage
func TestHostInterface(t *testing.T) {
	var h Host = NewLocal()

	// Test basic interface methods
	if h.String() == "" {
		t.Error("Host.String() should return non-empty string")
	}

	if !h.IsLocal() {
		t.Error("Local host should return true for IsLocal()")
	}

	available, err := h.IsAvailable()
	if err != nil {
		t.Errorf("IsAvailable() returned error: %v", err)
	}
	if !available {
		t.Error("Local host should be available")
	}

	if h.Hostname() == "" {
		t.Error("Hostname() should return non-empty string")
	}

	if h.Username() == "" {
		t.Error("Username() should return non-empty string")
	}

	if h.TempDir() == "" {
		t.Error("TempDir() should return non-empty string")
	}

	// Test filesystem operations
	wd, err := h.Getwd()
	if err != nil {
		t.Errorf("Getwd() failed: %v", err)
	}
	if wd == "" {
		t.Error("Getwd() should return non-empty path")
	}

	// Test that absolute path detection works
	if !h.IsAbs(wd) {
		t.Error("Current working directory should be absolute")
	}
}