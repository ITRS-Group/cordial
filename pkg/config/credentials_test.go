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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCredentials(t *testing.T) {
	// Test Credentials struct
	creds := Credentials{
		Domain:       "example.com",
		Username:     "testuser",
		Password:     "testpass",
		ClientID:     "client123",
		ClientSecret: "secret456",
		Token:        "token789",
		Renewal:      "renewal123",
	}

	if creds.Domain != "example.com" {
		t.Errorf("Domain = %q, want 'example.com'", creds.Domain)
	}

	if creds.Username != "testuser" {
		t.Errorf("Username = %q, want 'testuser'", creds.Username)
	}

	if creds.Password != "testpass" {
		t.Errorf("Password = %q, want 'testpass'", creds.Password)
	}
}

func TestFindCreds(t *testing.T) {
	config := New(KeyDelimiter("::"))

	// Set up test credentials
	config.Set("credentials::example.com::username", "user1")
	config.Set("credentials::example.com::password", "pass1")
	config.Set("credentials::sub.example.com::username", "user2")
	config.Set("credentials::sub.example.com::password", "pass2")
	config.Set("credentials::other.com::username", "user3")

	tests := []struct {
		name     string
		path     string
		wantUser string
		wantNil  bool
	}{
		{
			name:     "exact match",
			path:     "example.com",
			wantUser: "user1",
		},
		{
			name:     "subdomain match",
			path:     "sub.example.com",
			wantUser: "user2",
		},
		{
			name:     "longest match",
			path:     "test.sub.example.com",
			wantUser: "user2", // Should match sub.example.com, not example.com
		},
		{
			name:    "no match",
			path:    "nomatch.com",
			wantNil: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := config.FindCreds(tt.path)

			if tt.wantNil {
				if creds != nil {
					t.Error("FindCreds() should return nil")
				}
				return
			}

			if creds == nil {
				t.Fatal("FindCreds() returned nil, expected credentials")
			}

			if got := creds.GetString("username"); got != tt.wantUser {
				t.Errorf("FindCreds() username = %q, want %q", got, tt.wantUser)
			}
		})
	}
}

func TestFindCredsGlobal(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test config file with credentials
	configContent := `{
		"credentials": {
			"global.example.com": {
				"username": "globaluser",
				"password": "globalpass"
			}
		}
	}`

	configFile := filepath.Join(tempDir, "creds.json")
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test global FindCreds function
	creds := FindCreds("global.example.com", SetConfigFile(configFile))
	if creds == nil {
		t.Fatal("FindCreds() global returned nil")
	}

	if got := creds.GetString("username"); got != "globaluser" {
		t.Errorf("Global FindCreds() username = %q, want 'globaluser'", got)
	}
}

func TestAddCreds(t *testing.T) {
	tempDir := t.TempDir()

	creds := Credentials{
		Domain:   "test.example.com",
		Username: "newuser",
		Password: "newpass",
		Token:    "newtoken",
	}

	err := AddCreds(creds, AddDirs(tempDir), SetAppName("testapp"))
	if err != nil {
		t.Fatalf("AddCreds() failed: %v", err)
	}

	// Verify credentials were saved by trying to find them
	foundCreds := FindCreds("test.example.com", AddDirs(tempDir), SetAppName("testapp"))
	if foundCreds == nil {
		t.Fatal("AddCreds() did not save credentials properly")
	}

	if got := foundCreds.GetString("username"); got != "newuser" {
		t.Errorf("Added credentials username = %q, want 'newuser'", got)
	}

	if got := foundCreds.GetString("password"); got != "newpass" {
		t.Errorf("Added credentials password = %q, want 'newpass'", got)
	}

	if got := foundCreds.GetString("token"); got != "newtoken" {
		t.Errorf("Added credentials token = %q, want 'newtoken'", got)
	}
}

func TestDeleteCreds(t *testing.T) {
	tempDir := t.TempDir()

	// First add some credentials
	creds := Credentials{
		Domain:   "delete.example.com",
		Username: "deleteuser",
		Password: "deletepass",
	}

	err := AddCreds(creds, AddDirs(tempDir), SetAppName("testapp"))
	if err != nil {
		t.Fatalf("AddCreds() setup failed: %v", err)
	}

	// Verify they exist
	foundCreds := FindCreds("delete.example.com", AddDirs(tempDir), SetAppName("testapp"))
	if foundCreds == nil {
		t.Fatal("Setup: credentials not found after adding")
	}

	// Delete them
	err = DeleteCreds("delete.example.com", AddDirs(tempDir), SetAppName("testapp"))
	if err != nil {
		t.Fatalf("DeleteCreds() failed: %v", err)
	}

	// Verify they're gone
	foundCreds = FindCreds("delete.example.com", AddDirs(tempDir), SetAppName("testapp"))
	if foundCreds != nil {
		t.Error("DeleteCreds() did not remove credentials")
	}
}

func TestDeleteAllCreds(t *testing.T) {
	tempDir := t.TempDir()

	// Add multiple credentials
	creds1 := Credentials{
		Domain:   "all1.example.com",
		Username: "user1",
	}
	creds2 := Credentials{
		Domain:   "all2.example.com",
		Username: "user2",
	}

	options := []FileOptions{AddDirs(tempDir), SetAppName("testapp")}

	err := AddCreds(creds1, options...)
	if err != nil {
		t.Fatalf("AddCreds() 1 failed: %v", err)
	}

	err = AddCreds(creds2, options...)
	if err != nil {
		t.Fatalf("AddCreds() 2 failed: %v", err)
	}

	// Verify both exist
	if FindCreds("all1.example.com", options...) == nil {
		t.Fatal("Setup: first credentials not found")
	}
	if FindCreds("all2.example.com", options...) == nil {
		t.Fatal("Setup: second credentials not found")
	}

	// Delete all
	err = DeleteAllCreds(options...)
	if err != nil {
		t.Fatalf("DeleteAllCreds() failed: %v", err)
	}

	// Verify both are gone
	if FindCreds("all1.example.com", options...) != nil {
		t.Error("DeleteAllCreds() did not remove first credentials")
	}
	if FindCreds("all2.example.com", options...) != nil {
		t.Error("DeleteAllCreds() did not remove second credentials")
	}
}

func TestCredentialsWithNilConfig(t *testing.T) {
	var config *Config = nil

	// FindCreds should handle nil config gracefully
	creds := config.FindCreds("any.domain.com")
	if creds != nil {
		t.Error("FindCreds() with nil config should return nil")
	}
}

func TestCredentialsEmptyDomain(t *testing.T) {
	tempDir := t.TempDir()

	// Test adding credentials with empty domain
	creds := Credentials{
		Domain:   "",
		Username: "emptyuser",
	}

	err := AddCreds(creds, AddDirs(tempDir))
	// This might fail or succeed depending on implementation
	// The test is mainly to ensure it doesn't panic
	if err != nil {
		t.Logf("AddCreds() with empty domain failed as expected: %v", err)
	}
}

func TestCredentialsComplexDomains(t *testing.T) {
	config := New(KeyDelimiter("::"))

	// Set up credentials for various domain patterns
	config.Set("credentials::api.service.company.com::username", "api_user")
	config.Set("credentials::service.company.com::username", "service_user")
	config.Set("credentials::company.com::username", "company_user")
	config.Set("credentials::localhost:8080::username", "local_user")
	config.Set("credentials::192.168.1.100::username", "ip_user")

	tests := []struct {
		name     string
		domain   string
		wantUser string
	}{
		{
			name:     "full API domain",
			domain:   "api.service.company.com",
			wantUser: "api_user",
		},
		{
			name:     "service domain",
			domain:   "service.company.com",
			wantUser: "service_user",
		},
		{
			name:     "company domain",
			domain:   "company.com",
			wantUser: "company_user",
		},
		{
			name:     "localhost with port",
			domain:   "localhost:8080",
			wantUser: "local_user",
		},
		{
			name:     "IP address",
			domain:   "192.168.1.100",
			wantUser: "ip_user",
		},
		{
			name:     "subdomain of API",
			domain:   "v1.api.service.company.com",
			wantUser: "api_user", // Should match longest prefix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := config.FindCreds(tt.domain)
			if creds == nil {
				t.Fatalf("FindCreds(%q) returned nil", tt.domain)
			}

			if got := creds.GetString("username"); got != tt.wantUser {
				t.Errorf("FindCreds(%q) username = %q, want %q", tt.domain, got, tt.wantUser)
			}
		})
	}
}

func TestCredentialsPartialMatches(t *testing.T) {
	config := New(KeyDelimiter("::"))

	// Set up some test credentials
	config.Set("credentials::example.com::username", "example_user")
	config.Set("credentials::test.example.com::username", "test_user")

	// Test partial domain matching
	creds := config.FindCreds("sub.test.example.com")
	if creds == nil {
		t.Fatal("FindCreds() should find partial match")
	}

	// Should match the longest available match (test.example.com)
	if got := creds.GetString("username"); got != "test_user" {
		t.Errorf("FindCreds() should match longest prefix, got %q, want 'test_user'", got)
	}
}

func TestCredentialsWithAllFields(t *testing.T) {
	tempDir := t.TempDir()

	// Test with all credential fields populated
	fullCreds := Credentials{
		Domain:       "full.example.com",
		Username:     "fulluser",
		Password:     "fullpass",
		ClientID:     "full_client_id",
		ClientSecret: "full_client_secret",
		Token:        "full_token",
		Renewal:      "full_renewal",
	}

	err := AddCreds(fullCreds, AddDirs(tempDir))
	if err != nil {
		t.Fatalf("AddCreds() with full credentials failed: %v", err)
	}

	// Retrieve and verify all fields
	retrieved := FindCreds("full.example.com", AddDirs(tempDir))
	if retrieved == nil {
		t.Fatal("FindCreds() returned nil for full credentials")
	}

	if got := retrieved.GetString("username"); got != "fulluser" {
		t.Errorf("username = %q, want 'fulluser'", got)
	}
	if got := retrieved.GetString("password"); got != "fullpass" {
		t.Errorf("password = %q, want 'fullpass'", got)
	}
	if got := retrieved.GetString("client_id"); got != "full_client_id" {
		t.Errorf("client_id = %q, want 'full_client_id'", got)
	}
	if got := retrieved.GetString("client_secret"); got != "full_client_secret" {
		t.Errorf("client_secret = %q, want 'full_client_secret'", got)
	}
	if got := retrieved.GetString("token"); got != "full_token" {
		t.Errorf("token = %q, want 'full_token'", got)
	}
	if got := retrieved.GetString("renewal"); got != "full_renewal" {
		t.Errorf("renewal = %q, want 'full_renewal'", got)
	}
}
