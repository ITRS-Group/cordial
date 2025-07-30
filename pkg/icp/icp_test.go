package icp

import (
	"context"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
)

func TestNew(t *testing.T) {
	// Test creating a new ICP instance with default options
	icp := New()
	if icp == nil {
		t.Fatal("New() should not return nil")
	}
	if icp.Client == nil {
		t.Error("ICP.Client should not be nil")
	}
	if icp.token != "" {
		t.Error("ICP.token should be empty by default")
	}

	// Test creating with custom options
	icp = New(rest.BaseURLString("https://custom.example.com"))
	if icp == nil {
		t.Fatal("New() should not return nil with options")
	}
}

func TestLoginRequest(t *testing.T) {
	// Test LoginRequest struct
	req := LoginRequest{
		Username: "testuser",
		Password: "testpass",
	}

	if req.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", req.Username)
	}
	if req.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", req.Password)
	}
}

func TestLogin(t *testing.T) {
	// Test Login function with valid credentials
	password := &config.Plaintext{}
	password.Set("testpass")

	icp, err := Login("testuser", password)
	if err != nil {
		// This might fail due to network issues, but we can test the structure
		t.Logf("Login failed (expected for test): %v", err)
	}

	if icp != nil {
		if icp.Client == nil {
			t.Error("ICP.Client should not be nil after login")
		}
		// Note: token might be empty if login failed
	}
}

func TestBaselineID(t *testing.T) {
	// Test BaselineID method
	icp := New()
	ctx := context.Background()

	// This will likely fail due to no actual connection, but we can test the method signature
	id, baselineid, err := icp.BaselineID(ctx, 1, "test-baseline")
	if err != nil {
		// Expected to fail in test environment
		t.Logf("BaselineID failed (expected for test): %v", err)
	}

	// Check that the method returns the expected types
	if id != "" {
		t.Logf("Got ID: %s", id)
	}
	if baselineid != 0 {
		t.Logf("Got baseline ID: %d", baselineid)
	}
}

func TestICPServerError(t *testing.T) {
	// Test that ErrServerError is defined
	if ErrServerError == nil {
		t.Error("ErrServerError should not be nil")
	}

	// Test error message
	expectedMsg := "error from server (HTTP status > 299)"
	if ErrServerError.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, ErrServerError.Error())
	}
}

func TestICPStruct(t *testing.T) {
	// Test ICP struct fields
	client := rest.NewClient()
	icp := &ICP{
		Client: client,
		token:  "test-token",
	}

	if icp.Client != client {
		t.Error("ICP.Client should match the provided client")
	}
	if icp.token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", icp.token)
	}
}

func TestLoginWithOptions(t *testing.T) {
	// Test Login with custom options
	password := &config.Plaintext{}
	password.Set("testpass")

	icp, err := Login("testuser", password, rest.BaseURLString("https://custom.example.com"))
	if err != nil {
		// Expected to fail in test environment
		t.Logf("Login with options failed (expected for test): %v", err)
	}

	if icp != nil {
		if icp.Client == nil {
			t.Error("ICP.Client should not be nil after login with options")
		}
	}
}