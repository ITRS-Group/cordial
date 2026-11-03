package profile

import "testing"

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
