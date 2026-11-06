package xping

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName                        xml.Name          `xml:"x-ping" json:"-" yaml:"-"`
	TargetNode                     *common.Host      `xml:"targetNode,omitempty" json:"targetNode,omitempty" yaml:"targetNode,omitempty" geneos:"deprecated"`
	TargetNodes                    []common.Host     `xml:"targetNodes>targetNode,omitempty" json:"targetNodes,omitempty" yaml:"targetNodes,omitempty"`
	RecvInterfacesVar              *common.Reference `xml:"var-recvInterfaces,omitempty" json:"recvInterfaceVar,omitempty" yaml:"recvInterfaceVar,omitempty"`
	RecvInterfaces                 []common.Value    `xml:"recvInterfaces>data>recvInterface,omitempty" json:"recvInterface,omitempty" yaml:"recvInterface,omitempty" geneos:"deprecated"`
	SendInterface                  *common.Value     `xml:"sendInterface,omitempty" json:"sendInterface,omitempty" yaml:"sendInterface,omitempty"`
	SourceIPVar                    *common.Value     `xml:"var-sourceIP,omitempty" json:"sourceIPVar,omitempty" yaml:"sourceIPVar,omitempty"`
	SourceIP                       *common.IPAddress `xml:"sourceIP,omitempty" json:"sourceIP,omitempty" yaml:"sourceIP,omitempty" geneos:"deprecated"`
	TimeoutVar                     *common.Value     `xml:"var-timeout,omitempty" json:"timeoutVar,omitempty" yaml:"timeoutVar,omitempty"`
	Timeout                        *int              `xml:"timeout,omitempty" json:"timeout,omitempty" yaml:"timeout,omitempty" geneos:"deprecated"`
	AllowablePacketLossVar         *common.Value     `xml:"var-allowablePacketLoss,omitempty" json:"allowablePacketLossVar,omitempty" yaml:"allowablePacketLossVar,omitempty"`
	AllowablePacketLoss            *common.Value     `xml:"allowablePacketLoss,omitempty" json:"allowablePacketLoss,omitempty" yaml:"allowablePacketLoss,omitempty" geneos:"deprecated"`
	RecvPortVar                    *common.Value     `xml:"var-recvPort,omitempty" json:"recvPortVar,omitempty" yaml:"recvPortVar,omitempty"`
	RecvPort                       *int              `xml:"recvPort,omitempty" json:"recvPort,omitempty" yaml:"recvPort,omitempty" geneos:"deprecated"`
	Protocol                       *Protocol         `xml:"protocol,omitempty" json:"protocol,omitempty" yaml:"protocol,omitempty"`
	ICMPPayloadSizeVar             *common.Value     `xml:"var-icmpPayloadSize,omitempty" json:"icmpPayloadSizeVar,omitempty" yaml:"icmpPayloadSizeVar,omitempty"`
	ICMPPayloadSize                *int              `xml:"icmpPayloadSize,omitempty" json:"icmpPayloadSize,omitempty" yaml:"icmpPayloadSize,omitempty" geneos:"deprecated"`
	Private                        *Private          `xml:"private,omitempty" json:"private,omitempty" yaml:"private,omitempty"`
	CheckTargetHostHasChangedIPVar *common.Value     `xml:"var-checkTargetHostHasChangedIp,omitempty" json:"checkTargetHostHasChangedIpVar,omitempty" yaml:"checkTargetHostHasChangedIpVar,omitempty"`
	CheckTargetHostHasChangedIP    *bool             `xml:"checkTargetHostHasChangedIp,omitempty" json:"checkTargetHostHasChangedIp,omitempty" yaml:"checkTargetHostHasChangedIp,omitempty" geneos:"deprecated"`
}

type Protocol struct {
	ICMP *common.EmptyStruct `xml:"ICMP,omitempty"`
	TCP  *TCPMode            `xml:"TCP,omitempty"`
}

type TCPMode struct {
	Service *Service `xml:"service,omitempty"`
}

type Service struct {
	VarName *common.SingleLineStringVar `xml:"var-name,omitempty"`
	Name    *string                     `xml:"name,omitempty" geneos:"deprecated"`
	VarPort *common.Value               `xml:"var-port,omitempty"`
	Port    *int                        `xml:"port,omitempty" geneos:"deprecated"`
}

type Private struct {
	Debug    *bool `xml:"debug,omitempty"`
	PCEDebug *bool `xml:"pceDebug,omitempty"`
	PSDebug  *bool `xml:"psDebug,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
