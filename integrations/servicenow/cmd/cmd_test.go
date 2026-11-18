package cmd

import (
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
)

func TestAPIKeyValid(t *testing.T) {
	tests := []struct {
		expected, key string
		want          bool
	}{
		{"secret", "secret", true},
		{"secret", "Secret", false},
		{"secret", "secre", false},
		{"secret", "secrets", false},
		{"secret", "", false},
		{"", "", false},
		{"", "anything", false},
	}
	for _, test := range tests {
		if got := apiKeyValid(test.expected, test.key); got != test.want {
			t.Errorf("apiKeyValid(%q, %q) = %t, want %t", test.expected, test.key, got, test.want)
		}
	}
}

func TestExecnameIsTheExecutableName(t *testing.T) {
	if execname != cordial.ExecutableName() || strings.ContainsAny(execname, `/\`) || strings.HasSuffix(strings.ToLower(execname), ".exe") {
		t.Errorf("execname = %q, want %q", execname, cordial.ExecutableName())
	}
}
