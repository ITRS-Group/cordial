package toolkit

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type ToolkitPlugin struct {
	XMLName                    xml.Name                     `xml:"toolkit" json:"-" yaml:"-"`
	SamplerScript              *common.SingleLineStringVar  `xml:"samplerScript"`
	SamplerTimeout             *common.Value                `xml:"scriptTimeout,omitempty"`
	NumberOfDataviews          *common.Value                `xml:"numberOfDataviews,omitempty"`
	RowLimit                   *common.Value                `xml:"rowLimit,omitempty"`
	TerminateOnTimeOut         *common.Value                `xml:"terminateOnTimeOut,omitempty"`
	EnvironmentVariables       *[]EnvironmentVariable       `xml:"environmentVariables>variable,omitempty"`
	SecureEnvironmentVariables *[]SecureEnvironmentVariable `xml:"secureEnvironmentVariables>variable,omitempty"`
	Script                     *ToolkitScript               `xml:"script,omitempty"`
	ShowExecutionDuration      *bool                        `xml:"showExecutionDuration,omitempty"`
	Other                      []common.RawXMLElement       `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes            []xml.Attr                   `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the ToolkitPlugin without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v ToolkitPlugin) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

func (p *ToolkitPlugin) String() string { return p.XMLName.Local }

type EnvironmentVariable struct {
	XMLName         xml.Name                    `xml:"variable" json:"-" yaml:"-"`
	Name            string                      `xml:"name"`
	Value           *common.SingleLineStringVar `xml:"value"`
	Other           []common.RawXMLElement      `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr                  `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

type SecureEnvironmentVariable struct {
	XMLName         xml.Name               `xml:"variable" json:"-" yaml:"-"`
	Name            string                 `xml:"name"`
	Value           *common.Value          `xml:"value"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

type ToolkitScript struct {
	Contents        *ToolkitScriptContents `xml:"contents"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

type ToolkitScriptContents struct {
	Data            *string                `xml:"data,omitempty"`
	Var             *common.Reference      `xml:"var,omitempty"`
	Filename        *common.Value          `xml:"filename,omitempty"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}
