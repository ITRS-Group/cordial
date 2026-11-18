package xpath

import (
	"encoding/json"
	"errors"
	"reflect"
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

func ptr[T any](v T) *T { return &v }

func TestParseStringRoundTripSpecialNames(t *testing.T) {
	names := []string{
		`/dev/sda1`,
		`/`,
		`a"b`,
		`a")]b`,
		`back\slash`,
		`C:\temp\new`,
		"tab\tnew\nline",
		`[(@name="x")]`,
		`name with spaces/and"quotes"`,
		`ünï/çødé`,
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			x := NewTableCellPath(name, name)
			x.Dataview.Name = name
			x.Entity = &Entity{Name: name, Attributes: map[string]string{name: name, "k": "v/w"}}
			x.Sampler = &Sampler{Name: name, Type: ptr(name)}

			parsed, err := Parse(x.String())
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", x.String(), err)
			}
			if !reflect.DeepEqual(parsed, x) {
				t.Errorf("Parse(String()) = %+v, want %+v", parsed, x)
			}
			if got := parsed.String(); got != x.String() {
				t.Errorf("String() after round trip = %q, want %q", got, x.String())
			}

			// and through JSON
			b, err := json.Marshal(x)
			if err != nil {
				t.Fatal(err)
			}
			var y XPath
			if err := json.Unmarshal(b, &y); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v", b, err)
			}
			if !reflect.DeepEqual(&y, x) {
				t.Errorf("JSON round trip = %+v, want %+v", &y, x)
			}
		})
	}
}

func TestParseKeepsRowsAndHeadlinesSuffix(t *testing.T) {
	base := `/geneos/gateway[(@name="gw")]/directory/probe/managedEntity/sampler/dataview[(@name="dv")]`
	for _, input := range []string{
		base,
		base + `/rows`,
		base + `/rows/row`,
		base + `/rows/row/cell`,
		base + `/headlines`,
		base + `/headlines/cell`,
		base + `/headlines/cell[(@name="h")]`,
	} {
		t.Run(input, func(t *testing.T) {
			x, err := Parse(input)
			if err != nil {
				t.Fatal(err)
			}
			if got := x.String(); got != input {
				t.Errorf("String() = %q, want %q", got, input)
			}
		})
	}

	// the shorter forms are still dataview paths
	for _, suffix := range []string{`/rows`, `/headlines`} {
		x, _ := Parse(base + suffix)
		if !x.IsDataview() {
			t.Errorf("%s: not a dataview path", suffix)
		}
	}

	// wildcard form
	x, err := Parse(`//dataview[(@name="dv")]/rows`)
	if err != nil || !x.Rows || x.Dataview.Name != "dv" {
		t.Errorf("wildcard rows path = %+v, %v", x, err)
	}
}

func TestUnmarshalJSONNullAndEmpty(t *testing.T) {
	x := XPath{Gateway: &Gateway{Name: "keep"}}
	if err := json.Unmarshal([]byte(`null`), &x); err != nil || x.Gateway == nil || x.Gateway.Name != "keep" {
		t.Errorf("null: x = %+v, err = %v", x, err)
	}
	if err := json.Unmarshal([]byte(`""`), &x); err != nil || x.Gateway != nil {
		t.Errorf(`"": x = %+v, err = %v`, x, err)
	}
	// an empty XPath marshals to "" and must round trip
	b, _ := json.Marshal(XPath{})
	var y XPath
	if err := json.Unmarshal(b, &y); err != nil {
		t.Errorf("round trip of empty XPath: %v", err)
	}
	// a bad path leaves the value alone
	x = XPath{Gateway: &Gateway{Name: "keep"}}
	if err := json.Unmarshal([]byte(`"/not/geneos"`), &x); err == nil || x.Gateway == nil || x.Gateway.Name != "keep" {
		t.Errorf("bad path: x = %+v, err = %v", x, err)
	}

	// as a field
	var s struct {
		Target *XPath `json:"target"`
	}
	if err := json.Unmarshal([]byte(`{"target":null}`), &s); err != nil || s.Target != nil {
		t.Errorf("null field: %+v, %v", s, err)
	}
}

func TestMatch(t *testing.T) {
	x := &XPath{
		Gateway:  &Gateway{Name: "gw"},
		Probe:    &Probe{Name: "probe1"},
		Entity:   &Entity{Name: "server1", Attributes: map[string]string{"ENV": "prod", "DC": "london"}},
		Sampler:  &Sampler{Name: "disk", Type: ptr("Infra")},
		Dataview: &Dataview{Name: "disk"},
		Rows:     true,
		Row:      &Row{Name: "/dev/sda1"},
		Column:   &Column{Name: "usage"},
	}
	noType := x.Clone()
	noType.Sampler.Type = nil

	tests := []struct {
		name   string
		x      *XPath
		filter *XPath
		want   bool
	}{
		{"nil filter", x, nil, false},
		{"nil x", nil, x, false},
		{"empty filter", x, &XPath{}, true},
		{"gateway glob", x, &XPath{Gateway: &Gateway{Name: "g*"}}, true},
		{"gateway mismatch", x, &XPath{Gateway: &Gateway{Name: "other"}}, false},
		{"probe glob", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{Name: "probe?"}}, true},
		{"attribute only entity", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{Attributes: map[string]string{"ENV": "prod"}}}, true},
		{"attribute only entity glob", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{Attributes: map[string]string{"DC": "lon*"}}}, true},
		{"attribute only mismatch", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{Attributes: map[string]string{"ENV": "dev"}}}, false},
		{"attribute missing", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{Attributes: map[string]string{"NOPE": "*"}}}, false},
		{"entity name and attribute", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{Name: "server*", Attributes: map[string]string{"ENV": "prod"}}}, true},
		{"type only sampler", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{Type: ptr("Infra")}}, true},
		{"type only sampler mismatch", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{Type: ptr("Other")}}, false},
		{"type filter, sampler has no type", noType, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{Type: ptr("Infra")}}, false},
		{"empty type filter, sampler has no type", noType, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{Name: "disk", Type: ptr("")}}, true},
		{"wildcard type matches no type", noType, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{Type: ptr("*")}}, true},
		{"sampler name and type", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{Name: "disk", Type: ptr("In*")}}, true},
		{"row with slashes and star", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{}, Dataview: &Dataview{}, Rows: true, Row: &Row{Name: "*"}}, true},
		{"row with slashes and prefix", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{}, Dataview: &Dataview{}, Rows: true, Row: &Row{Name: "/dev/*"}}, true},
		{"row with slashes mismatch", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{}, Dataview: &Dataview{}, Rows: true, Row: &Row{Name: "/dev/sdb*"}}, false},
		{"row question mark", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{}, Dataview: &Dataview{}, Rows: true, Row: &Row{Name: "/dev?sda1"}}, true},
		{"row exact with slashes", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{}, Dataview: &Dataview{}, Rows: true, Row: &Row{Name: "/dev/sda1"}}, true},
		{"row escaped slash", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{}, Dataview: &Dataview{}, Rows: true, Row: &Row{Name: `\/dev\/sda1`}}, true},
		{"column", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{}, Dataview: &Dataview{}, Rows: true, Row: &Row{}, Column: &Column{Name: "us*"}}, true},
		{"headline filter on row path", x, &XPath{Gateway: &Gateway{}, Probe: &Probe{}, Entity: &Entity{}, Sampler: &Sampler{}, Dataview: &Dataview{}, Headline: &Headline{Name: "*"}}, false},
		{"bad pattern", x, &XPath{Gateway: &Gateway{Name: "[a-"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.x.Match(tt.filter); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveToDoesNotShareElements(t *testing.T) {
	typ := "T"
	attrs := map[string]string{"B": "2"}
	x, err := Parse(`/geneos/gateway[(@name="gw")]/directory/probe[(@name="p")]/managedEntity[(@name="e")][(attr("A")="1")]/sampler[(@name="s")][(@type="T")]/dataview[(@name="dv")]/rows/row[(@name="r")]/cell[(@column="c")]`)
	if err != nil {
		t.Fatal(err)
	}
	orig := x.String()

	tests := map[string]func(x *XPath) *XPath{
		"gateway":  func(x *XPath) *XPath { return x.ResolveTo(&Gateway{Name: "new"}) },
		"probe":    func(x *XPath) *XPath { return x.ResolveTo(&Probe{Name: "new"}) },
		"entity":   func(x *XPath) *XPath { return x.ResolveTo(&Entity{Name: "new", Attributes: attrs}) },
		"sampler":  func(x *XPath) *XPath { return x.ResolveTo(&Sampler{Name: "new", Type: &typ}) },
		"dataview": func(x *XPath) *XPath { return x.ResolveTo(&Dataview{Name: "new"}) },
		"headline": func(x *XPath) *XPath { return x.ResolveTo(&Headline{Name: "new"}) },
		"row":      func(x *XPath) *XPath { return x.ResolveTo(&Row{Name: "new"}) },
		"column":   func(x *XPath) *XPath { return x.ResolveTo(&Column{Name: "new"}) },
	}
	for name, resolve := range tests {
		t.Run(name, func(t *testing.T) {
			for _, start := range []*XPath{x, {}} {
				r := resolve(start)
				if r == nil {
					t.Fatal("nil result")
				}
				// modify everything in the result
				if r.Gateway != nil {
					r.Gateway.Name = "changed"
				}
				if r.Probe != nil {
					r.Probe.Name = "changed"
				}
				if r.Entity != nil {
					r.Entity.Name = "changed"
					r.Entity.Attributes["X"] = "changed"
				}
				if r.Sampler != nil {
					r.Sampler.Name = "changed"
					if r.Sampler.Type != nil {
						*r.Sampler.Type = "changed"
					}
				}
				if r.Dataview != nil {
					r.Dataview.Name = "changed"
				}
				if r.Row != nil {
					r.Row.Name = "changed"
				}
				if r.Column != nil {
					r.Column.Name = "changed"
				}
			}
			if got := x.String(); got != orig {
				t.Errorf("receiver was changed through the result: %q", got)
			}
			if typ != "T" || len(attrs) != 1 {
				t.Errorf("argument was changed through the result: %q %v", typ, attrs)
			}
		})
	}

	var nilPath *XPath
	if r := nilPath.ResolveTo(&Dataview{Name: "dv"}); r == nil || r.Dataview == nil || r.Dataview.Name != "dv" {
		t.Errorf("ResolveTo on a nil XPath = %+v", r)
	}
}
