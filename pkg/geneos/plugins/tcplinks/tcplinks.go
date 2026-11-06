package tcplinks

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName       xml.Name                     `xml:"tcp-links" json:"-" yaml:"-"`
	LocalPorts    []common.SingleLineStringVar `xml:"localPorts>port,omitempty" json:"localPorts,omitempty" yaml:"localPorts,omitempty"`
	RemotePorts   []common.SingleLineStringVar `xml:"remotePorts>port,omitempty" json:"remotePorts,omitempty" yaml:"remotePorts,omitempty"`
	Command       *common.SingleLineStringVar  `xml:"command,omitempty" json:"command,omitempty"`
	NameContents  *NameContents                `xml:"nameContents,omitempty" json:"nameContents,omitempty" yaml:"nameContents,omitempty"`
	ServiceTokens []common.SingleLineStringVar `xml:"serviceTokens>serviceToken,omitempty" json:"serviceTokens,omitempty" yaml:"serviceTokens,omitempty"`
	ShowAllStates *common.Value                `xml:"showAllStates,omitempty" json:"showAllStates,omitempty" yaml:"showAllStates,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }

type NameContents struct {
	ShowLocalAddress  *common.Value `xml:"showLocalAddress,omitempty" json:"showLocalAddress,omitempty" yaml:"showLocalAddress,omitempty"`
	ShowLocalPort     *common.Value `xml:"showLocalPort,omitempty" json:"showLocalPort,omitempty" yaml:"showLocalPort,omitempty"`
	ShowRemoteAddress *common.Value `xml:"showRemoteAddress,omitempty" json:"showRemoteAddress,omitempty" yaml:"showRemoteAddress,omitempty"`
	ShowRemotePort    *common.Value `xml:"showRemotePort,omitempty" json:"showRemotePort,omitempty" yaml:"showRemotePort,omitempty"`
}
