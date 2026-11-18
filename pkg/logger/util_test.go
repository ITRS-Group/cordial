package logger

import (
	"reflect"
	"testing"
)

func TestRedact(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "password=secret", want: "password=[REDACTED]"},
		{input: "--api_key=abc123", want: "--api_key=[REDACTED]"},
		{input: "-ssl-private-key=/etc/geneos/client.pem", want: "-ssl-private-key=/etc/geneos/client.pem"},
		{input: "encoded +encs+secretvalue", want: "encoded [REDACTED]"},
		{input: "ordinary message", want: "ordinary message"},
		{input: "password='two words' next", want: "password=[REDACTED] next"},
		{input: `password="two words" next`, want: "password=[REDACTED] next"},
		{input: "password='unterminated words", want: "password=[REDACTED]"},
		{input: `password="unterminated words`, want: "password=[REDACTED]"},
	}
	for _, test := range tests {
		if got := Redact(test.input); got != test.want {
			t.Errorf("Redact(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestRedactSlice(t *testing.T) {
	in := []string{"token=abc", "safe"}
	got := RedactSlice(in)
	want := []string{"token=[REDACTED]", "safe"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RedactSlice() = %#v, want %#v", got, want)
	}
	if in[0] != "token=abc" {
		t.Errorf("RedactSlice() modified its argument: %v", in)
	}
	if RedactSlice(nil) != nil {
		t.Error("RedactSlice(nil) != nil")
	}
}

func TestRedactJSON(t *testing.T) {
	input := []byte(`{"password":"secret","apiKey":"abc123","note":"safe"}`)
	want := []byte(`{"password":"[REDACTED]","apiKey":"[REDACTED]","note":"safe"}`)
	if got := RedactJSON(input); !reflect.DeepEqual(got, want) {
		t.Errorf("RedactJSON() = %s, want %s", got, want)
	}
}

func TestRedactJSONEscapedQuote(t *testing.T) {
	input := []byte(`{"password":"se\"cret","note":"safe"}`)
	want := []byte(`{"password":"[REDACTED]","note":"safe"}`)
	if got := RedactJSON(input); !reflect.DeepEqual(got, want) {
		t.Errorf("RedactJSON() = %s, want %s", got, want)
	}
}

func TestQuotedCmdLineAndCommandLine(t *testing.T) {
	args := []string{"geneos", "run", "--password", "secret value", "-ssl-private-key=/etc/geneos/client.pem", "--api_key=abc123"}
	want := `geneos run --password [REDACTED] -ssl-private-key=/etc/geneos/client.pem --api_key=[REDACTED]`
	if got := QuotedCmdLine(args); got != want {
		t.Errorf("QuotedCmdLine() = %q, want %q", got, want)
	}
	attr := CommandLine(args)
	if attr.Key != "commandline" || attr.Value.String() != want {
		t.Errorf("CommandLine() = %#v, want commandline=%q", attr, want)
	}
}
