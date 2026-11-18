package controlm

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName         xml.Name               `xml:"control-m" json:"-" yaml:"-"`
	Host            *common.Host           `xml:"host"`
	Port            *common.Value          `xml:"port"`
	Username        *common.Value          `xml:"username"`
	Password        *common.Value          `xml:"password"`
	Timeout         *common.Value          `xml:"timeout,omitempty"`
	VerifyServer    *bool                  `xml:"verifyServer,omitempty"`
	RowLimit        *common.Value          `xml:"rowLimit,omitempty"`
	Dataviews       []Dataview             `xml:"dataviews>dataview,omitempty"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Plugin without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Plugin) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

func (*Plugin) String() string { return "control-m" }

type Dataview struct {
	Name            string                 `xml:"name"`
	Parameters      []Parameter            `xml:"parameters,omitempty"`
	Columns         []string               `xml:"columns>column,omitempty"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Dataview without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Dataview) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

type Parameter struct {
	Parameter       string                 `xml:"parameter"`
	Criteria        string                 `xml:"criteria"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}
