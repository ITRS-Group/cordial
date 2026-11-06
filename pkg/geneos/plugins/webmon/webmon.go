package webmon

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName             xml.Name      `xml:"web-mon" json:"-" yaml:"-"`
	Host                *common.Value `xml:"host"`
	ServerURL           *common.Value `xml:"serverUrl,omitempty"`
	Display             *Display      `xml:"display,omitempty"`
	MaxInitialSample    *int          `xml:"maxInitialSample,omitempty" geneos:"deprecated"`
	VarMaxInitialSample *common.Value `xml:"var-maxInitialSample,omitempty"`
	MaxServerDuration   *common.Value `xml:"maxServerDuration,omitempty"`
	GSSNegotiate        *GSSNegotiate `xml:"gssnegotiate,omitempty"`
}

type Display struct {
	VarDisplayMode *common.Value `xml:"var-displayMode,omitempty"`
	DisplayMode    *common.Value `xml:"displayMode,omitempty" geneos:"deprecated"`
	VarDisplayURL  *common.Value `xml:"var-displayURL,omitempty"`
	DisplayURL     *common.Value `xml:"displayURL,omitempty" geneos:"deprecated"`
	DisplayHeaders []string      `xml:"displayHeaders>displayHeader,omitempty"`
}

type GSSNegotiate struct {
	Principal *common.SingleLineStringVar `xml:"principal"`
	Keytab    *common.SingleLineStringVar `xml:"keytab"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
