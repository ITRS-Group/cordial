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
}

func (p *ToolkitPlugin) String() string { return p.XMLName.Local }

type EnvironmentVariable struct {
	XMLName xml.Name                    `xml:"variable" json:"-" yaml:"-"`
	Name    string                      `xml:"name"`
	Value   *common.SingleLineStringVar `xml:"value"`
}

type SecureEnvironmentVariable struct {
	XMLName xml.Name      `xml:"variable" json:"-" yaml:"-"`
	Name    string        `xml:"name"`
	Value   *common.Value `xml:"value"`
}

type ToolkitScript struct {
	Contents *ToolkitScriptContents `xml:"contents"`
}

type ToolkitScriptContents struct {
	Data     *string           `xml:"data,omitempty"`
	Var      *common.Reference `xml:"var,omitempty"`
	Filename *common.Value     `xml:"filename,omitempty"`
}
