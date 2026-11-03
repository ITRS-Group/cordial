package cmd

import "testing"

func TestCommandFlagNormalization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "license", want: "licence"},
		{input: "attr", want: "attribute"},
		{input: "var", want: "variable"},
		{input: "output", want: "output"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			if got := string(cmdNormalizeFunc(nil, test.input)); got != test.want {
				t.Errorf("cmdNormalizeFunc(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
