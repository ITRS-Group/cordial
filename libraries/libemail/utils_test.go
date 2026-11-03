package main

import (
	stdmail "net/mail"
	"reflect"
	"strings"
	"testing"
	"time"

	mail "github.com/go-mail/mail/v2"
)

func TestSetupMailBuildsFilteredAddressHeadersAndCleansConfig(t *testing.T) {
	conf := EMailConfig{
		"_TO":           "alice@example.com, bob@example.com, +15551234567",
		"_TO_NAME":      "Alice, Bob, SMS",
		"_TO_INFO_TYPE": "email, E-MAIL, phone",
		"_CC":           "ops@example.com",
		"_CC_NAME":      "Operations",
		"_FROM":         "geneos@example.com",
		"_FROM_NAME":    "Geneos",
		"_UNRELATED":    "kept",
	}

	message, err := setupMail(conf)
	if err != nil {
		t.Fatalf("setupMail() error = %v", err)
	}
	to, err := stdmail.ParseAddressList(strings.Join(message.GetHeader("To"), ","))
	if err != nil {
		t.Fatalf("parse To header: %v", err)
	}
	if len(to) != 2 || to[0].Address != "alice@example.com" || to[0].Name != "Alice" || to[1].Address != "bob@example.com" || to[1].Name != "Bob" {
		t.Errorf("To recipients = %#v, want Alice and Bob email addresses", to)
	}
	cc, err := stdmail.ParseAddressList(strings.Join(message.GetHeader("Cc"), ","))
	if err != nil {
		t.Fatalf("parse Cc header: %v", err)
	}
	if len(cc) != 1 || cc[0].Address != "ops@example.com" || cc[0].Name != "Operations" {
		t.Errorf("Cc recipients = %#v, want Operations <ops@example.com>", cc)
	}
	from, err := stdmail.ParseAddress(strings.Join(message.GetHeader("From"), ","))
	if err != nil {
		t.Fatalf("parse From header: %v", err)
	}
	if from.Address != "geneos@example.com" || from.Name != "Geneos" {
		t.Errorf("From = %#v, want Geneos <geneos@example.com>", from)
	}
	for _, key := range []string{"_TO", "_TO_NAME", "_TO_INFO_TYPE", "_CC", "_CC_NAME", "_FROM", "_FROM_NAME"} {
		if _, ok := conf[key]; ok {
			t.Errorf("setupMail() retained private header config %q", key)
		}
	}
	if conf["_UNRELATED"] != "kept" {
		t.Errorf("setupMail() removed unrelated config: %#v", conf)
	}
}

func TestSetupMailDebugPreservesHeaderConfig(t *testing.T) {
	conf := EMailConfig{"_DEBUG": "TrUe", "_TO": "ops@example.com"}
	if _, err := setupMail(conf); err != nil {
		t.Fatalf("setupMail() error = %v", err)
	}
	if conf["_TO"] != "ops@example.com" {
		t.Errorf("debug setup removed recipient config: %#v", conf)
	}
}

func TestAddAddressesRejectsMismatchedMetadata(t *testing.T) {
	tests := []struct {
		name string
		conf EMailConfig
	}{
		{name: "names", conf: EMailConfig{"_TO": "a@example.com,b@example.com", "_TO_NAME": "A"}},
		{name: "info types", conf: EMailConfig{"_TO": "a@example.com,b@example.com", "_TO_INFO_TYPE": "email"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := addAddresses(mail.NewMessage(), test.conf, "To"); err == nil {
				t.Fatal("addAddresses() accepted mismatched address metadata")
			}
		})
	}
}

func TestAddAddressesLeavesHeaderEmptyWhenAllTypesAreFiltered(t *testing.T) {
	message := mail.NewMessage()
	conf := EMailConfig{
		"_TO":           "pager@example.com",
		"_TO_INFO_TYPE": "pager",
	}
	if err := addAddresses(message, conf, "To"); err != nil {
		t.Fatalf("addAddresses() error = %v", err)
	}
	if got := message.GetHeader("To"); len(got) != 0 {
		t.Errorf("To header = %#v, want no email addresses", got)
	}
}

func TestReplArgsSubstitutesKnownAndMissingValues(t *testing.T) {
	got := replArgs("Hello %(_NAME), incident %(ID): %(_UNKNOWN)", EMailConfig{
		"_NAME": "Operations",
		"ID":    "INC001",
	})
	want := "Hello Operations, incident INC001: "
	if got != want {
		t.Errorf("replArgs() = %q, want %q", got, want)
	}
}

func TestDialServerUsesConfiguredSMTPSettings(t *testing.T) {
	dialer, err := dialServer(EMailConfig{
		"_SMTP_SERVER":  "smtp.example.test",
		"_SMTP_PORT":    "2525",
		"_SMTP_TIMEOUT": "7",
		"_SMTP_TLS":     "none",
		"_DEBUG":        "true",
	})
	if err != nil {
		t.Fatalf("dialServer() error = %v", err)
	}
	if dialer.Host != "smtp.example.test" || dialer.Port != 2525 || dialer.Timeout != 7*time.Second {
		t.Errorf("dialer settings = (%q, %d, %s)", dialer.Host, dialer.Port, dialer.Timeout)
	}
	if dialer.StartTLSPolicy != mail.NoStartTLS {
		t.Errorf("StartTLSPolicy = %v, want NoStartTLS", dialer.StartTLSPolicy)
	}
}

func TestSplitCommaTrimSpaceAndDefaults(t *testing.T) {
	if got := splitCommaTrimSpace("  a@example.com, b@example.com ,, "); !reflect.DeepEqual(got, []string{"a@example.com", "b@example.com", "", ""}) {
		t.Errorf("splitCommaTrimSpace() = %#v", got)
	}
	if got := splitCommaTrimSpace(""); got == nil || len(got) != 0 {
		t.Errorf("splitCommaTrimSpace(empty) = %#v, want empty non-nil slice", got)
	}
	if got := getWithDefault("missing", EMailConfig{}, "default"); got != "default" {
		t.Errorf("getWithDefault() = %q, want default", got)
	}
	if got := getWithDefaultInt("invalid", EMailConfig{"invalid": "not-a-number"}, 25); got != 0 {
		t.Errorf("getWithDefaultInt(invalid) = %d, want 0", got)
	}
}
