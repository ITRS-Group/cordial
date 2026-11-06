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

type Weekdays struct {
	Monday    bool `xml:"monday"`
	Tuesday   bool `xml:"tuesday"`
	Wednesday bool `xml:"wednesday"`
	Thursday  bool `xml:"thursday"`
	Friday    bool `xml:"friday"`
	Saturday  bool `xml:"saturday"`
	Sunday    bool `xml:"sunday"`
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
}

// NewValue converts primitive input into a Geneos value.
func NewValue(in any) (value *Value) {
	value = &Value{}
	switch item := in.(type) {
	case string:
		item = strings.TrimSpace(item)
		if item == "" {
			return nil
		}
		if strings.HasPrefix(item, "$(") && strings.HasSuffix(item, ")") {
			value.Var = &Var{Var: item[2 : len(item)-1]}
		} else {
			value.Data = &Data{Data: item}
		}
	case []string:
		for _, str := range item {
			value.Data = &Data{Data: str}
		}
	default:
		if reflect.TypeOf(item).Kind() == reflect.Slice {
			slice := reflect.ValueOf(item)
			for i := 0; i < slice.Len(); i++ {
				value.Data = &Data{Data: fmt.Sprint(slice.Index(i))}
			}
		} else {
			value.Data = &Data{Data: fmt.Sprint(item)}
		}
	}
	return
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
	return fmt.Sprint(s.Data)
}

// Host identifies a host by name, address, or variable.
type Host struct {
	Name      string     `xml:"name,omitempty"`
	IPAddress *IPAddress `xml:"ipAddress,omitempty"`
	Var       *Reference `xml:"var,omitempty"`
}

func (t *Host) String() string {
	if t == nil {
		return ""
	}
	if t.IPAddress != nil {
		return t.IPAddress.String()
	}
	if t.Var != nil {
		return "$(" + t.Name + ")"
	}
	return t.Name
}

// IPAddress stores four address octets.
type IPAddress struct {
	Octets []int `xml:"value"`
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
	Regex string      `xml:"regex" json:"regex,omitempty" yaml:"regex,omitempty"`
	Flags *RegexFlags `xml:"flags,omitempty" json:"flags,omitempty" yaml:"flags,omitempty"`
}

type RegexFlags struct {
	CaseInsensitive *bool `xml:"i,omitempty" json:"i,omitempty" yaml:"i,omitempty"`
	DotMatchesAll   *bool `xml:"s,omitempty" json:"s,omitempty" yaml:"s,omitempty"`
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
