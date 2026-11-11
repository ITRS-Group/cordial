package xpath

import (
	"errors"
	"testing"
)

func TestParseXPathRoundTrip(t *testing.T) {
	paths := []string{
		`/geneos/gateway[(@name="gw")]/directory`,
		`/geneos/gateway[(@name="gw")]/directory/probe[(@name="np")]/managedEntity[(@name="server")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]/headlines/cell[(@name="Uptime")]`,
		`/geneos/gateway[(@name="gw")]/directory/probe[(@name="np")]/managedEntity[(@name="server")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]/rows/row[(@name="r1")]/cell[(@column="value")]`,
	}
	for _, input := range paths {
		t.Run(input, func(t *testing.T) {
			parsed, err := Parse(input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got := parsed.String(); got != input {
				t.Errorf("String() = %q, want %q", got, input)
			}
		})
	}
}

func TestXPathStringSortsEntityAttributes(t *testing.T) {
	x := &XPath{
		Gateway: &Gateway{Name: "gw"},
		Probe:   &Probe{},
		Entity: &Entity{Attributes: map[string]string{
			"DOMAIN":      "Trading and Investment",
			"APPLICATION": "PB India",
			"DEPARTMENT":  "PBW",
			"CATEGORY":    "Database",
		}},
	}
	want := `/geneos/gateway[(@name="gw")]/directory/probe/managedEntity[(attr("APPLICATION")="PB India")][(attr("CATEGORY")="Database")][(attr("DEPARTMENT")="PBW")][(attr("DOMAIN")="Trading and Investment")]`

	if got := x.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestParseXPathRejectsInvalidPaths(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "relative", input: `geneos/gateway`, wantErr: ErrRelativePath},
		{name: "wrong root", input: `/other/gateway`, wantErr: ErrInvalidPath},
		{name: "incomplete component", input: `/geneos/gateway/directory/probe/unknown`, wantErr: ErrInvalidPath},
		{name: "unsupported relative wildcard", input: `//unsupported`, wantErr: ErrInvalidPath},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse(test.input); !errors.Is(err, test.wantErr) {
				t.Fatalf("Parse(%q) error = %v, want %v", test.input, err, test.wantErr)
			}
		})
	}

	// TODO: define whether escaped quotes and separators in XPath names are supported.
}
