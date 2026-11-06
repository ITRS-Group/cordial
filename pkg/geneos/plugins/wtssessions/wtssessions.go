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
}

type Private struct {
	Debug *common.SingleLineStringVar `xml:"debug,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
