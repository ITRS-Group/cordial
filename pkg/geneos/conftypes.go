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
	"encoding/xml"
	"fmt"
	"strings"
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
	Ref string `xml:"ref,attr" json:",omitempty" yaml:",omitempty"`
}

// A SingleLineString is a container for a single line string that
// can be made up of static text and variable references. Use like this:
//
//	type MyContainer struct {
//	    XMLName  xml.Name             `xml:"mycontainer"`
//	    VarField *SingleLineString `xml:"fieldname"`
//	}
//
//	func blah() {
//	    x := MyContainer{
//	        VarField: geneos.SingleLineString("hello $(var) world!")
//	    }
//	    ...
//	}
type SingleLineString struct {
	Parts []interface{}
}

// NewSingleLineString take a plain string and locates any Geneos style
// variables of the form $(var) - note these are parenthesis and not brackets -
// and splits the string into Data and Var parts as required so that this can be
// used directly in the XML encodings.
func NewSingleLineString(in string) (s *SingleLineString) {
	s = &SingleLineString{}
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

func (s *SingleLineString) String() (out string) {
	for _, p := range s.Parts {
		switch t := p.(type) {
		case Data:
			out += t.Data
		case Var:
			out += fmt.Sprintf("$(%s)", t.Var)
		default:
			panic("unknown part type")
		}
	}
	return
}

type Var struct {
	XMLName xml.Name `xml:"var" json:"-" yaml:"-"`
	Var     string   `xml:"ref,attr"`
}

type Data struct {
	XMLName xml.Name `xml:"data" json:"-" yaml:"-"`
	Data    string   `xml:",chardata"`
}

// A Value can contain multiple parts. In the most basic and common form
// it is a mix of text (as "data") and variables
type Value struct {
	Parts []interface{}
}

// NewValue takes an argument and if a string removes leading and
// trailing spaces. If the string is of the form "$(var)" then returns a
// pointer to a VarData struct containing a Var{} or if a non-empty
// string returns a Data{}. If the string is empty then a nil pointer is
// returned. Any other value is copied as is. This allows
// `xml:",omixempty"“ to leave out VarData fields that contain no data.
func NewValue(in interface{}) (n *Value) {
	n = &Value{}
	switch s := in.(type) {
	case string:
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		if strings.HasPrefix(s, "$(") && strings.HasSuffix(s, ")") {
			n.Parts = append(n.Parts, Var{Var: s[2 : len(s)-1]})
		} else {
			n.Parts = append(n.Parts, Data{Data: s})
		}
	default:
		n.Parts = append(n.Parts, in)
	}

	return
}

type Regex struct {
	Regex string        `xml:"regex"`
	Flags *[]RegexFlags `xml:"regexFlags,omitempty"`
}

type RegexFlags struct {
	CaseInsensitive *bool `xml:"i,omitempty"`
	DotMatchesAll   *bool `xml:"s,omitempty"`
}
