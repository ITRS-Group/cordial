package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandConfigEnvironmentAndLookupTable(t *testing.T) {
	c := New()
	Set(c, "service.host", "  gateway.example  ")
	t.Setenv("CORDIAL_TEST_TOKEN", "  test-token  ")

	tests := []struct {
		name    string
		input   string
		options []ExpandOption
		want    string
	}{
		{name: "config reference", input: `${config:service.host}`, want: "gateway.example"},
		{name: "dotted config reference", input: `${service.host}`, want: "gateway.example"},
		{name: "environment reference", input: `${env:CORDIAL_TEST_TOKEN}`, want: "test-token"},
		{name: "lookup table", input: `${region}`, options: []ExpandOption{LookupTable(map[string]string{"region": "  eu-west  "})}, want: "eu-west"},
		{name: "literal dollar variable is not expanded", input: "$CORDIAL_TEST_TOKEN", want: "$CORDIAL_TEST_TOKEN"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Expand[string](c, test.input, test.options...); got != test.want {
				t.Errorf("Expand(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestExpandExternalFileLookupCanBeDisabled(t *testing.T) {
	file := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(file, []byte("secret-content\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	c := New()
	if got := Expand[string](c, "${"+file+"}"); got != "secret-content" {
		t.Fatalf("default file expansion = %q, want %q", got, "secret-content")
	}
	if got := Expand[string](c, "${"+file+"}", ExternalLookups(false)); got != "" {
		t.Errorf("disabled file expansion = %q, want empty result", got)
	}

	// TODO: decide whether external lookup failures need a distinguishable error result instead of an empty expansion.
}

func TestExpandEncryptedValueAndNoDecode(t *testing.T) {
	values := NewKeyValues()
	defer values.Destroy()
	ciphertext, err := values.EncodeString("highly-sensitive")
	if err != nil {
		t.Fatalf("EncodeString() error = %v", err)
	}
	keyfile := filepath.Join(t.TempDir(), "keyfile.aes")
	if err := os.WriteFile(keyfile, []byte(values.String()), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	input := "${enc:" + keyfile + ":+encs+" + ciphertext + "}"
	c := New()

	if got := Expand[string](c, input); got != "highly-sensitive" {
		t.Fatalf("encrypted expansion = %q, want plaintext", got)
	}
	if got := Expand[string](c, input, NoDecode(true)); got != input {
		t.Errorf("NoDecode expansion = %q, want original encrypted value", got)
	}
}
