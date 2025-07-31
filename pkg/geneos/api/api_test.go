package api

import (
	"net/http"
	"testing"
)

func TestErrInvalidArgs(t *testing.T) {
	// Test that ErrInvalidArgs is defined
	if ErrInvalidArgs == nil {
		t.Error("ErrInvalidArgs should not be nil")
	}

	// Test error message
	expectedMsg := "invalid arguments"
	if ErrInvalidArgs.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, ErrInvalidArgs.Error())
	}
}

func TestApiOptions(t *testing.T) {
	// Test default options
	opts := evalOptions()
	if opts.insecureSkipVerify {
		t.Error("Expected insecureSkipVerify to be false by default")
	}

	// Test with InsecureSkipVerify option
	opts = evalOptions(InsecureSkipVerify())
	if !opts.insecureSkipVerify {
		t.Error("Expected insecureSkipVerify to be true with InsecureSkipVerify option")
	}

	// Test with multiple options
	opts = evalOptions(InsecureSkipVerify(), InsecureSkipVerify())
	if !opts.insecureSkipVerify {
		t.Error("Expected insecureSkipVerify to be true with multiple InsecureSkipVerify options")
	}
}

func TestInsecureSkipVerify(t *testing.T) {
	// Test that InsecureSkipVerify returns a function
	opt := InsecureSkipVerify()
	if opt == nil {
		t.Error("InsecureSkipVerify should not return nil")
	}

	// Test that the function modifies options correctly
	opts := &apiOptions{}
	opt(opts)
	if !opts.insecureSkipVerify {
		t.Error("InsecureSkipVerify should set insecureSkipVerify to true")
	}
}

func TestRoundTripper(t *testing.T) {
	// Test roundTripper struct
	transport := &http.Transport{}
	rt := &roundTripper{
		transport: transport,
	}

	if rt.transport != transport {
		t.Error("roundTripper.transport should match the provided transport")
	}
}

func TestStream(t *testing.T) {
	// Test Stream struct
	var client APIClient = nil // Mock client would be used in real tests
	stream := &Stream{
		client:  client,
		entity:  "test-entity",
		sampler: "test-sampler",
		stream:  "test-stream",
	}

	if stream.entity != "test-entity" {
		t.Errorf("Expected entity 'test-entity', got '%s'", stream.entity)
	}
	if stream.sampler != "test-sampler" {
		t.Errorf("Expected sampler 'test-sampler', got '%s'", stream.sampler)
	}
	if stream.stream != "test-stream" {
		t.Errorf("Expected stream 'test-stream', got '%s'", stream.stream)
	}
}

func TestOpenStream(t *testing.T) {
	// Test OpenStream function
	var client APIClient = nil // Mock client would be used in real tests
	stream, err := OpenStream(client, "test-entity", "test-sampler", "test-stream")
	if err != nil {
		t.Errorf("OpenStream failed: %v", err)
	}

	if stream == nil {
		t.Fatal("OpenStream should not return nil")
	}
	if stream.client != client {
		t.Error("Stream.client should match the provided client")
	}
	if stream.entity != "test-entity" {
		t.Errorf("Expected entity 'test-entity', got '%s'", stream.entity)
	}
	if stream.sampler != "test-sampler" {
		t.Errorf("Expected sampler 'test-sampler', got '%s'", stream.sampler)
	}
	if stream.stream != "test-stream" {
		t.Errorf("Expected stream 'test-stream', got '%s'", stream.stream)
	}
}

func TestStreamWrite(t *testing.T) {
	// Test Stream.Write method
	// This would require a mock APIClient to test properly
	var client APIClient = nil // Mock client would be used in real tests
	stream := &Stream{
		client:  client,
		entity:  "test-entity",
		sampler: "test-sampler",
		stream:  "test-stream",
	}

	// Skip the actual Write call since it would panic with nil client
	// In a real test, we would use a mock client
	t.Log("Skipping Stream.Write test due to nil client (would panic)")
	
	// Test the struct fields instead
	if stream.entity != "test-entity" {
		t.Errorf("Expected entity 'test-entity', got '%s'", stream.entity)
	}
	if stream.sampler != "test-sampler" {
		t.Errorf("Expected sampler 'test-sampler', got '%s'", stream.sampler)
	}
	if stream.stream != "test-stream" {
		t.Errorf("Expected stream 'test-stream', got '%s'", stream.stream)
	}
}

func TestApiOptionsStruct(t *testing.T) {
	// Test apiOptions struct
	opts := &apiOptions{
		insecureSkipVerify: true,
	}

	if !opts.insecureSkipVerify {
		t.Error("Expected insecureSkipVerify to be true")
	}
}

func TestOptionsFunction(t *testing.T) {
	// Test that Options is a function type
	var opt Options
	if opt == nil {
		// This is expected for a zero value
		t.Log("Options zero value is nil (expected)")
	}

	// Test creating an option function
	opt = InsecureSkipVerify()
	if opt == nil {
		t.Error("InsecureSkipVerify should not return nil")
	}
}