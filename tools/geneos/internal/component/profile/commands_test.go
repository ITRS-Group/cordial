package profile

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateProfileListFormat(t *testing.T) {
	for _, format := range []string{"column", "csv", "json", "pretty", "table", "toolkit"} {
		if err := validateProfileListFormat(format); err != nil {
			t.Errorf("validateProfileListFormat(%q) error = %v", format, err)
		}
	}
	if err := validateProfileListFormat("xml"); err == nil {
		t.Fatal("validateProfileListFormat accepted unsupported xml format")
	}
}

func TestFormatProfileListCSV(t *testing.T) {
	profiles := map[string]any{
		"demo": map[string]any{"_short": "Demonstration profile"},
	}
	var output bytes.Buffer
	if err := formatProfileList(&output, profiles, "/etc/geneos/profiles.yaml", "csv"); err != nil {
		t.Fatalf("formatProfileList() error = %v", err)
	}
	for _, want := range []string{"Profile", "Description", "From", "demo", "Demonstration profile", "/etc/geneos/profiles.yaml"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("formatProfileList() output %q does not contain %q", output.String(), want)
		}
	}
}

func TestFormatProfileListJSON(t *testing.T) {
	profiles := map[string]any{
		"demo": map[string]any{"_short": "Demonstration profile"},
	}
	var output bytes.Buffer
	if err := formatProfileList(&output, profiles, "/etc/geneos/profiles.yaml", "json"); err != nil {
		t.Fatalf("formatProfileList() error = %v", err)
	}
	for _, want := range []string{`"profile":"demo"`, `"description":"Demonstration profile"`, `"from":"/etc/geneos/profiles.yaml"`} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("formatProfileList() output %q does not contain %q", output.String(), want)
		}
	}
}
