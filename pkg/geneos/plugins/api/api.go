package api

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName          xml.Name                    `xml:"api" json:"-" yaml:"-"`
	ExpectViews      *ExpectViews                `xml:"expectViews,omitempty"`
	ExpectViewDelay  *common.Value               `xml:"expectViewDelay,omitempty"`
	Parameters       []Parameter                 `xml:"parameters>parameter"`
	SummaryView      *common.SingleLineStringVar `xml:"showSummaryView>always>viewName,omitempty"`
	SummaryViewNever *common.EmptyStruct         `xml:"showSummaryView>never,omitempty"`
	Private          *Private                    `xml:"private,omitempty"`
	Other            []common.RawXMLElement      `xml:",any" json:",omitempty" yaml:",omitempty"`
}

func (*Plugin) String() string { return "api" }

type ExpectViews struct {
	Views []ExpectView           `xml:"expectView,omitempty"`
	Other []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

type ExpectView struct {
	View  *common.SingleLineStringVar `xml:"view"`
	Group *common.SingleLineStringVar `xml:"group"`
	Other []common.RawXMLElement      `xml:",any" json:",omitempty" yaml:",omitempty"`
}

type Private struct {
	DebugFlags []string               `xml:"debugFlags>flag,omitempty"`
	Other      []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Private without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Private) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

type Parameter struct {
	Name  string                      `xml:"name"`
	Value *common.SingleLineStringVar `xml:"value"`
	Other []common.RawXMLElement      `xml:",any" json:",omitempty" yaml:",omitempty"`
}

type StreamsPlugin struct {
	XMLName    xml.Name               `xml:"api-streams" json:"-" yaml:"-"`
	Streams    *Streams               `xml:"streams"`
	BufferSize *common.Value          `xml:"bufferSize,omitempty"`
	CreateView *common.Value          `xml:"createView,omitempty"`
	Other      []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

func (*StreamsPlugin) String() string { return "api-streams" }

type Streams struct {
	XMLName xml.Name                      `xml:"streams" json:"-" yaml:"-"`
	Stream  []*common.SingleLineStringVar `xml:"stream"`
	Other   []common.RawXMLElement        `xml:",any" json:",omitempty" yaml:",omitempty"`
}
