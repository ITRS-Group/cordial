package wtssessions

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName       xml.Name                    `xml:"wts-sessions" json:"-" yaml:"-"`
	Counters      []common.Value              `xml:"counters>counter,omitempty"`
	MessageTitle  *common.SingleLineStringVar `xml:"messageTitle,omitempty"`
	DisableEvents *common.Value               `xml:"disableEvents,omitempty"`
	Private       *Private                    `xml:"private,omitempty"`
	Other         []common.RawXMLElement      `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Plugin without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Plugin) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

type Private struct {
	Debug *common.SingleLineStringVar `xml:"debug,omitempty"`
	Other []common.RawXMLElement      `xml:",any" json:",omitempty" yaml:",omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
