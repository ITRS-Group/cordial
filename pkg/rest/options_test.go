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

package rest

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

// TestEvalOptions tests the evalOptions function
func TestEvalOptions(t *testing.T) {
	t.Run("Default options", func(t *testing.T) {
		opts := evalOptions()
		
		if opts == nil {
			t.Fatal("evalOptions() returned nil")
		}
		
		if opts.baseURL == nil {
			t.Fatal("Default baseURL should not be nil")
		}
		
		if opts.baseURL.String() != "https://localhost" {
			t.Errorf("Expected default baseURL 'https://localhost', got %s", opts.baseURL.String())
		}
		
		if opts.client == nil {
			t.Fatal("Default client should not be nil")
		}
		
		if opts.setupRequest != nil {
			t.Error("Default setupRequest should be nil")
		}
	})

	t.Run("Single option", func(t *testing.T) {
		customURL := "https://api.example.com"
		opts := evalOptions(BaseURLString(customURL))
		
		if opts.baseURL.String() != customURL {
			t.Errorf("Expected baseURL %s, got %s", customURL, opts.baseURL.String())
		}
	})

	t.Run("Multiple options", func(t *testing.T) {
		customURL := "https://multi.example.com"
		customClient := &http.Client{Timeout: 10 * time.Second}
		
		opts := evalOptions(
			BaseURLString(customURL),
			HTTPClient(customClient),
		)
		
		if opts.baseURL.String() != customURL {
			t.Errorf("Expected baseURL %s, got %s", customURL, opts.baseURL.String())
		}
		
		if opts.client != customClient {
			t.Error("Expected custom client to be set")
		}
		
		if opts.client.Timeout != 10*time.Second {
			t.Errorf("Expected client timeout 10s, got %v", opts.client.Timeout)
		}
	})
}

// TestBaseURLString tests the BaseURLString option function
func TestBaseURLString(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{
			name:    "Valid HTTPS URL",
			baseURL: "https://api.example.com",
			wantErr: false,
		},
		{
			name:    "Valid HTTP URL",
			baseURL: "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "URL with path",
			baseURL: "https://api.example.com/v1",
			wantErr: false,
		},
		{
			name:    "URL with port",
			baseURL: "https://example.com:9999",
			wantErr: false,
		},
		{
			name:    "Invalid URL (empty)",
			baseURL: "",
			wantErr: false, // url.Parse doesn't fail on empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := BaseURLString(tt.baseURL)
			
			opts := &restOptions{}
			option(opts)
			
			if opts.baseURL == nil {
				t.Fatal("baseURL should not be nil")
			}
			
			if opts.baseURL.String() != tt.baseURL {
				t.Errorf("Expected baseURL %s, got %s", tt.baseURL, opts.baseURL.String())
			}
		})
	}
}

// TestBaseURL tests the BaseURL option function
func TestBaseURL(t *testing.T) {
	t.Run("Valid URL object", func(t *testing.T) {
		originalURL, err := url.Parse("https://test.example.com/api")
		if err != nil {
			t.Fatalf("Failed to parse test URL: %v", err)
		}
		
		option := BaseURL(originalURL)
		
		opts := &restOptions{}
		option(opts)
		
		if opts.baseURL != originalURL {
			t.Error("Expected baseURL to be the same object")
		}
		
		if opts.baseURL.String() != originalURL.String() {
			t.Errorf("Expected baseURL %s, got %s", originalURL.String(), opts.baseURL.String())
		}
	})

	t.Run("Nil URL", func(t *testing.T) {
		option := BaseURL(nil)
		
		opts := &restOptions{}
		option(opts)
		
		if opts.baseURL != nil {
			t.Error("Expected baseURL to be nil when passed nil")
		}
	})

	t.Run("URL with complex components", func(t *testing.T) {
		originalURL, err := url.Parse("https://user:pass@example.com:8443/api/v2?param=value#fragment")
		if err != nil {
			t.Fatalf("Failed to parse complex URL: %v", err)
		}
		
		option := BaseURL(originalURL)
		
		opts := &restOptions{}
		option(opts)
		
		if opts.baseURL.String() != originalURL.String() {
			t.Errorf("Expected baseURL %s, got %s", originalURL.String(), opts.baseURL.String())
		}
		
		// Verify specific components
		if opts.baseURL.Scheme != "https" {
			t.Errorf("Expected scheme https, got %s", opts.baseURL.Scheme)
		}
		
		if opts.baseURL.Host != "example.com:8443" {
			t.Errorf("Expected host example.com:8443, got %s", opts.baseURL.Host)
		}
		
		if opts.baseURL.Path != "/api/v2" {
			t.Errorf("Expected path /api/v2, got %s", opts.baseURL.Path)
		}
	})
}

// TestHTTPClient tests the HTTPClient option function
func TestHTTPClient(t *testing.T) {
	t.Run("Custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{
			Timeout: 30 * time.Second,
		}
		
		option := HTTPClient(customClient)
		
		opts := &restOptions{}
		option(opts)
		
		if opts.client != customClient {
			t.Error("Expected client to be the same object")
		}
		
		if opts.client.Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", opts.client.Timeout)
		}
	})

	t.Run("Nil HTTP client", func(t *testing.T) {
		option := HTTPClient(nil)
		
		opts := &restOptions{}
		option(opts)
		
		if opts.client != nil {
			t.Error("Expected client to be nil when passed nil")
		}
	})

	t.Run("HTTP client with custom transport", func(t *testing.T) {
		customTransport := &http.Transport{
			MaxIdleConns: 100,
		}
		
		customClient := &http.Client{
			Transport: customTransport,
			Timeout:   15 * time.Second,
		}
		
		option := HTTPClient(customClient)
		
		opts := &restOptions{}
		option(opts)
		
		if opts.client.Transport != customTransport {
			t.Error("Expected custom transport to be preserved")
		}
		
		if opts.client.Timeout != 15*time.Second {
			t.Errorf("Expected timeout 15s, got %v", opts.client.Timeout)
		}
	})
}

// TestSetupRequestFunc tests the SetupRequestFunc option function
func TestSetupRequestFuncOption(t *testing.T) {
	t.Run("Custom setup function", func(t *testing.T) {
		setupCalled := false
		setupFunc := func(req *http.Request, c *Client, body []byte) {
			setupCalled = true
		}
		
		option := SetupRequestFunc(setupFunc)
		
		opts := &restOptions{}
		option(opts)
		
		if opts.setupRequest == nil {
			t.Fatal("setupRequest should not be nil")
		}
		
		// Test that the function works
		opts.setupRequest(nil, nil, nil)
		
		if !setupCalled {
			t.Error("Setup function should have been called")
		}
	})

	t.Run("Nil setup function", func(t *testing.T) {
		option := SetupRequestFunc(nil)
		
		opts := &restOptions{}
		option(opts)
		
		if opts.setupRequest != nil {
			t.Error("Expected setupRequest to be nil when passed nil")
		}
	})

	t.Run("Setup function with parameters", func(t *testing.T) {
		var capturedReq *http.Request
		var capturedClient *Client
		var capturedBody []byte
		
		setupFunc := func(req *http.Request, c *Client, body []byte) {
			capturedReq = req
			capturedClient = c
			capturedBody = body
		}
		
		option := SetupRequestFunc(setupFunc)
		
		opts := &restOptions{}
		option(opts)
		
		// Create test data
		testReq, _ := http.NewRequest("GET", "http://example.com", nil)
		testClient := &Client{}
		testBody := []byte("test body")
		
		// Call the setup function
		opts.setupRequest(testReq, testClient, testBody)
		
		if capturedReq != testReq {
			t.Error("Request should be passed correctly")
		}
		
		if capturedClient != testClient {
			t.Error("Client should be passed correctly")
		}
		
		if string(capturedBody) != string(testBody) {
			t.Errorf("Expected body %s, got %s", string(testBody), string(capturedBody))
		}
	})
}

// TestOptionsIntegration tests how options work together
func TestOptionsIntegration(t *testing.T) {
	t.Run("All options together", func(t *testing.T) {
		baseURL := "https://integration.example.com/api"
		customClient := &http.Client{Timeout: 25 * time.Second}
		setupCalled := false
		setupFunc := func(req *http.Request, c *Client, body []byte) {
			setupCalled = true
		}
		
		opts := evalOptions(
			BaseURLString(baseURL),
			HTTPClient(customClient),
			SetupRequestFunc(setupFunc),
		)
		
		// Verify baseURL
		if opts.baseURL.String() != baseURL {
			t.Errorf("Expected baseURL %s, got %s", baseURL, opts.baseURL.String())
		}
		
		// Verify HTTP client
		if opts.client != customClient {
			t.Error("Expected custom HTTP client")
		}
		
		if opts.client.Timeout != 25*time.Second {
			t.Errorf("Expected timeout 25s, got %v", opts.client.Timeout)
		}
		
		// Verify setup function
		if opts.setupRequest == nil {
			t.Fatal("setupRequest should not be nil")
		}
		
		opts.setupRequest(nil, nil, nil)
		if !setupCalled {
			t.Error("Setup function should have been called")
		}
	})

	t.Run("Options override order", func(t *testing.T) {
		firstURL := "https://first.example.com"
		secondURL := "https://second.example.com"
		
		opts := evalOptions(
			BaseURLString(firstURL),
			BaseURLString(secondURL), // This should override the first
		)
		
		if opts.baseURL.String() != secondURL {
			t.Errorf("Expected second URL %s to override first, got %s", secondURL, opts.baseURL.String())
		}
	})

	t.Run("Empty options list", func(t *testing.T) {
		opts := evalOptions()
		
		// Should have default values
		if opts.baseURL.String() != "https://localhost" {
			t.Errorf("Expected default baseURL, got %s", opts.baseURL.String())
		}
		
		if opts.client == nil {
			t.Error("Expected default client")
		}
		
		if opts.setupRequest != nil {
			t.Error("Expected no setup function by default")
		}
	})
}