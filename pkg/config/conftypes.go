package config

import (
	"encoding/xml"
	"strings"
)

// Vars is a container for specific variable types. Only one field should
// be set at a time. This list is not complete, much like many of the
// configuration settings.
type Vars struct {
	XMLName    xml.Name    `xml:"var"`
	Name       string      `xml:"name,attr"`
	Boolean    *bool       `xml:"boolean,omitempty"`
	Double     *float64    `xml:"double,omitempty"`
	Integer    *int64      `xml:"integer,omitempty"`
	String     string      `xml:"string,omitempty"`
	StringList *StringList `xml:"stringList,omitempty"`
	Macro      *Macro      `xml:"macro,omitempty"`
}

// Macro is a container for the various macro variable types. Only
// initialise one field to an empty struct, the rest must be nil
// pointers. e.g.
//
//	macro := config.Macro{InsecureGatewayPort: &config.EmptyStruct{}}
//
type Macro struct {
	InsecureGatewayPort *EmptyStruct `xml:"insecureGatewayPort,omitempty"`
	GatewayName         *EmptyStruct `xml:"gatewayName,omitempty"`
	NetprobeName        *EmptyStruct `xml:"netprobeName,omitempty"`
	NetprobeHost        *EmptyStruct `xml:"netprobeHost,omitempty"`
	NetprobePort        *EmptyStruct `xml:"netprobePort,omitempty"`
	ManagedEntitiesName *EmptyStruct `xml:"managedEntityName,omitempty"`
	SamplerName         *EmptyStruct `xml:"samplerName,omitempty"`
	SecureGatewayPort   *EmptyStruct `xml:"secureGatewayPort,omitempty"`
}

// EmptyStruct is an empty struct used to indicate which macro VarMacro refers
// to.
type EmptyStruct struct{}

type StringList struct {
	Strings []string `xml:"string"`
}

type VarRef struct {
	Name string `xml:"ref,attr"`
}

// A container for a single line string that can be made up of static text
// and variable references. Use like this:
//
//	type MyContainer struct {
//		XMLName  xml.Name         `xml:"mycontainer`
//		VarField SingleLineStringVar `xml:"fieldname"`
//	}
//
//	func blah() {
//		x := MyContainer{
//			VarField: geneos.NewSingleLineString(geneos.Data{Data: "hello"}, geneos.Var{Var: "world"}, geneos.Data{Data: "!"})
//		}
//		...
//	}
type SingleLineStringVar struct {
	Parts []interface{}
}

func NewSingleLineStringVar(parts ...interface{}) (s SingleLineStringVar) {
	s.Parts = append([]interface{}{}, parts...)
	return
}

// ExpandSingleLineStringVar take a plain string and locates any Geneos style
// variables of the form $(var) - note these are parenthesis and not brackets -
// and splits the string into Data and Var parts as required so that this can be
// used directly in the XML encodings
func ExpandSingleLineStringVar(in string) (s SingleLineStringVar) {
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

type Var struct {
	XMLName xml.Name `xml:"var"`
	Var     string   `xml:"ref,attr"`
}

type Data struct {
	XMLName xml.Name `xml:"data"`
	Data    string   `xml:",chardata"`
}

// DataOrVar is a struct that contains either a Var or a Data type depending
// on the usage.
type DataOrVar struct {
	Part interface{}
}

// NewDataOrVar takes a string argument and removes leading and trailing
// spaces. If the string is of the form "$(var)" then returns a pointer
// to a VarData struct containing a Var{} or if a non-empty string
// returns a Data{}. If the string is empty then a nil pointer is
// returned. This allows `xml:",omixempty"`` to leave out VarData fields
// that contain no data.
func NewDataOrVar(s string) (n *DataOrVar) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	n = &DataOrVar{}
	if strings.HasPrefix(s, "$(") && strings.HasSuffix(s, ")") {
		n.Part = Var{Var: s[2 : len(s)-1]}
	} else {
		n.Part = Data{Data: s}
	}

	return
}
