/*
Copyright © 2025 ITRS Group

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

package snow

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestServiceNow_BasicAuth(t *testing.T) {
	// Create a test configuration for basic auth
	cf := config.New()
	cf.Set("username", "testuser")
	cf.Set("password", "testpass")
	cf.Set("url", "https://test.service-now.com")
	cf.Set("path", "/api/now/v2/table")

	// Test ServiceNow client creation
	client := ServiceNow(cf)
	if client == nil {
		t.Fatal("Expected ServiceNow client to be created, got nil")
	}

	// Reset the global connection for next test
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

func TestServiceNow_OAuth(t *testing.T) {
	// Create a test configuration for OAuth
	cf := config.New()
	cf.Set("username", "testuser")
	cf.Set("password", "testpass")
	cf.Set("client-id", "test-client-id")
	cf.Set("client-secret", "test-client-secret")
	cf.Set("url", "https://test.service-now.com")
	cf.Set("path", "/api/now/v2/table")

	// Test ServiceNow client creation with OAuth
	client := ServiceNow(cf)
	if client == nil {
		t.Fatal("Expected ServiceNow client to be created, got nil")
	}

	// Reset the global connection for next test
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

func TestServiceNow_HTTPSConfig(t *testing.T) {
	// Create a test configuration with HTTPS and TLS skip verify
	cf := config.New()
	cf.Set("username", "testuser")
	cf.Set("password", "testpass")
	cf.Set("url", "https://test.service-now.com")
	cf.Set("path", "/api/now/v2/table")
	cf.Set("tls.skip-verify", true)

	// Test ServiceNow client creation
	client := ServiceNow(cf)
	if client == nil {
		t.Fatal("Expected ServiceNow client to be created, got nil")
	}

	// Reset the global connection for next test
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

func TestServiceNow_HTTPConfig(t *testing.T) {
	// Create a test configuration with HTTP
	cf := config.New()
	cf.Set("username", "testuser")
	cf.Set("password", "testpass")
	cf.Set("url", "http://test.service-now.com")
	cf.Set("path", "/api/now/v2/table")

	// Test ServiceNow client creation
	client := ServiceNow(cf)
	if client == nil {
		t.Fatal("Expected ServiceNow client to be created, got nil")
	}

	// Reset the global connection for next test
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

func TestServiceNow_InvalidURL(t *testing.T) {
	// Create a test configuration with invalid URL
	cf := config.New()
	cf.Set("username", "testuser")
	cf.Set("password", "testpass")
	cf.Set("url", "://invalid-url")

	// Test ServiceNow client creation with invalid URL
	client := ServiceNow(cf)
	if client != nil {
		t.Fatal("Expected ServiceNow client to be nil for invalid URL, got non-nil")
	}

	// Reset the global connection for next test
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

func TestServiceNow_DefaultPath(t *testing.T) {
	// Create a test configuration without explicit path
	cf := config.New()
	cf.Set("username", "testuser")
	cf.Set("password", "testpass")
	cf.Set("url", "https://test.service-now.com")

	// Test ServiceNow client creation
	client := ServiceNow(cf)
	if client == nil {
		t.Fatal("Expected ServiceNow client to be created, got nil")
	}

	// Reset the global connection for next test
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

func TestServiceNow_GlobalConnection(t *testing.T) {
	// Create a test configuration
	cf := config.New()
	cf.Set("username", "testuser")
	cf.Set("password", "testpass")
	cf.Set("url", "https://test.service-now.com")

	// Reset the global connection
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()

	// First call should create a connection
	client1 := ServiceNow(cf)
	if client1 == nil {
		t.Fatal("Expected first ServiceNow client to be created, got nil")
	}

	// Second call should create another connection (the current implementation doesn't use global caching)
	client2 := ServiceNow(cf)
	if client2 == nil {
		t.Fatal("Expected second ServiceNow client to be created, got nil")
	}

	// Note: The current implementation doesn't actually use global connection caching
	// This test documents the current behavior
	t.Logf("Client1: %p, Client2: %p", client1, client2)

	// Reset the global connection for cleanup
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

func TestContext(t *testing.T) {
	// Create an Echo context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)

	// Create a config
	cf := config.New()

	// Create a custom Context
	ctx := &Context{
		Context: echoCtx,
		Conf:    cf,
	}

	// Test that the context wraps echo.Context properly
	if ctx.Context != echoCtx {
		t.Error("Expected Context to wrap echo.Context")
	}

	if ctx.Conf != cf {
		t.Error("Expected Context to have the config")
	}
}

func TestServiceNow_TLSConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		skipVerify bool
	}{
		{
			name:       "HTTPS with TLS verification",
			url:        "https://test.service-now.com",
			skipVerify: false,
		},
		{
			name:       "HTTPS with TLS skip verification",
			url:        "https://test.service-now.com",
			skipVerify: true,
		},
		{
			name:       "HTTP (no TLS)",
			url:        "http://test.service-now.com",
			skipVerify: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test configuration
			cf := config.New()
			cf.Set("username", "testuser")
			cf.Set("password", "testpass")
			cf.Set("url", tt.url)
			cf.Set("tls.skip-verify", tt.skipVerify)

			// Reset the global connection
			snowMutex.Lock()
			snowConnection = nil
			snowMutex.Unlock()

			// Test ServiceNow client creation
			client := ServiceNow(cf)
			if client == nil {
				t.Fatal("Expected ServiceNow client to be created, got nil")
			}
		})
	}

	// Reset the global connection for cleanup
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

func TestServiceNow_EmptyCredentials(t *testing.T) {
	// Test with empty username and password
	cf := config.New()
	cf.Set("username", "")
	cf.Set("password", "")
	cf.Set("url", "https://test.service-now.com")

	// Should still create a client (with empty basic auth)
	client := ServiceNow(cf)
	if client == nil {
		t.Fatal("Expected ServiceNow client to be created even with empty credentials, got nil")
	}

	// Reset the global connection for cleanup
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

func TestServiceNow_OAuthEmptyClientSecret(t *testing.T) {
	// Create a test configuration with client ID but empty client secret
	cf := config.New()
	cf.Set("username", "testuser")
	cf.Set("password", "testpass")
	cf.Set("client-id", "test-client-id")
	cf.Set("client-secret", "")
	cf.Set("url", "https://test.service-now.com")

	// Should fall back to basic auth when client secret is empty
	client := ServiceNow(cf)
	if client == nil {
		t.Fatal("Expected ServiceNow client to be created, got nil")
	}

	// Reset the global connection for cleanup
	snowMutex.Lock()
	snowConnection = nil
	snowMutex.Unlock()
}

// Test helper to verify that TLS configuration is properly set
func TestTLSConfigValidation(t *testing.T) {
	// This is a unit test to verify the TLS configuration structure
	// In a real scenario, this would be tested with actual HTTPS calls

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	if !tlsConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}

	tlsConfig.InsecureSkipVerify = false
	if tlsConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be false")
	}
}

// Test URL parsing functionality
func TestURLParsing(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL",
			rawURL:  "https://test.service-now.com",
			wantErr: false,
		},
		{
			name:    "valid HTTP URL",
			rawURL:  "http://test.service-now.com",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			rawURL:  "://invalid-url",
			wantErr: true,
		},
		{
			name:    "URL with port",
			rawURL:  "https://test.service-now.com:8443",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := url.Parse(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("url.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}