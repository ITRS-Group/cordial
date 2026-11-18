package wmi

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName           xml.Name               `xml:"wmi" json:"-" yaml:"-"`
	Namespace         *string                `xml:"namespace,omitempty"`
	ShowQueryDuration *bool                  `xml:"showQueryDuration,omitempty"`
	Context           *Context               `xml:"context,omitempty"`
	Other             []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes   []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

type Context struct {
	ProviderArchitecture *string                `xml:"providerArchitecture,omitempty"`
	Other                []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes      []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

func (*Plugin) String() string { return "wmi" }
