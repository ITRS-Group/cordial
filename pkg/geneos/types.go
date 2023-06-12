package geneos

import "encoding/xml"

type Types struct {
	XMLName    xml.Name    `xml:"types" json:"-" yaml:"-"`
	Types      []Type      `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"type"`
	TypeGroups []TypeGroup `json:",omitempty" yaml:",omitempty" mapstructure:"typegroup"`
}

type TypeGroup struct {
	XMLName    xml.Name    `xml:"typeGroup" json:"-" yaml:"-"`
	Name       string      `xml:"name,attr"`
	Disabled   bool        `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Types      []Type      `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"type"`
	TypeGroups []TypeGroup `json:",omitempty" yaml:",omitempty" mapstructure:"typegroup"`
}

type Type struct {
	XMLName     xml.Name     `xml:"type" json:"-" yaml:"-"`
	Name        string       `xml:"name,attr"`
	Disabled    bool         `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Environment *Reference   `xml:"environment,omitempty" json:",omitempty" yaml:",omitempty"`
	Vars        []Vars       `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"var"`
	Samplers    []SamplerRef `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
}

type SamplerRef struct {
	XMLName  xml.Name `xml:"sampler" json:"-" yaml:"-"`
	Ref      string   `xml:"ref,attr" json:",omitempty" yaml:",omitempty"`
	Disabled bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
}
