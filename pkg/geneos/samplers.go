package geneos

import (
	"encoding/xml"
)

// SamplerOut is for output of XML. Plugins make it hard to share the
// same types between marshal and unmarshal
type SamplersOut struct {
	XMLName       xml.Name          `xml:"samplers" json:"-" yaml:"-"`
	Samplers      []SamplerOut      `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroupOut `xml:"samplerGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

type SamplerGroupOut struct {
	XMLName       xml.Name          `xml:"samplerGroup" json:"-" yaml:"-"`
	Name          string            `xml:"name,attr"`
	Disabled      bool              `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Samplers      []SamplerOut      `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroupOut `xml:"samplerGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

// A Sampler is a Geneos Sampler structure. The Plugin field should be
// populated with a pointer to a Plugin struct of the wanted type.
type SamplerOut struct {
	XMLName                xml.Name                `xml:"sampler" json:"-" yaml:"-"`
	Name                   string                  `xml:"name,attr"`
	Disabled               bool                    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Comment                string                  `xml:",comment" json:",omitempty" yaml:",omitempty"`
	Group                  *SingleLineString       `xml:"var-group,omitempty" json:",omitempty" yaml:",omitempty"`
	Interval               *Value                  `xml:"sampleInterval,omitempty" json:",omitempty" yaml:",omitempty"`
	SampleOnStartup        bool                    `xml:"sampleOnStartup" json:",omitempty" yaml:",omitempty"`
	Plugin                 interface{}             `xml:"plugin,omitempty" json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Dataviews              *[]Dataview             `xml:"dataviews>dataview,omitempty" json:",omitempty" yaml:",omitempty"`
	Schemas                *Schemas                `xml:"schemas,omitempty" json:",omitempty" yaml:",omitempty"`
	StandardisedFormatting *StandardisedFormatting `xml:"standardisedFormatting,omitempty" json:",omitempty" yaml:",omitempty"`
}

type Samplers struct {
	XMLName       xml.Name       `xml:"samplers" json:"-" yaml:"-"`
	Samplers      []Sampler      `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroup `xml:"samplerGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

type SamplerGroup struct {
	XMLName       xml.Name       `xml:"samplerGroup" json:"-" yaml:"-"`
	Name          string         `xml:"name,attr"`
	Disabled      bool           `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Samplers      []Sampler      `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroup `xml:"samplerGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

// A Sampler is a Geneos Sampler structure. The Plugin field should be
// populated with a pointer to a Plugin struct of the wanted type.
type Sampler struct {
	XMLName                xml.Name                `xml:"sampler" json:"-" yaml:"-"`
	Name                   string                  `xml:"name,attr"`
	Disabled               bool                    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Comment                string                  `xml:",comment" json:",omitempty" yaml:",omitempty"`
	Group                  *SingleLineString       `xml:"var-group,omitempty" json:",omitempty" yaml:",omitempty"`
	Interval               *Value                  `xml:"sampleInterval,omitempty" json:",omitempty" yaml:",omitempty"`
	SampleOnStartup        bool                    `xml:"sampleOnStartup" json:",omitempty" yaml:",omitempty"`
	Plugin                 *Plugin                 `xml:"plugin,omitempty" json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Dataviews              *[]Dataview             `xml:"dataviews>dataview,omitempty" json:",omitempty" yaml:",omitempty"`
	Schemas                *Schemas                `xml:"schemas,omitempty" json:",omitempty" yaml:",omitempty"`
	StandardisedFormatting *StandardisedFormatting `xml:"standardisedFormatting,omitempty" json:",omitempty" yaml:",omitempty"`
}

// Plugin lists all the plugins we know about
type Plugin struct {
	// Process    *ProcessesPlugin  `xml:"processes,omitempty" json:"processes,omitempty" yaml:"processes,omitempty"`
	FKM        *FKMPlugin        `xml:"fkm,omitempty" json:"fkm,omitempty" yaml:"fkm,omitempty"`
	FTM        *FTMPlugin        `xml:"ftm,omitempty" json:"ftm,omitempty" yaml:"ftm,omitempty"`
	Toolkit    *ToolkitPlugin    `xml:"toolkit,omitempty" json:"toolkit,omitempty" yaml:"toolkit,omitempty"`
	SQLToolkit *SQLToolkitPlugin `xml:"sql-toolkit,omitempty" json:"sql-toolkit,omitempty" yaml:"sql-toolkit,omitempty"`
	API        *APIPlugin        `xml:"api,omitempty" json:"api,omitempty" yaml:"api,omitempty"`
	APIStreams *APIStreamsPlugin `xml:"api-streams,omitempty" json:"api-streams,omitempty" yaml:"api-streams,omitempty"`
	GatewaySQL *GatewaySQLPlugin `xml:"Gateway-sql,omitempty" json:"Gateway-sql,omitempty" yaml:"Gateway-sql,omitempty"`
	Other      string            `xml:",any,omitempty" json:",omitempty" yaml:",omitempty"`
}
