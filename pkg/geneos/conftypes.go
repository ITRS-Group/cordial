package geneos

import (
	"encoding/xml"
	"fmt"
)

// Vars is a container for specific variable types.
type Vars struct {
	XMLName         xml.Name        `xml:"var" json:"-" yaml:"-"`
	Comment         string          `xml:",comment"`
	Name            string          `xml:"name,attr"`
	Boolean         *bool           `xml:"boolean,omitempty" json:",omitempty" yaml:",omitempty"`
	Double          *float64        `xml:"double,omitempty" json:",omitempty" yaml:",omitempty"`
	Integer         *int64          `xml:"integer,omitempty" json:",omitempty" yaml:",omitempty"`
	String          string          `xml:"string,omitempty" json:",omitempty" yaml:",omitempty"`
	StringList      *StringList     `xml:"stringList,omitempty" json:",omitempty" yaml:",omitempty"`
	NameValueList   *NameValueList  `xml:"nameValueList,omitempty" json:",omitempty" yaml:",omitempty"`
	Macro           *Macro          `xml:"macro,omitempty" json:",omitempty" yaml:",omitempty"`
	Regex           *Regex          `xml:"regex,omitempty" json:",omitempty" yaml:",omitempty"`
	Other           []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr      `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

func (v Vars) GetKey() string { return v.Name }

type Macro struct {
	InsecureGatewayPort *EmptyStruct    `xml:"insecureGatewayPort,omitempty" json:",omitempty" yaml:",omitempty"`
	GatewayName         *EmptyStruct    `xml:"gatewayName,omitempty" json:",omitempty" yaml:",omitempty"`
	NetprobeName        *EmptyStruct    `xml:"netprobeName,omitempty" json:",omitempty" yaml:",omitempty"`
	NetprobeHost        *EmptyStruct    `xml:"netprobeHost,omitempty" json:",omitempty" yaml:",omitempty"`
	NetprobePort        *EmptyStruct    `xml:"netprobePort,omitempty" json:",omitempty" yaml:",omitempty"`
	ManagedEntitiesName *EmptyStruct    `xml:"managedEntityName,omitempty" json:",omitempty" yaml:",omitempty"`
	SamplerName         *EmptyStruct    `xml:"samplerName,omitempty" json:",omitempty" yaml:",omitempty"`
	SecureGatewayPort   *EmptyStruct    `xml:"secureGatewayPort,omitempty" json:",omitempty" yaml:",omitempty"`
	Other               []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes     []xml.Attr      `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
	Comment             string          `xml:",comment"`
}

type StringList struct {
	Strings []string `xml:"string" mapstructure:"string"`
	Comment string   `xml:",comment"`
}

// UnmarshalXML decodes the list from its child elements. Without this
// encoding/xml would use UnmarshalText and ignore the elements.
func (s *StringList) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain StringList
	return d.DecodeElement((*plain)(s), &start)
}

// UnmarshalText resets the list when text is empty, which is how an
// empty value in a configuration is decoded. A list cannot be set from
// non-empty text.
func (s *StringList) UnmarshalText(text []byte) error {
	if len(text) != 0 {
		return fmt.Errorf("cannot decode %q as a string list", text)
	}
	*s = StringList{}
	return nil
}

type NameValueList struct {
	NameValues []NameValue `xml:"item,omitempty" json:",omitempty" yaml:",omitempty"`
	Comment    string      `xml:",comment"`
}

// UnmarshalXML decodes the list from its child elements. Without this
// encoding/xml would use UnmarshalText and ignore the elements.
func (s *NameValueList) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain NameValueList
	return d.DecodeElement((*plain)(s), &start)
}

// UnmarshalText resets the list when text is empty, which is how an
// empty value in a configuration is decoded. A list cannot be set from
// non-empty text.
func (s *NameValueList) UnmarshalText(text []byte) error {
	if len(text) != 0 {
		return fmt.Errorf("cannot decode %q as a name value list", text)
	}
	*s = NameValueList{}
	return nil
}

type NameValue struct {
	Name    string `xml:"name"`
	Value   string `xml:"value"`
	Comment string `xml:",comment"`
}
