package common

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestNewValue(t *testing.T) {
	str := "from pointer"
	var nilStr *string
	type named string

	tests := []struct {
		name     string
		in       any
		wantNil  bool
		wantData string
		wantVar  string
	}{
		{name: "nil", in: nil, wantNil: true},
		{name: "nil pointer", in: nilStr, wantNil: true},
		{name: "empty string", in: "", wantNil: true},
		{name: "blank string", in: "  \t", wantNil: true},
		{name: "string", in: " text ", wantData: "text"},
		{name: "named string", in: named("n"), wantData: "n"},
		{name: "pointer", in: &str, wantData: "from pointer"},
		{name: "variable", in: "$(name)", wantVar: "name"},
		{name: "int", in: 42, wantData: "42"},
		{name: "bool", in: true, wantData: "true"},
		{name: "string slice keeps every element", in: []string{"a", "b", "c"}, wantData: "a,b,c"},
		{name: "int slice keeps every element", in: []int{1, 2, 3}, wantData: "1,2,3"},
		{name: "empty slice", in: []string{}, wantNil: true},
		{name: "two variables are data", in: "$(a)b$(c)", wantData: "$(a)b$(c)"},
		{name: "empty variable name is data", in: "$()", wantData: "$()"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValue(tt.in)
			if tt.wantNil {
				if v != nil {
					t.Fatalf("NewValue(%v) = %+v, want nil", tt.in, v)
				}
				return
			}
			if v == nil {
				t.Fatalf("NewValue(%v) = nil", tt.in)
			}
			switch {
			case tt.wantVar != "":
				if v.Var == nil || v.Var.Var != tt.wantVar || v.Data != nil {
					t.Errorf("got %+v, want var %q", v, tt.wantVar)
				}
			default:
				if v.Data == nil || v.Data.Data != tt.wantData || v.Var != nil {
					t.Errorf("got %+v, want data %q", v, tt.wantData)
				}
			}
		})
	}
}

func TestValueString(t *testing.T) {
	tests := []struct {
		name string
		v    *Value
		want string
	}{
		{"nil", nil, ""},
		{"empty", &Value{}, ""},
		{"data", &Value{Data: &Data{Data: "x"}}, "x"},
		{"var", &Value{Var: &Var{Var: "v"}}, "$(v)"},
		{"extpwd", &Value{ExtPwd: "p"}, "extpwd:p"},
		{"aes", &Value{StdAES: "p"}, "[encoded-password]"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("%s: String() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestHostString(t *testing.T) {
	tests := []struct {
		name string
		h    *Host
		want string
	}{
		{"nil", nil, ""},
		{"name", &Host{Name: "server"}, "server"},
		{"ip", &Host{IPAddress: &IPAddress{Octets: []int{10, 0, 0, 1}}}, "10.0.0.1"},
		{"variable", &Host{Var: &Reference{Name: "HOST"}}, "$(HOST)"},
	}
	for _, tt := range tests {
		if got := tt.h.String(); got != tt.want {
			t.Errorf("%s: String() = %q, want %q", tt.name, got, tt.want)
		}
	}

	// and as decoded from XML
	var h Host
	if err := xml.Unmarshal([]byte(`<host><var ref="MYHOST"/></host>`), &h); err != nil {
		t.Fatal(err)
	}
	if got := h.String(); got != "$(MYHOST)" {
		t.Errorf("decoded variable host = %q", got)
	}
}

func TestValueXMLRoundTrip(t *testing.T) {
	for _, in := range []string{
		`<v><data>text</data></v>`,
		`<v><var ref="x"></var></v>`,
		`<v><stdAES>+encs+ABC</stdAES></v>`,
		`<v><extPwd>abc</extPwd></v>`,
	} {
		var v struct {
			XMLName xml.Name `xml:"v"`
			Value
		}
		if err := xml.Unmarshal([]byte(in), &v); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		out, err := xml.Marshal(&v)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), strings.TrimSuffix(strings.TrimPrefix(in, "<v>"), "</v>")) {
			t.Errorf("round trip of %s = %s", in, out)
		}
	}
}
