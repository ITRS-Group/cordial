package profile

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestProfileLicenceOption(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "default", want: "geneos.lic"},
		{name: "explicit", args: []string{"-L", "/tmp/nofile.txt"}, want: "/tmp/nofile.txt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			flags := pflag.NewFlagSet(test.name, pflag.ContinueOnError)
			flags.StringP("licence", "L", "geneos.lic", "Licence file location")
			flags.Bool("debug", false, "Debug logging")
			if err := flags.Parse(test.args); err != nil {
				t.Fatal(err)
			}
			options := PflagSetToMap(flags, IncludeFlags...)
			got, err := Option(nil, "option:licence", false, options)
			if err != nil {
				t.Fatalf("expand licence option: %v", err)
			}
			if got != test.want {
				t.Errorf("licence option = %q, want %q", got, test.want)
			}
			if _, ok := options["debug"]; ok {
				t.Error("debug flag should not be included in profile options")
			}
		})
	}
}

func TestSelectPrefix(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]any
		expr  string
		trim  bool
		want  string
	}{
		{
			name:  "parameter begins with a character in the prefix",
			input: map[string]any{"secret": "provided"},
			expr:  "select:secret:fallback",
			want:  "provided",
		},
		{
			name: "fallback",
			expr: "select:missing:fallback",
			want: "fallback",
		},
		{
			name:  "trim selected value",
			input: map[string]any{"name": "  Demo  "},
			expr:  "select:name:fallback",
			trim:  true,
			want:  "Demo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := SelectPrefix(test.input, test.expr, test.trim)
			if err != nil {
				t.Fatalf("SelectPrefix() error = %v", err)
			}
			if got != test.want {
				t.Errorf("SelectPrefix() = %q, want %q", got, test.want)
			}
		})
	}
}
