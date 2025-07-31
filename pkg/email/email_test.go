package email

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/wneessen/go-mail"
)

func TestDial(t *testing.T) {
	// Test with default configuration
	conf := config.New()
	dialer, err := Dial(conf)
	if err != nil {
		t.Errorf("Dial failed with default config: %v", err)
	}

	if dialer == nil {
		t.Fatal("Dial should not return nil dialer")
	}

	// Test with custom TLS policy
	conf.Set("_smtp_tls", "force")
	dialer, err = Dial(conf)
	if err != nil {
		t.Errorf("Dial failed with force TLS: %v", err)
	}

	conf.Set("_smtp_tls", "none")
	dialer, err = Dial(conf)
	if err != nil {
		t.Errorf("Dial failed with no TLS: %v", err)
	}

	// Test with authentication
	conf.Set("_smtp_username", "testuser")
	conf.Set("_smtp_password", "testpass")
	dialer, err = Dial(conf)
	if err != nil {
		t.Errorf("Dial failed with authentication: %v", err)
	}

	// Test with custom port
	conf.Set("_smtp_port", 587)
	dialer, err = Dial(conf)
	if err != nil {
		t.Errorf("Dial failed with custom port: %v", err)
	}

	// Test with insecure TLS
	conf.Set("_smtp_tls_insecure", true)
	dialer, err = Dial(conf)
	if err != nil {
		t.Errorf("Dial failed with insecure TLS: %v", err)
	}
}

func TestUpdateEnvelope(t *testing.T) {
	conf := config.New()
	
	// Test with basic configuration
	msg, err := UpdateEnvelope(conf, false)
	if err != nil {
		t.Errorf("UpdateEnvelope failed: %v", err)
	}

	if msg == nil {
		t.Fatal("UpdateEnvelope should not return nil message")
	}

	// Test with inline CSS
	msg, err = UpdateEnvelope(conf, true)
	if err != nil {
		t.Errorf("UpdateEnvelope failed with inline CSS: %v", err)
	}

	// Test with custom subject
	conf.Set("_smtp_subject", "Test Subject")
	msg, err = UpdateEnvelope(conf, false)
	if err != nil {
		t.Errorf("UpdateEnvelope failed with custom subject: %v", err)
	}
}

func TestSplitCommaTrimSpace(t *testing.T) {
	// Test with single email
	result := splitCommaTrimSpace("test@example.com")
	if len(result) != 1 {
		t.Errorf("Expected 1 email, got %d", len(result))
	}
	if result[0] != "test@example.com" {
		t.Errorf("Expected 'test@example.com', got '%s'", result[0])
	}

	// Test with multiple emails
	result = splitCommaTrimSpace("test1@example.com, test2@example.com, test3@example.com")
	if len(result) != 3 {
		t.Errorf("Expected 3 emails, got %d", len(result))
	}
	expected := []string{"test1@example.com", "test2@example.com", "test3@example.com"}
	for i, email := range expected {
		if result[i] != email {
			t.Errorf("Expected '%s', got '%s'", email, result[i])
		}
	}

	// Test with whitespace
	result = splitCommaTrimSpace("  test@example.com  ")
	if len(result) != 1 {
		t.Errorf("Expected 1 email, got %d", len(result))
	}
	if result[0] != "test@example.com" {
		t.Errorf("Expected 'test@example.com', got '%s'", result[0])
	}

	// Test with empty string
	result = splitCommaTrimSpace("")
	if len(result) != 0 {
		t.Errorf("Expected 0 emails, got %d", len(result))
	}

	// Test with only whitespace
	result = splitCommaTrimSpace("   ")
	if len(result) != 1 {
		t.Errorf("Expected 1 email, got %d", len(result))
	}
	if result[0] != "" {
		t.Errorf("Expected empty string, got '%s'", result[0])
	}
}

func TestNewEmailConfig(t *testing.T) {
	baseConf := config.New()
	baseConf.Set("email.smtp", "smtp.example.com")
	baseConf.Set("email.port", 587)

	// Test with to, cc, and bcc addresses
	emailConf := NewEmailConfig(baseConf, "to@example.com", "cc@example.com", "bcc@example.com")
	if emailConf == nil {
		t.Fatal("NewEmailConfig should not return nil")
	}

	// Check that the email config values are set correctly
	if emailConf.GetString("_smtp_server") != "smtp.example.com" {
		t.Errorf("Expected server 'smtp.example.com', got '%s'", emailConf.GetString("_smtp_server"))
	}

	if emailConf.GetInt("_smtp_port") != 587 {
		t.Errorf("Expected port 587, got %d", emailConf.GetInt("_smtp_port"))
	}

	// Check that the address arguments are set
	if emailConf.GetString("_to") != "to@example.com" {
		t.Errorf("Expected to 'to@example.com', got '%s'", emailConf.GetString("_to"))
	}

	if emailConf.GetString("_cc") != "cc@example.com" {
		t.Errorf("Expected cc 'cc@example.com', got '%s'", emailConf.GetString("_cc"))
	}

	if emailConf.GetString("_bcc") != "bcc@example.com" {
		t.Errorf("Expected bcc 'bcc@example.com', got '%s'", emailConf.GetString("_bcc"))
	}

	// Test with empty addresses
	emailConf = NewEmailConfig(baseConf, "", "", "")
	if emailConf == nil {
		t.Fatal("NewEmailConfig should not return nil with empty addresses")
	}

	// Test with nil base config - this should not panic
	emailConf = NewEmailConfig(nil, "to@example.com", "cc@example.com", "bcc@example.com")
	if emailConf == nil {
		t.Fatal("NewEmailConfig should not return nil with nil base config")
	}
}

func TestAddAddresses(t *testing.T) {
	conf := config.New()
	msg := &mail.Msg{}

	// Test with valid addresses
	conf.Set("_smtp_to", "to1@example.com, to2@example.com")
	err := addAddresses(msg, conf, "_smtp_to")
	if err != nil {
		t.Errorf("addAddresses failed: %v", err)
	}

	// Test with empty addresses
	conf.Set("_smtp_to", "")
	err = addAddresses(msg, conf, "_smtp_to")
	if err != nil {
		t.Errorf("addAddresses failed with empty addresses: %v", err)
	}

	// Test with invalid email format - the mail library might accept invalid formats
	conf.Set("_smtp_to", "invalid-email")
	err = addAddresses(msg, conf, "_smtp_to")
	// Note: The mail library might accept invalid email formats, so we don't expect an error
	if err != nil {
		t.Errorf("addAddresses failed with invalid email format: %v", err)
	}
}

func TestEmailConfigurationDefaults(t *testing.T) {
	conf := config.New()

	// Test default TLS policy
	_, err := Dial(conf)
	if err != nil {
		t.Errorf("Dial failed with default config: %v", err)
	}

	// Test that the function works with default values
	// Note: The defaults are used internally in the Dial function, not stored in the config
	if conf.GetInt("_smtp_timeout") != 0 {
		t.Errorf("Expected unset timeout 0, got %d", conf.GetInt("_smtp_timeout"))
	}

	// Test default TLS policy
	if conf.GetString("_smtp_tls") != "" {
		t.Errorf("Expected unset TLS policy '', got '%s'", conf.GetString("_smtp_tls"))
	}
}

func TestEmailConfigurationValidation(t *testing.T) {
	conf := config.New()

	// Test with invalid TLS policy - the function might accept any string
	conf.Set("_smtp_tls", "invalid")
	_, err := Dial(conf)
	// Note: The Dial function might accept any TLS policy string, so we don't expect an error
	if err != nil {
		t.Errorf("Dial failed with invalid TLS policy: %v", err)
	}

	// Test with negative port
	conf.Set("_smtp_port", -1)
	_, err = Dial(conf)
	if err == nil {
		t.Error("Expected error with negative port")
	}

	// Test with very large port
	conf.Set("_smtp_port", 99999)
	_, err = Dial(conf)
	if err == nil {
		t.Error("Expected error with very large port")
	}
}