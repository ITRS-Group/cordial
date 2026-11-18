package ims

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

// hostile values must come back exactly as they were sent, whatever the
// expansion does with substituted values
func TestExpandLiteralDoesNotExpandValues(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretFile, []byte("file-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("IMS_TEST_SECRET", "env-secret")

	cf := config.New()
	config.Set(cf, "secret", "config-secret")

	values := []string{
		"${env:IMS_TEST_SECRET}",
		"${IMS_TEST_SECRET}",
		"${config:secret}",
		"${secret}",
		"${file:" + secretFile + "}",
		"${" + secretFile + "}",
		"  ${env:IMS_TEST_SECRET}  ",
		"$",
		"{env:IMS_TEST_SECRET}",
		"${${env:IMS_TEST_SECRET}}",
	}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			// the whole value, and the value joined with other
			// substitutions, as expansion repeats on the result
			got := ExpandLiteral(cf, "[${__incident_subject}]", false, map[string]string{"__incident_subject": value})
			if want := "[" + value + "]"; got != want {
				t.Errorf("ExpandLiteral() = %q, want %q", got, want)
			}
		})
	}

	t.Run("split across substitutions", func(t *testing.T) {
		got := ExpandLiteral(cf, "${a}${b}", true, map[string]string{"a": "$", "b": "{env:IMS_TEST_SECRET}"})
		if got != "${env:IMS_TEST_SECRET}" {
			t.Errorf("ExpandLiteral() = %q", got)
		}
	})

	t.Run("later tables are protected too", func(t *testing.T) {
		got := ExpandLiteral(cf, "${x} ${y}", true,
			map[string]string{"x": "1"}, map[string]string{"y": "${env:IMS_TEST_SECRET}", "x": "ignored"})
		if got != "1 ${env:IMS_TEST_SECRET}" {
			t.Errorf("ExpandLiteral() = %q", got)
		}
	})

	t.Run("template is still expanded", func(t *testing.T) {
		got := ExpandLiteral(cf, "${config:secret}/${v}/${env:IMS_TEST_SECRET}", true, map[string]string{"v": "x"})
		if got != "config-secret/x/env-secret" {
			t.Errorf("ExpandLiteral() = %q", got)
		}
	})
}

func TestExpandLiteralTrim(t *testing.T) {
	cf := config.New()
	tests := []struct {
		trim bool
		want string
	}{
		{true, "[x]"},
		{false, "[ x ]"},
	}
	for _, test := range tests {
		if got := ExpandLiteral(cf, "[${v}]", test.trim, map[string]string{"v": " x "}); got != test.want {
			t.Errorf("trim=%t: ExpandLiteral() = %q, want %q", test.trim, got, test.want)
		}
	}
}

func TestLiteralsEncode(t *testing.T) {
	cf := config.New()
	t.Setenv("IMS_TEST_SECRET", "env-secret")

	tests := []string{
		`plain`,
		`a "quoted" value`,
		`back\slash`,
		"new\nline",
		`"},{"field":"x`,
		`${env:IMS_TEST_SECRET}`,
	}
	for _, value := range tests {
		l := NewLiterals(true, JSONString, map[string]string{"v": value})
		doc := l.Restore(config.Expand[string](cf, `{"value":"${v}"}`, l.Option()))
		var got struct{ Value string }
		if err := json.Unmarshal([]byte(doc), &got); err != nil {
			t.Errorf("%q: document %s is not valid JSON: %v", value, doc, err)
			continue
		}
		if got.Value != value {
			t.Errorf("value = %q, want %q (document %s)", got.Value, value, doc)
		}
	}
}

func TestLiteralsPlaceholdersAreNotPredictable(t *testing.T) {
	a := NewLiterals(false, nil, map[string]string{"v": "1"})
	b := NewLiterals(false, nil, map[string]string{"v": "1"})
	if a.tables[0]["v"] == b.tables[0]["v"] || strings.ContainsAny(a.tables[0]["v"], "${}") {
		t.Errorf("placeholders %q and %q must differ and be free of expansion syntax", a.tables[0]["v"], b.tables[0]["v"])
	}
}
