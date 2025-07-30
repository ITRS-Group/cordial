package streams

import (
	"io"
	"net/url"
	"testing"
)

func TestOpen(t *testing.T) {
	// Test with valid URL
	validURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse valid URL: %v", err)
	}

	// Test with stream name
	stream, err := Open(validURL, "testEntity", "testSampler", "testStream")
	if err != nil {
		t.Fatalf("Open failed with stream name: %v", err)
	}

	if stream == nil {
		t.Fatal("Open returned nil stream")
	}

	if stream.name != "testStream" {
		t.Errorf("Expected stream name 'testStream', got '%s'", stream.name)
	}

	// Test without stream name (should use sampler name)
	stream, err = Open(validURL, "testEntity", "testSampler", "")
	if err != nil {
		t.Fatalf("Open failed without stream name: %v", err)
	}

	if stream.name != "testSampler" {
		t.Errorf("Expected stream name 'testSampler', got '%s'", stream.name)
	}

	// Test with empty entity name (should work as it's not validated)
	stream, err = Open(validURL, "", "testSampler", "testStream")
	if err != nil {
		t.Fatalf("Open failed with empty entity name: %v", err)
	}

	if stream == nil {
		t.Fatal("Open returned nil stream with empty entity name")
	}

	// Test with empty sampler name (should work as it's not validated)
	stream, err = Open(validURL, "testEntity", "", "testStream")
	if err != nil {
		t.Fatalf("Open failed with empty sampler name: %v", err)
	}

	if stream == nil {
		t.Fatal("Open returned nil stream with empty sampler name")
	}
}

func TestStreamWrite(t *testing.T) {
	validURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse valid URL: %v", err)
	}

	stream, err := Open(validURL, "testEntity", "testSampler", "testStream")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Test Write with data (will fail due to network connection, but we can test the logic)
	testData := []byte("test data")
	n, err := stream.Write(testData)
	// We expect an error due to network connection, and the length should be 0 when there's an error
	if n != 0 {
		t.Errorf("Expected written length 0 when there's an error, got %d", n)
	}

	// Test Write with empty stream name
	emptyStream := &Stream{name: ""}
	_, err = emptyStream.Write(testData)
	if err == nil {
		t.Error("Expected error when stream name is empty")
	}
}

func TestStreamWriteString(t *testing.T) {
	validURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse valid URL: %v", err)
	}

	stream, err := Open(validURL, "testEntity", "testSampler", "testStream")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Test WriteString with data (will fail due to network connection, but we can test the logic)
	testData := "test string data"
	n, err := stream.WriteString(testData)
	// We expect an error due to network connection, and the length should be 0 when there's an error
	if n != 0 {
		t.Errorf("Expected written length 0 when there's an error, got %d", n)
	}

	// Test WriteString with empty stream name
	emptyStream := &Stream{name: ""}
	_, err = emptyStream.WriteString(testData)
	if err == nil {
		t.Error("Expected error when stream name is empty")
	}
}

func TestNewRESTStream(t *testing.T) {
	validURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse valid URL: %v", err)
	}

	stream, err := NewRESTStream(validURL, "testEntity", "testSampler", "testStream")
	if err != nil {
		t.Fatalf("NewRESTStream failed: %v", err)
	}

	if stream.baseurl == nil {
		t.Error("RESTStream baseurl should not be nil")
	}

	if stream.client == nil {
		t.Error("RESTStream client should not be nil")
	}

	// Test with empty entity name (should work as it's not validated)
	stream, err = NewRESTStream(validURL, "", "testSampler", "testStream")
	if err != nil {
		t.Fatalf("NewRESTStream failed with empty entity name: %v", err)
	}

	if stream.baseurl == nil {
		t.Error("RESTStream baseurl should not be nil with empty entity name")
	}

	// Test with empty sampler name (should work as it's not validated)
	stream, err = NewRESTStream(validURL, "testEntity", "", "testStream")
	if err != nil {
		t.Fatalf("NewRESTStream failed with empty sampler name: %v", err)
	}

	if stream.baseurl == nil {
		t.Error("RESTStream baseurl should not be nil with empty sampler name")
	}

	// Test with empty stream name (should work as it's not validated)
	stream, err = NewRESTStream(validURL, "testEntity", "testSampler", "")
	if err != nil {
		t.Fatalf("NewRESTStream failed with empty stream name: %v", err)
	}

	if stream.baseurl == nil {
		t.Error("RESTStream baseurl should not be nil with empty stream name")
	}
}

func TestRESTStreamWrite(t *testing.T) {
	validURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse valid URL: %v", err)
	}

	stream, err := NewRESTStream(validURL, "testEntity", "testSampler", "testStream")
	if err != nil {
		t.Fatalf("NewRESTStream failed: %v", err)
	}

	// Test Write with data (will fail due to network connection, but we can test the logic)
	testData := []byte("test data")
	n, err := stream.Write(testData)
	// We expect an error due to network connection, and the length should be 0 when there's an error
	if n != 0 {
		t.Errorf("Expected written length 0 when there's an error, got %d", n)
	}

	// Test Write with empty data
	emptyData := []byte{}
	n, err = stream.Write(emptyData)
	// We expect an error due to network connection, and the length should be 0 when there's an error
	if n != 0 {
		t.Errorf("Expected written length 0 when there's an error, got %d", n)
	}
}

func TestStreamImplementsInterfaces(t *testing.T) {
	validURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse valid URL: %v", err)
	}

	stream, err := Open(validURL, "testEntity", "testSampler", "testStream")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Test that Stream implements io.Writer
	var _ io.Writer = stream

	// Test that Stream implements io.StringWriter
	var _ io.StringWriter = stream
}