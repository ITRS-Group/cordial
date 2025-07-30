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
	"testing"

	"github.com/awnumar/memguard"
)

func TestNewSSHRemote(t *testing.T) {
	tests := []struct {
		name    string
		remName string
		options []any
	}{
		{
			name:    "Basic SSH remote",
			remName: "test-host",
			options: []any{},
		},
		{
			name:    "SSH remote with username",
			remName: "test-host",
			options: []any{Username("testuser")},
		},
		{
			name:    "SSH remote with hostname",
			remName: "test-host",
			options: []any{Hostname("example.com")},
		},
		{
			name:    "SSH remote with port",
			remName: "test-host",
			options: []any{Port(2222)},
		},
		{
			name:    "SSH remote with multiple options",
			remName: "test-host",
			options: []any{
				Username("testuser"),
				Hostname("example.com"),
				Port(2222),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewSSHRemote(tt.remName, tt.options...)
			if h == nil {
				t.Fatal("NewSSHRemote() returned nil")
			}

			ssh, ok := h.(*SSHRemote)
			if !ok {
				t.Fatal("NewSSHRemote() did not return *SSHRemote")
			}

			if ssh.name != tt.remName {
				t.Errorf("Expected name %q, got %q", tt.remName, ssh.name)
			}

			// Test that it implements Host interface
			var _ Host = h
		})
	}
}

func TestSSHOptions(t *testing.T) {
	t.Run("Username option", func(t *testing.T) {
		h := NewSSHRemote("test", Username("myuser"))
		ssh := h.(*SSHRemote)
		if ssh.username != "myuser" {
			t.Errorf("Expected username %q, got %q", "myuser", ssh.username)
		}
	})

	t.Run("Hostname option", func(t *testing.T) {
		h := NewSSHRemote("test", Hostname("example.com"))
		ssh := h.(*SSHRemote)
		if ssh.hostname != "example.com" {
			t.Errorf("Expected hostname %q, got %q", "example.com", ssh.hostname)
		}
	})

	t.Run("Port option", func(t *testing.T) {
		h := NewSSHRemote("test", Port(2222))
		ssh := h.(*SSHRemote)
		if ssh.port != 2222 {
			t.Errorf("Expected port %d, got %d", 2222, ssh.port)
		}
	})

	t.Run("Password option", func(t *testing.T) {
		password := memguard.NewEnclave([]byte("testpass"))

		h := NewSSHRemote("test", Password(password))
		ssh := h.(*SSHRemote)
		if ssh.password != password {
			t.Error("Password option not set correctly")
		}
	})

	t.Run("PrivateKeyFiles option", func(t *testing.T) {
		keyFiles := []string{"/path/to/key1", "/path/to/key2"}
		h := NewSSHRemote("test", PrivateKeyFiles(keyFiles...))
		ssh := h.(*SSHRemote)
		
		if len(ssh.keys) != len(keyFiles) {
			t.Errorf("Expected %d key files, got %d", len(keyFiles), len(ssh.keys))
		}

		for i, key := range keyFiles {
			if ssh.keys[i] != key {
				t.Errorf("Expected key file %q, got %q", key, ssh.keys[i])
			}
		}
	})

	t.Run("Multiple PrivateKeyFiles calls", func(t *testing.T) {
		h := NewSSHRemote("test", 
			PrivateKeyFiles("/key1", "/key2"),
			PrivateKeyFiles("/key3"),
		)
		ssh := h.(*SSHRemote)
		
		expected := []string{"/key1", "/key2", "/key3"}
		if len(ssh.keys) != len(expected) {
			t.Errorf("Expected %d key files, got %d", len(expected), len(ssh.keys))
		}

		for i, expectedKey := range expected {
			if ssh.keys[i] != expectedKey {
				t.Errorf("Expected key file %q at index %d, got %q", expectedKey, i, ssh.keys[i])
			}
		}
	})
}

func TestSSHRemote_BasicMethods(t *testing.T) {
	h := NewSSHRemote("test-host", 
		Username("testuser"),
		Hostname("example.com"),
		Port(2222),
	)

	t.Run("Username", func(t *testing.T) {
		if h.Username() != "testuser" {
			t.Errorf("Expected username %q, got %q", "testuser", h.Username())
		}
	})

	t.Run("Hostname", func(t *testing.T) {
		if h.Hostname() != "example.com" {
			t.Errorf("Expected hostname %q, got %q", "example.com", h.Hostname())
		}
	})

	t.Run("IsLocal", func(t *testing.T) {
		if h.IsLocal() {
			t.Error("SSH remote should not be local")
		}
	})

	t.Run("String", func(t *testing.T) {
		str := h.String()
		if str == "" {
			t.Error("String() should return non-empty string")
		}
		// The string representation should include the name
		if str != "test-host" {
			t.Errorf("Expected string representation to include name, got %q", str)
		}
	})

	t.Run("HostPath", func(t *testing.T) {
		testPath := "/some/path"
		result := h.HostPath(testPath)
		expected := "test-host:" + testPath
		if result != expected {
			t.Errorf("HostPath(%q) = %q, expected %q", testPath, result, expected)
		}
	})
}

func TestSSHRemote_IsAvailable(t *testing.T) {
	t.Run("No hostname set", func(t *testing.T) {
		h := NewSSHRemote("test")
		available, err := h.IsAvailable()
		if available {
			t.Error("Host without hostname should not be available")
		}
		if err == nil {
			t.Error("Expected error when hostname not set")
		}
	})

	t.Run("With hostname but no connection", func(t *testing.T) {
		h := NewSSHRemote("test", Hostname("nonexistent.example.com"))
		available, err := h.IsAvailable()
		// This should fail because we can't actually connect
		if available {
			t.Error("Non-existent host should not be available")
		}
		if err == nil {
			t.Error("Expected error when connecting to non-existent host")
		}
	})
}

func TestSSHRemote_Dial_ValidationErrors(t *testing.T) {
	t.Run("Nil receiver", func(t *testing.T) {
		var h *SSHRemote
		_, err := h.Dial()
		if err != ErrInvalidArgs {
			t.Errorf("Expected ErrInvalidArgs, got %v", err)
		}
	})

	t.Run("Empty hostname", func(t *testing.T) {
		h := NewSSHRemote("test").(*SSHRemote)
		_, err := h.Dial()
		if err == nil {
			t.Error("Expected error when hostname is empty")
		}
	})

	t.Run("Recent failure cached", func(t *testing.T) {
		h := NewSSHRemote("test", Hostname("nonexistent.example.com")).(*SSHRemote)
		
		// First attempt should fail
		_, err1 := h.Dial()
		if err1 == nil {
			t.Error("Expected first dial to fail")
		}

		// Second attempt within 5 seconds should return cached error
		_, err2 := h.Dial()
		if err2 == nil {
			t.Error("Expected second dial to return cached error")
		}
		
		// The error should be the same (cached)
		if err2 != h.failed {
			t.Error("Expected cached error to be returned")
		}
	})
}

func TestSSHRemote_DefaultValues(t *testing.T) {
	h := NewSSHRemote("test", Hostname("example.com"))
	ssh := h.(*SSHRemote)

	// Test that username defaults to current user if not specified
	if ssh.username == "" {
		t.Error("Username should have a default value")
	}

	// Test that port defaults are handled in Dial()
	if ssh.port != 0 {
		t.Error("Port should be 0 initially (will default to 22 in Dial)")
	}
}

func TestSSHRemote_LastError(t *testing.T) {
	h := NewSSHRemote("test", Hostname("nonexistent.example.com"))
	ssh := h.(*SSHRemote)

	// Initially should have no error
	if h.LastError() != nil {
		t.Error("LastError() should initially return nil")
	}

	// After failed dial, should have error
	// Note: We can't test Dial() directly as it's not in the Host interface
	// Instead, test IsAvailable which internally uses similar logic
	_, err := h.IsAvailable()
	if err == nil {
		t.Error("Expected IsAvailable to fail")
	}

	lastErr := h.LastError()
	if lastErr == nil {
		t.Error("LastError() should return error after failed dial")
	}

	if lastErr != ssh.failed {
		t.Error("LastError() should return the same error as stored in failed field")
	}
}

func TestSSHRemote_ServerVersion(t *testing.T) {
	h := NewSSHRemote("test", Hostname("example.com"))

	// Without a connection, this should return empty or handle gracefully
	version := h.ServerVersion()
	// The actual behavior depends on implementation
	// We just test that it doesn't panic
	_ = version
}

func TestSSHRemote_GetFs(t *testing.T) {
	h := NewSSHRemote("test", Hostname("example.com"))

	// Without a connection, this might return nil or an error filesystem
	fs := h.GetFs()
	// We just test that it doesn't panic
	_ = fs
}

func TestSSHRemote_TempDir(t *testing.T) {
	h := NewSSHRemote("test", Hostname("example.com"))

	// Without a connection, this should handle gracefully
	tempDir := h.TempDir()
	// The actual behavior depends on implementation
	// We just test that it doesn't panic
	_ = tempDir
}

func TestSSHRemote_FileOperations_NoConnection(t *testing.T) {
	h := NewSSHRemote("test", Hostname("nonexistent.example.com"))

	// Test that file operations fail gracefully when not connected
	testCases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ReadFile",
			fn: func() error {
				_, err := h.ReadFile("/tmp/test")
				return err
			},
		},
		{
			name: "WriteFile",
			fn: func() error {
				return h.WriteFile("/tmp/test", []byte("test"), 0644)
			},
		},
		{
			name: "Stat",
			fn: func() error {
				_, err := h.Stat("/tmp")
				return err
			},
		},
		{
			name: "Lstat",
			fn: func() error {
				_, err := h.Lstat("/tmp")
				return err
			},
		},
		{
			name: "MkdirAll",
			fn: func() error {
				return h.MkdirAll("/tmp/test", 0755)
			},
		},
		{
			name: "Remove",
			fn: func() error {
				return h.Remove("/tmp/test")
			},
		},
		{
			name: "RemoveAll",
			fn: func() error {
				return h.RemoveAll("/tmp/test")
			},
		},
		{
			name: "Rename",
			fn: func() error {
				return h.Rename("/tmp/old", "/tmp/new")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if err == nil {
				t.Errorf("%s should fail when not connected", tc.name)
			}
		})
	}
}

func TestSSHRemote_ProcessOperations_NoConnection(t *testing.T) {
	h := NewSSHRemote("test", Hostname("nonexistent.example.com"))

	t.Run("Signal", func(t *testing.T) {
		err := h.Signal(1234, 15) // SIGTERM
		if err == nil {
			t.Error("Signal should fail when not connected")
		}
	})
}

func TestReadSSHKeys(t *testing.T) {
	// Test with empty file list
	signers := readSSHkeys(nil, "/tmp", []string{}...)
	if len(signers) != 0 {
		t.Errorf("Expected 0 signers for empty file list, got %d", len(signers))
	}

	// Test with non-existent files
	signers = readSSHkeys(nil, "/tmp", "/nonexistent/key1", "/nonexistent/key2")
	if len(signers) != 0 {
		t.Errorf("Expected 0 signers for non-existent files, got %d", len(signers))
	}
}

func TestEvalOptions(t *testing.T) {
	remote := &SSHRemote{}

	// Test with no options
	evalOptions(remote)
	if remote.username == "" {
		t.Error("Username should have default value")
	}

	// Test with valid options
	remote2 := &SSHRemote{}
	evalOptions(remote2, Username("testuser"), Port(2222))
	if remote2.username != "testuser" {
		t.Errorf("Expected username 'testuser', got %q", remote2.username)
	}
	if remote2.port != 2222 {
		t.Errorf("Expected port 2222, got %d", remote2.port)
	}

	// Test with invalid option type (should be ignored)
	remote3 := &SSHRemote{}
	evalOptions(remote3, "invalid option")
	// Should not panic and should set defaults
	if remote3.username == "" {
		t.Error("Username should have default value even with invalid option")
	}
}

// Test the SSH options functions independently
func TestSSHOptionFunctions(t *testing.T) {
	t.Run("Username function", func(t *testing.T) {
		opt := Username("testuser")
		remote := &SSHRemote{}
		opt(remote)
		if remote.username != "testuser" {
			t.Errorf("Expected username 'testuser', got %q", remote.username)
		}
	})

	t.Run("Password function", func(t *testing.T) {
		password := memguard.NewEnclave([]byte("secret"))
		
		opt := Password(password)
		remote := &SSHRemote{}
		opt(remote)
		if remote.password != password {
			t.Error("Password not set correctly")
		}
	})

	t.Run("Port function", func(t *testing.T) {
		opt := Port(8022)
		remote := &SSHRemote{}
		opt(remote)
		if remote.port != 8022 {
			t.Errorf("Expected port 8022, got %d", remote.port)
		}
	})

	t.Run("Hostname function", func(t *testing.T) {
		opt := Hostname("test.example.com")
		remote := &SSHRemote{}
		opt(remote)
		if remote.hostname != "test.example.com" {
			t.Errorf("Expected hostname 'test.example.com', got %q", remote.hostname)
		}
	})

	t.Run("PrivateKeyFiles function", func(t *testing.T) {
		opt := PrivateKeyFiles("/key1", "/key2", "/key3")
		remote := &SSHRemote{}
		opt(remote)
		
		expected := []string{"/key1", "/key2", "/key3"}
		if len(remote.keys) != len(expected) {
			t.Errorf("Expected %d keys, got %d", len(expected), len(remote.keys))
		}
		
		for i, key := range expected {
			if remote.keys[i] != key {
				t.Errorf("Expected key %q at index %d, got %q", key, i, remote.keys[i])
			}
		}
	})
}