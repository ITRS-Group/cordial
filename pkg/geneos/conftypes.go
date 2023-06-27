/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package geneos

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// Vars is a container for specific variable types. Only one field should
// be set at a time. This list is not complete, much like many of the
// configuration settings.
type Vars struct {
	XMLName       xml.Name       `xml:"var" json:"-" yaml:"-"`
	Name          string         `xml:"name,attr"`
	Boolean       *bool          `xml:"boolean,omitempty" json:",omitempty" yaml:",omitempty"`
	Double        *float64       `xml:"double,omitempty" json:",omitempty" yaml:",omitempty"`
	Integer       *int64         `xml:"integer,omitempty" json:",omitempty" yaml:",omitempty"`
	String        string         `xml:"string,omitempty" json:",omitempty" yaml:",omitempty"`
	StringList    *StringList    `xml:"stringList,omitempty" json:",omitempty" yaml:",omitempty"`
	NameValueList *NameValueList `xml:"nameValueList,omitempty" json:",omitempty" yaml:",omitempty"`
	Macro         *Macro         `xml:"macro,omitempty" json:",omitempty" yaml:",omitempty"`
}

// GetKey satisfies the KeyedObject interface
func (v Vars) GetKey() string {
	return v.Name
}

// Macro is a container for the various macro variable types. Only
// initialise one field to an empty struct, the rest must be nil
// pointers. e.g.
//
//	macro := geneos.Macro{InsecureGatewayPort: &geneos.EmptyStruct{}}
type Macro struct {
	InsecureGatewayPort *EmptyStruct `xml:"insecureGatewayPort,omitempty" json:",omitempty" yaml:",omitempty"`
	GatewayName         *EmptyStruct `xml:"gatewayName,omitempty" json:",omitempty" yaml:",omitempty"`
	NetprobeName        *EmptyStruct `xml:"netprobeName,omitempty" json:",omitempty" yaml:",omitempty"`
	NetprobeHost        *EmptyStruct `xml:"netprobeHost,omitempty" json:",omitempty" yaml:",omitempty"`
	NetprobePort        *EmptyStruct `xml:"netprobePort,omitempty" json:",omitempty" yaml:",omitempty"`
	ManagedEntitiesName *EmptyStruct `xml:"managedEntityName,omitempty" json:",omitempty" yaml:",omitempty"`
	SamplerName         *EmptyStruct `xml:"samplerName,omitempty" json:",omitempty" yaml:",omitempty"`
	SecureGatewayPort   *EmptyStruct `xml:"secureGatewayPort,omitempty" json:",omitempty" yaml:",omitempty"`
}

// EmptyStruct is an empty struct used to indicate which macro VarMacro refers
// to.
type EmptyStruct struct{}

type StringList struct {
	Strings []string `xml:"string" mapstructure:"string"`
}

// UnmarshalText conforms to the mapstructure decode hook to covert an
// empty tag to a map, not a string (from mxj)
func (s *StringList) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		s = &StringList{}
	}
	return nil
}

type NameValueList struct {
	NameValues []NameValue `xml:"item,omitempty" json:",omitempty" yaml:",omitempty"`
}

// UnmarshalText conforms to the mapstructure decode hook to covert an
// empty tag to a map, not a string (from mxj)
func (s *NameValueList) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		s = &NameValueList{}
	}
	return nil
}

type NameValue struct {
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

type Reference struct {
	Name string `xml:"ref,attr" json:",omitempty" yaml:",omitempty"`
}

// A SingleLineStringVar is a container for a single line string that
// can be made up of static text and variable references. Use like this:
//
//	type MyContainer struct {
//	    XMLName  xml.Name             `xml:"mycontainer"`
//	    VarField *SingleLineStringVar `xml:"fieldname"`
//	}
//
//	func blah() {
//	    x := MyContainer{
//	        VarField: geneos.SingleLineStringVar("hello $(var) world!")
//	    }
//	    ...
//	}
type SingleLineStringVar struct {
	Parts []interface{}
}

// NewSingleLineString take a plain string and locates any Geneos style
// variables of the form $(var) - note these are parenthesis and not brackets -
// and splits the string into Data and Var parts as required so that this can be
// used directly in the XML encodings.
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

// ensure that Value satisfies xml.Unmarshaler interface
var _ xml.Unmarshaler = (*SingleLineStringVar)(nil)

func (v *SingleLineStringVar) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
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
			err = d.DecodeElement(&t, &element)
			if err != nil {
				return err
			}
			v.Parts = append(v.Parts, t)
		case "var":
			t := &Var{}
			err = d.DecodeElement(&t, &element)
			if err != nil {
				return err
			}
			v.Parts = append(v.Parts, t)
		}
	}
}

var _ fmt.Stringer = (*SingleLineStringVar)(nil)

func (s *SingleLineStringVar) String() (out string) {
	if s == nil {
		return
	}
	for _, p := range s.Parts {
		s, ok := p.(fmt.Stringer)
		if ok {
			out += s.String()
		}
	}
	return
}

var _ json.Marshaler = (*SingleLineStringVar)(nil)

func (s *SingleLineStringVar) MarshalJSON() (out []byte, err error) {
	return json.Marshal(s.String())
}

var _ yaml.Marshaler = (*SingleLineStringVar)(nil)

func (s *SingleLineStringVar) MarshalYAML() (out interface{}, err error) {
	out = s.String()
	return
}

type Var struct {
	XMLName xml.Name `xml:"var" json:"-" yaml:"-"`
	Var     string   `xml:"ref,attr"`
}

func (v Var) String() string {
	return "$(" + v.Var + ")"
}

type Data struct {
	XMLName xml.Name `xml:"data" json:"-" yaml:"-"`
	Data    string   `xml:",chardata"`
}

func (d Data) String() string {
	return d.Data
}

// A Value is either a simple string or a variable
//
// It can also contain an external password reference or an encoded password
type Value struct {
	Data   *Data  `xml:"data,omitempty" json:",omitempty" yaml:",omitempty"`
	Var    *Var   `xml:"var,omitempty" json:",omitempty" yaml:",omitempty"`
	ExtPwd string `xml:"extPwd,omitempty" json:",omitempty" yaml:",omitempty"`
	StdAES string `xml:"stdAES,omitempty" json:",omitempty" yaml:",omitempty"`
}

// NewValue takes an argument and if a string removes leading and
// trailing spaces. If the string is of the form "$(var)" then returns a
// pointer to a VarData struct containing a Var{} or if a non-empty
// string returns a Data{}. If the string is empty then a nil pointer is
// returned. Any other value is copied as is. This allows
// `xml:",omitempty"“ to leave out VarData fields that contain no data.
func NewValue(in interface{}) (n *Value) {
	n = &Value{}
	switch s := in.(type) {
	case string:
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		if strings.HasPrefix(s, "$(") && strings.HasSuffix(s, ")") {
			n.Var = &Var{Var: s[2 : len(s)-1]}
		} else {
			n.Data = &Data{Data: s}
			// n.Data = append(n.Data, Data{Data: s})
		}
	case []string:
		for _, str := range s {
			n.Data = &Data{Data: str}
			// n.Data = append(n.Data, Data{Data: str})
		}
	default:
		if reflect.TypeOf(s).Kind() == reflect.Slice {
			sl := reflect.ValueOf(s)
			for i := 0; i < sl.Len(); i++ {
				n.Data = &Data{Data: fmt.Sprint(sl.Index(i))}
				// n.Data = append(n.Data, Data{Data: fmt.Sprint(sl.Index(i))})
			}
		} else {
			n.Data = &Data{Data: fmt.Sprint(s)}
			// n.Data = append(n.Data, Data{Data: fmt.Sprint(s)})
		}
	}

	return
}

// ensure that Value satisfies xml.Unmarshaler interface
var _ xml.Unmarshaler = (*Value)(nil)

func (v *Value) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
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
			t := Data{}
			err = d.DecodeElement(&t, &element)
			if err != nil {
				return err
			}
			v.Data = &t
			// v.Data = append(v.Data, t)
		case "var":
			t := &Var{}
			err = d.DecodeElement(&t, &element)
			if err != nil {
				return err
			}
			v.Var = t
		case "extPwd":
			err = d.DecodeElement(&v.ExtPwd, &element)
			if err != nil {
				return err
			}
		case "stdAES":
			err = d.DecodeElement(&v.StdAES, &element)
			if err != nil {
				panic(err)
				return err
			}
		}
	}
}

var _ fmt.Stringer = (*Value)(nil)

func (s *Value) String() (out string) {
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

type Regex struct {
	Regex string        `xml:"regex"`
	Flags *[]RegexFlags `xml:"regexFlags,omitempty"`
}

type RegexFlags struct {
	CaseInsensitive *bool `xml:"i,omitempty"`
	DotMatchesAll   *bool `xml:"s,omitempty"`
}

type Host struct {
	Name      string     `xml:"name,omitempty"`
	IPAddress *IPAddress `xml:"ipAddress,omitempty"`
	Var       *Reference `xml:"var,omitempty"`
}

func (t *Host) String() string {
	if t.IPAddress != nil {
		return fmt.Sprintf("%d.%d.%d.%d", t.IPAddress.Octets[0], t.IPAddress.Octets[1], t.IPAddress.Octets[2], t.IPAddress.Octets[3])
	}
	if t.Var != nil {
		return "$(" + t.Name + ")"
	}
	return t.Name
}

type IPAddress struct {
	Octets [4]uint8 `xml:"value"`
}
