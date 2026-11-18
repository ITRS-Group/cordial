package fixanalyser2

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName xml.Name `xml:"fix-analyser2" json:"-" yaml:"-"`

	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
