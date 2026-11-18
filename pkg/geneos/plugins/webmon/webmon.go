package webmon

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName             xml.Name               `xml:"web-mon" json:"-" yaml:"-"`
	Host                *common.Value          `xml:"host"`
	ServerURL           *common.Value          `xml:"serverUrl,omitempty"`
	Display             *Display               `xml:"display,omitempty"`
	MaxInitialSample    *int                   `xml:"maxInitialSample,omitempty" geneos:"deprecated"`
	VarMaxInitialSample *common.Value          `xml:"var-maxInitialSample,omitempty"`
	MaxServerDuration   *common.Value          `xml:"maxServerDuration,omitempty"`
	GSSNegotiate        *GSSNegotiate          `xml:"gssnegotiate,omitempty"`
	Other               []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes     []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

type Display struct {
	VarDisplayMode  *common.Value          `xml:"var-displayMode,omitempty"`
	DisplayMode     *common.Value          `xml:"displayMode,omitempty" geneos:"deprecated"`
	VarDisplayURL   *common.Value          `xml:"var-displayURL,omitempty"`
	DisplayURL      *common.Value          `xml:"displayURL,omitempty" geneos:"deprecated"`
	DisplayHeaders  []string               `xml:"displayHeaders>displayHeader,omitempty"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Display without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Display) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

type GSSNegotiate struct {
	Principal       *common.SingleLineStringVar `xml:"principal"`
	Keytab          *common.SingleLineStringVar `xml:"keytab"`
	Other           []common.RawXMLElement      `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr                  `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
