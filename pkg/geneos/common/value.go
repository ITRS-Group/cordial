package common

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// Reference is an XML reference attribute.
type Reference struct {
	Name string `xml:"ref,attr" json:",omitempty" yaml:",omitempty"`
}

type EmptyStruct struct{}

// RawXMLElement preserves the nested contents of schema choices not yet
// modeled explicitly and of any elements found where the model has no
// field for them, so that they are written back unchanged.
type RawXMLElement struct {
	XMLName    xml.Name
	Comment    string     `xml:",comment"`
	Attributes []xml.Attr `xml:",any,attr"`
	Content    string     `xml:",innerxml"`
}

type Weekdays struct {
	Monday    bool            `xml:"monday"`
	Tuesday   bool            `xml:"tuesday"`
	Wednesday bool            `xml:"wednesday"`
	Thursday  bool            `xml:"thursday"`
	Friday    bool            `xml:"friday"`
	Saturday  bool            `xml:"saturday"`
	Sunday    bool            `xml:"sunday"`
	Other     []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// SingleLineStringVar represents a string that may contain Geneos variables.
type SingleLineStringVar struct {
	Parts []any
}

// NewSingleLineString splits Geneos $(name) references into XML data and var parts.
func NewSingleLineString(in string) (s *SingleLineStringVar) {
	s = &SingleLineStringVar{}
	for i := 0; i < len(in); i++ {
		st := strings.Index(in[i:], "$(")
		if st == -1 {
			s.Parts = append(s.Parts, Data{Data: in[i:]})
			return
		}
		if st > 0 {
			s.Parts = append(s.Parts, Data{Data: in[i : i+st]})
		}
		en := strings.Index(in[i+st+2:], ")")
		if en == -1 {
			s.Parts = append(s.Parts, Var{Var: in[i+st+2:]})
			return
		}
		s.Parts = append(s.Parts, Var{Var: in[i+st+2 : i+st+2+en]})
		i += st + 2 + en
	}
	return
}

var _ xml.Unmarshaler = (*SingleLineStringVar)(nil)

// UnmarshalXML decodes data and var parts.
func (v *SingleLineStringVar) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if v == nil {
		v = &SingleLineStringVar{}
	}
	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		element, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch element.Name.Local {
		case "data":
			t := &Data{}
			if err := d.DecodeElement(t, &element); err != nil {
				return err
			}
			v.Parts = append(v.Parts, t)
		case "var":
			t := &Var{}
			if err := d.DecodeElement(t, &element); err != nil {
				return err
			}
			v.Parts = append(v.Parts, t)
		}
	}
}

var _ fmt.Stringer = (*SingleLineStringVar)(nil)

// String joins the data and variable parts into a Geneos string.
func (s *SingleLineStringVar) String() (out string) {
	if s == nil {
		return
	}
	for _, part := range s.Parts {
		if value, ok := part.(fmt.Stringer); ok {
			out += value.String()
		}
	}
	return
}

var _ json.Marshaler = (*SingleLineStringVar)(nil)

// MarshalJSON encodes the expanded string value.
func (s *SingleLineStringVar) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

var _ yaml.Marshaler = (*SingleLineStringVar)(nil)

// MarshalYAML encodes the expanded string value.
func (s *SingleLineStringVar) MarshalYAML() (any, error) {
	return s.String(), nil
}

// Var is a Geneos variable reference.
type Var struct {
	XMLName xml.Name `xml:"var" json:"-" yaml:"-"`
	Var     string   `xml:"ref,attr"`
}

func (v Var) String() string { return "$(" + v.Var + ")" }

// Data is literal text in a Geneos variable value.
type Data struct {
	XMLName xml.Name `xml:"data" json:"-" yaml:"-"`
	Data    string   `xml:",chardata"`
}

func (d Data) String() string { return d.Data }

// Value is either literal data, a variable reference, or a supported password encoding.
type Value struct {
	Data   *Data  `xml:"data,omitempty" json:",omitempty" yaml:",omitempty"`
	Var    *Var   `xml:"var,omitempty" json:",omitempty" yaml:",omitempty"`
	ExtPwd string `xml:"extPwd,omitempty" json:",omitempty" yaml:",omitempty"`
	StdAES string `xml:"stdAES,omitempty" json:",omitempty" yaml:",omitempty"`

	// Other keeps any other elements, from a choice with more than a
	// simple value, so that they are not lost
	Other []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// NewValue converts primitive input into a Geneos value. Input that is
// nil, an empty string (after trimming white space) or an empty slice
// results in a nil Value. A string of the form "$(name)" is a variable
// reference and anything else is literal data. The elements of a slice
// are joined with a comma. Pointers are followed.
func NewValue(in any) *Value {
	if in == nil {
		return nil
	}
	rv := reflect.ValueOf(in)
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	var text string
	switch rv.Kind() {
	case reflect.String:
		text = strings.TrimSpace(rv.String())
	case reflect.Slice, reflect.Array:
		items := make([]string, 0, rv.Len())
		for i := range rv.Len() {
			items = append(items, fmt.Sprint(rv.Index(i).Interface()))
		}
		text = strings.Join(items, ",")
	default:
		text = fmt.Sprint(rv.Interface())
	}
	if text == "" {
		return nil
	}

	// a variable reference is a single "$(name)" and nothing else
	if len(text) > 3 && strings.HasPrefix(text, "$(") && strings.IndexByte(text, ')') == len(text)-1 {
		return &Value{Var: &Var{Var: text[2 : len(text)-1]}}
	}
	return &Value{Data: &Data{Data: text}}
}

var _ xml.Unmarshaler = (*Value)(nil)

// UnmarshalXML decodes data, var, and password variants.
func (v *Value) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if v == nil {
		v = &Value{}
	}
	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		element, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch element.Name.Local {
		case "data":
			data := Data{}
			if err := d.DecodeElement(&data, &element); err != nil {
				return err
			}
			v.Data = &data
		case "var":
			ref := &Var{}
			if err := d.DecodeElement(ref, &element); err != nil {
				return err
			}
			v.Var = ref
		case "extPwd":
			if err := d.DecodeElement(&v.ExtPwd, &element); err != nil {
				return err
			}
		case "stdAES":
			if err := d.DecodeElement(&v.StdAES, &element); err != nil {
				return err
			}
		default:
			var other RawXMLElement
			if err := d.DecodeElement(&other, &element); err != nil {
				return err
			}
			v.Other = append(v.Other, other)
		}
	}
}

var _ fmt.Stringer = (*Value)(nil)

// String converts a Geneos value to its textual representation.
func (s *Value) String() string {
	if s == nil {
		return ""
	}
	if s.Var != nil {
		return "$(" + s.Var.Var + ")"
	}
	if s.ExtPwd != "" {
		return "extpwd:" + s.ExtPwd
	}
	if s.StdAES != "" {
		return "[encoded-password]"
	}
	if s.Data != nil {
		return s.Data.Data
	}
	return ""
}

// Host identifies a host by name, address, or variable.
type Host struct {
	Name      string          `xml:"name,omitempty"`
	IPAddress *IPAddress      `xml:"ipAddress,omitempty"`
	Var       *Reference      `xml:"var,omitempty"`
	Other     []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

func (t *Host) String() string {
	if t == nil {
		return ""
	}
	if t.IPAddress != nil {
		return t.IPAddress.String()
	}
	if t.Var != nil {
		return "$(" + t.Var.Name + ")"
	}
	return t.Name
}

// IPAddress stores four address octets.
type IPAddress struct {
	Octets []int           `xml:"value"`
	Other  []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

func (ip *IPAddress) String() string {
	if ip == nil {
		return "0.0.0.0"
	}
	if len(ip.Octets) != 4 {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.%d", ip.Octets[0], ip.Octets[1], ip.Octets[2], ip.Octets[3])
}

type Regex struct {
	Regex string          `xml:"regex" json:"regex,omitempty" yaml:"regex,omitempty"`
	Flags *RegexFlags     `xml:"flags,omitempty" json:"flags,omitempty" yaml:"flags,omitempty"`
	Other []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

type RegexFlags struct {
	CaseInsensitive *bool           `xml:"i,omitempty" json:"i,omitempty" yaml:"i,omitempty"`
	DotMatchesAll   *bool           `xml:"s,omitempty" json:"s,omitempty" yaml:"s,omitempty"`
	Other           []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

func (r *Regex) String() (s string) {
	if r == nil {
		return ""
	}
	s = fmt.Sprintf("/%s/", r.Regex)
	if r.Flags == nil {
		return
	}
	if r.Flags.CaseInsensitive != nil && *r.Flags.CaseInsensitive {
		s += "i"
	}
	if r.Flags.DotMatchesAll != nil && *r.Flags.DotMatchesAll {
		s += "s"
	}
	return
}
