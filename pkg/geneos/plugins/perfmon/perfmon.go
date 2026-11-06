package perfmon

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName                xml.Name                    `xml:"perfmon" json:"-" yaml:"-"`
	LanguageID             *common.SingleLineStringVar `xml:"languageID,omitempty"`
	ObjectRefreshThreshold *common.Value               `xml:"objectRefreshThreshold,omitempty"`
	Private                *Private                    `xml:"private,omitempty"`
}

type Private struct {
	Debug *bool `xml:"debug,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
