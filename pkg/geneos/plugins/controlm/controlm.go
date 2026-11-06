package controlm

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName      xml.Name      `xml:"control-m" json:"-" yaml:"-"`
	Host         *common.Host  `xml:"host"`
	Port         *common.Value `xml:"port"`
	Username     *common.Value `xml:"username"`
	Password     *common.Value `xml:"password"`
	Timeout      *common.Value `xml:"timeout,omitempty"`
	VerifyServer *bool         `xml:"verifyServer,omitempty"`
	RowLimit     *common.Value `xml:"rowLimit,omitempty"`
	Dataviews    []Dataview    `xml:"dataviews>dataview,omitempty"`
}

func (*Plugin) String() string { return "control-m" }

type Dataview struct {
	Name       string      `xml:"name"`
	Parameters []Parameter `xml:"parameters,omitempty"`
	Columns    []string    `xml:"columns>column,omitempty"`
}

type Parameter struct {
	Parameter string `xml:"parameter"`
	Criteria  string `xml:"criteria"`
}
