package geneos

import "encoding/xml"

type Samplers struct {
	XMLName       xml.Name       `xml:"samplers" json:"-" yaml:"-"`
	Samplers      []Sampler      `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroup `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

type SamplerGroup struct {
	XMLName       xml.Name       `xml:"samplerGroup" json:"-" yaml:"-"`
	Name          string         `xml:"name,attr"`
	Disabled      bool           `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Samplers      []Sampler      `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroup `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

// A Sampler is a Geneos Sampler structure. The Plugin field should be
// populated with a pointer to a Plugin struct of the wanted type.
type Sampler struct {
	XMLName                xml.Name                `xml:"sampler" json:"-" yaml:"-"`
	Name                   string                  `xml:"name,attr"`
	Disabled               bool                    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Comment                string                  `xml:",comment"`
	Group                  *SingleLineString       `xml:"var-group,omitempty" json:",omitempty" yaml:",omitempty"`
	Interval               *Value                  `xml:"sampleInterval,omitempty" json:",omitempty" yaml:",omitempty"`
	SampleOnStartup        bool                    `xml:"sampleOnStartup"`
	Plugin                 interface{}             `xml:"plugin"`
	Dataviews              *[]Dataview             `xml:"dataviews>dataview,omitempty" json:",omitempty" yaml:",omitempty"`
	Schemas                *Schemas                `xml:"schemas,omitempty" json:",omitempty" yaml:",omitempty"`
	StandardisedFormatting *StandardisedFormatting `xml:"standardisedFormatting,omitempty" json:",omitempty" yaml:",omitempty"`
}
