package gwhub

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{input: "P1W", want: Week},
		{input: "P2DT3H4M5.5S", want: 2*Day + 3*time.Hour + 4*time.Minute + 5500*time.Millisecond},
		{input: "PT90M", want: 90 * time.Minute},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := ParseDuration(test.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Errorf("ParseDuration(%q) = %s, want %s", test.input, got, test.want)
			}
		})
	}
}

func TestParseDurationRejectsMalformedInput(t *testing.T) {
	if _, err := ParseDuration("PT1X"); err == nil {
		t.Fatal("ParseDuration accepted malformed input")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "zero", want: "PT0.000000S"},
		{name: "week", in: Week, want: "P1W"},
		{name: "mixed", in: Week + Day + time.Hour + 30*time.Minute + 15*time.Second, want: "P1W1DT1H30M15.000000S"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FormatDuration(test.in); got != test.want {
				t.Errorf("FormatDuration(%s) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}
