package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

func TestExpandNestedConfigAndLoopLimit(t *testing.T) {
	c := New()
	Set(c, "a", "${config:b}")
	Set(c, "b", "${config:c}")
	Set(c, "c", "end")
	Set(c, "self", "x${config:self}")
	Set(c, "path", "${config:a}/${config:c}")

	tests := []struct {
		name    string
		input   string
		options []ExpandOption
		want    string
	}{
		{name: "three levels by default", input: "${config:a}", want: "end"},
		{name: "mixed text and references", input: "${config:path}", want: "end/end"},
		{name: "loop reaching exactly the depth", input: "${config:a}", options: []ExpandOption{Loop(3)}, want: "end"},
		{name: "loop one short leaves value unexpanded", input: "${config:a}", options: []ExpandOption{Loop(2)}, want: "${config:c}"},
		{name: "loop of one is no nesting", input: "${config:a}", options: []ExpandOption{Loop(1)}, want: "${config:b}"},
		{name: "loop of zero is no nesting", input: "${config:a}", options: []ExpandOption{Loop(0)}, want: "${config:b}"},
		{name: "negative loop is no nesting, not empty", input: "${config:a}", options: []ExpandOption{Loop(-1)}, want: "${config:b}"},
		{name: "plain string with negative loop", input: "no refs", options: []ExpandOption{Loop(-5)}, want: "no refs"},
		{name: "cycle terminates", input: "${config:self}", want: "xx${config:self}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Expand[string](c, test.input, test.options...); got != test.want {
				t.Errorf("Expand(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestExpandEscape(t *testing.T) {
	c := New()
	Set(c, "literal", "$${unchanged}")
	Set(c, "other", "plain")
	tests := []struct {
		input string
		want  string
	}{
		{input: "$${config:other}", want: "${config:other}"},
		{input: "a $${x} b ${config:other}", want: "a ${x} b plain"},
		{input: "${config:literal}", want: "${unchanged}"},
		{input: "$${", want: "${"},
		{input: "cost $5 and $", want: "cost $5 and $"},
		{input: "${config:other", want: "${config:other"},
		{input: `${unset\}name}`, want: ""},
	}
	for _, test := range tests {
		if got := Expand[string](c, test.input); got != test.want {
			t.Errorf("Expand(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestExpandVerbatimSources(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(file, []byte("file ${config:secret}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CORDIAL_TEST_INJECT", "env ${config:secret}")

	c := New()
	Set(c, "secret", "TOPSECRET")
	Set(c, "viaconfig", "${config:secret}")

	table := map[string]string{
		"user":      "${config:secret}",
		"file":      "${file:" + file + "}",
		"dotted.id": "${env:CORDIAL_TEST_INJECT}",
	}

	tests := []struct {
		name    string
		input   string
		options []ExpandOption
		want    string
	}{
		{name: "lookup values are not expanded", input: "${user}", options: []ExpandOption{LookupTable(table)}, want: "${config:secret}"},
		{name: "lookup value naming a file", input: "[${file}]", options: []ExpandOption{LookupTable(table)}, want: "[${file:" + file + "}]"},
		{name: "lookup table beats a dotted config key", input: "${dotted.id}", options: []ExpandOption{LookupTable(table)}, want: "${env:CORDIAL_TEST_INJECT}"},
		{name: "environment values are not expanded", input: "${env:CORDIAL_TEST_INJECT}", want: "env ${config:secret}"},
		{name: "plain name environment values are not expanded", input: "${CORDIAL_TEST_INJECT}", want: "env ${config:secret}"},
		{name: "file content is not expanded", input: "${file:" + file + "}", want: "file ${config:secret}"},
		{name: "file path content is not expanded", input: "${" + file + "}", want: "file ${config:secret}"},
		{name: "config key values are expanded", input: "${config:viaconfig}", want: "TOPSECRET"},
		{name: "unmatched plain name falls back to the environment", input: "${CORDIAL_TEST_INJECT}", options: []ExpandOption{LookupTable(table)}, want: "env ${config:secret}"},
		{name: "first table wins", input: "${a}", options: []ExpandOption{LookupTable(map[string]string{"a": "1"}, map[string]string{"a": "2"})}, want: "1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Expand[string](c, test.input, test.options...); got != test.want {
				t.Errorf("Expand(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestExpandPrefixResults(t *testing.T) {
	c := New()
	Set(c, "name", "gw1")
	fn := func(_ map[string]any, name string, _ bool, _ ...any) (string, error) {
		return "[" + strings.TrimPrefix(name, "my:") + "|${config:name}]", nil
	}

	if got, want := Expand[string](c, "${my:x}", Prefix("my", fn)), "[x|gw1]"; got != want {
		t.Errorf("Prefix result = %q, want nested expansion %q", got, want)
	}
	if got, want := Expand[string](c, "${my:x}", PrefixVerbatim("my", fn)), "[x|${config:name}]"; got != want {
		t.Errorf("PrefixVerbatim result = %q, want %q", got, want)
	}
	if got := Expand[string](c, "${my:x}", Prefix("my", fn), Loop(1)); got != "[x|${config:name}]" {
		t.Errorf("Prefix result with Loop(1) = %q, want no nesting", got)
	}
	// an unregistered prefix expands to nothing
	if got := Expand[string](c, "a${nosuch:x}b"); got != "ab" {
		t.Errorf("unregistered prefix = %q, want %q", got, "ab")
	}
}

func TestExpandHomeFileForms(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.WriteFile(filepath.Join(home, "note.txt"), []byte(" hello \n"), 0600); err != nil {
		t.Fatal(err)
	}

	c := New()
	for _, input := range []string{"${~/note.txt}", "${file:~/note.txt}"} {
		if got := Expand[string](c, input); got != "hello" {
			t.Errorf("Expand(%q) = %q, want %q", input, got, "hello")
		}
	}
	if got := Expand[string](c, "${~/note.txt}", ExternalLookups(false)); got != "" {
		t.Errorf("Expand with external lookups off = %q, want empty", got)
	}
}

func TestExpandDefaultAndInitialValues(t *testing.T) {
	t.Setenv("CORDIAL_TEST_PORT", "8080")
	t.Setenv("CORDIAL_TEST_EMPTY", "")
	c := New()
	Set(c, "empty", "${env:CORDIAL_TEST_EMPTY}")

	if got := Get[string](c, "unset", InitialValue("${env:CORDIAL_TEST_PORT}")); got != "8080" {
		t.Errorf("InitialValue string = %q, want expanded", got)
	}
	if got := Get[int](c, "unset", InitialValue("${env:CORDIAL_TEST_PORT}")); got != 8080 {
		t.Errorf("InitialValue int = %d, want 8080", got)
	}
	if got := Get[[]string](c, "unset", InitialValue([]string{"${env:CORDIAL_TEST_PORT}", "x"})); len(got) != 2 || got[0] != "8080" {
		t.Errorf("InitialValue []string = %v, want expanded", got)
	}
	if got := Get[string](c, "unset", InitialValue("${env:CORDIAL_TEST_PORT}"), NoExpand()); got != "${env:CORDIAL_TEST_PORT}" {
		t.Errorf("InitialValue with NoExpand = %q, want as-is", got)
	}
	if got := Get[string](c, "empty", DefaultValue("${env:CORDIAL_TEST_PORT}")); got != "${env:CORDIAL_TEST_PORT}" {
		t.Errorf("DefaultValue = %q, want used as-is", got)
	}
	if got := Get[uint16](c, "unset", DefaultValue(uint16(1234))); got != 1234 {
		t.Errorf("DefaultValue uint16 = %d, want 1234", got)
	}
}

func TestExpandNoDecodeBytesAndSecret(t *testing.T) {
	input := "${enc:/nonexistent/keyfile.aes:+encs+00112233445566778899AABBCCDDEEFF}"
	c := New()
	if got := Expand[[]byte](c, input, NoDecode(true)); string(got) != input {
		t.Errorf("Expand[[]byte] NoDecode = %q, want original", got)
	}
	Set(c, "pw", input)
	if got := Get[Secret](c, "pw", NoDecode(true)); string(got) != input {
		t.Errorf("Get[Secret] NoDecode = %q, want original", got)
	}
	if got := Get[[]byte](c, "pw", NoDecode(true)); string(got) != input {
		t.Errorf("Get[[]byte] NoDecode = %q, want original", got)
	}
}

func TestExpandDecodedValueIsNotReExpanded(t *testing.T) {
	values := NewKeyValues()
	defer values.Destroy()
	ciphertext, err := values.EncodeString("pa${config:other}ss")
	if err != nil {
		t.Fatal(err)
	}
	keyfile := filepath.Join(t.TempDir(), "keyfile.aes")
	if err := os.WriteFile(keyfile, []byte(values.String()), 0600); err != nil {
		t.Fatal(err)
	}
	c := New()
	Set(c, "other", "XX")
	Set(c, "cipher", "+encs+"+ciphertext)
	t.Setenv("CORDIAL_TEST_CIPHER", ciphertext)
	for _, input := range []string{
		"${enc:" + keyfile + ":+encs+" + ciphertext + "}",
		"${enc:" + keyfile + ":config:cipher}",
		"${enc:" + keyfile + ":env:CORDIAL_TEST_CIPHER}",
		"${enc:" + keyfile + ":CORDIAL_TEST_CIPHER}",
		"${enc:/nonexistent/keyfile|" + keyfile + ":+encs+" + ciphertext + "}",
	} {
		if got := Expand[string](c, input); got != "pa${config:other}ss" {
			t.Errorf("Expand(%q) = %q, want decoded value as-is", input, got)
		}
	}
}

func TestExpandCustomTablesAndPrefixAreLocal(t *testing.T) {
	fn := func(_ map[string]any, _ string, _ bool, _ ...any) (string, error) { return "custom", nil }
	a := New(SetDefaultExpandOptions(Prefix("localprefix", fn)))
	b := New()

	if got := Expand[string](a, "${localprefix:x}"); got != "custom" {
		t.Errorf("config with prefix = %q, want custom", got)
	}
	if got := Expand[string](b, "${localprefix:x}"); got != "" {
		t.Errorf("fresh config sees another config's prefix: %q", got)
	}

	// a per-call prefix must not be visible afterwards, and replacing a
	// built-in must not change the built-in
	file := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(file, []byte("real"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := Expand[string](b, "${percall:x}", Prefix("percall", fn)); got != "custom" {
		t.Errorf("per-call prefix = %q, want custom", got)
	}
	if got := Expand[string](b, "${percall:x}"); got != "" {
		t.Errorf("per-call prefix leaked: %q", got)
	}
	if got := Expand[string](b, "${file:"+file+"}", Prefix("file", fn)); got != "custom" {
		t.Errorf("replaced file prefix = %q, want custom", got)
	}
	if got := Expand[string](New(), "${file:"+file+"}"); got != "real" {
		t.Errorf("built-in file prefix changed by earlier call: %q", got)
	}
	if got := Expand[string](New(), "${expr:1+2}"); got != "" {
		t.Errorf("expressions enabled for another call: %q", got)
	}
	if got := Expand[string](New(), "${expr:1+2}", Expressions(true)); got != "3" {
		t.Errorf("expression = %q, want 3", got)
	}
}

func TestExpandConcurrentPrefixOptions(t *testing.T) {
	const workers = 8
	var wg sync.WaitGroup
	for i := range workers {
		cf := New()
		Set(cf, "id", strconv.Itoa(i))
		fn := func(_ map[string]any, _ string, _ bool, _ ...any) (string, error) {
			return "p" + strconv.Itoa(i), nil
		}
		wg.Go(func() {
			for range 200 {
				got := Expand[string](cf, "${pfx:x}-${config:id}", Prefix("pfx", fn), Expressions(true))
				if want := "p" + strconv.Itoa(i) + "-" + strconv.Itoa(i); got != want {
					t.Errorf("worker %d: got %q, want %q", i, got, want)
					return
				}
				cf.SetDefaultExpandOptions(Prefix("other", fn))
				_ = cf.DefaultExpandOptions()
			}
		})
	}
	wg.Wait()
}
