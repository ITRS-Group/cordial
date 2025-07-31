package plugins

import (
	"net/url"
	"sync"
	"testing"
	"time"
)

// MockPlugins implements the Plugins interface for testing
type MockPlugins struct {
	interval time.Duration
	started  bool
	closed   bool
}

func (m *MockPlugins) SetInterval(d time.Duration) {
	m.interval = d
}

func (m *MockPlugins) Interval() time.Duration {
	return m.interval
}

func (m *MockPlugins) Start(wg *sync.WaitGroup) error {
	m.started = true
	return nil
}

func (m *MockPlugins) Close() error {
	m.closed = true
	return nil
}

func TestOpen(t *testing.T) {
	// Test with valid URL
	validURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse valid URL: %v", err)
	}

	conn, err := Open(validURL, "testEntity", "testSampler")
	if err != nil {
		t.Fatalf("Open failed with valid URL: %v", err)
	}

	if conn == nil {
		t.Fatal("Open returned nil connection")
	}

	// Test with empty entity name (should work as it's not validated)
	conn, err = Open(validURL, "", "testSampler")
	if err != nil {
		t.Fatalf("Open failed with empty entity name: %v", err)
	}

	if conn == nil {
		t.Fatal("Open returned nil connection with empty entity name")
	}

	// Test with empty sampler name (should work as it's not validated)
	conn, err = Open(validURL, "testEntity", "")
	if err != nil {
		t.Fatalf("Open failed with empty sampler name: %v", err)
	}

	if conn == nil {
		t.Fatal("Open returned nil connection with empty sampler name")
	}
}

func TestConnection(t *testing.T) {
	// Test that Connection can be created
	validURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse valid URL: %v", err)
	}

	conn, err := Open(validURL, "testEntity", "testSampler")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Verify that Connection is not nil
	if conn == nil {
		t.Error("Connection should not be nil")
	}
}

func TestPluginsInterface(t *testing.T) {
	mock := &MockPlugins{}

	// Test SetInterval and Interval
	expectedInterval := 5 * time.Second
	mock.SetInterval(expectedInterval)
	if mock.Interval() != expectedInterval {
		t.Errorf("Expected interval %v, got %v", expectedInterval, mock.Interval())
	}

	// Test Start
	var wg sync.WaitGroup
	err := mock.Start(&wg)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}
	if !mock.started {
		t.Error("Start should set started flag")
	}

	// Test Close
	err = mock.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
	if !mock.closed {
		t.Error("Close should set closed flag")
	}
}