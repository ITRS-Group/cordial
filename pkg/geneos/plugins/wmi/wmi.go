package wmi

import "encoding/xml"

type Plugin struct {
	XMLName           xml.Name `xml:"wmi" json:"-" yaml:"-"`
	Namespace         *string  `xml:"namespace,omitempty"`
	ShowQueryDuration *bool    `xml:"showQueryDuration,omitempty"`
	Context           *Context `xml:"context,omitempty"`
}

type Context struct {
	ProviderArchitecture *string `xml:"providerArchitecture,omitempty"`
}

func (*Plugin) String() string { return "wmi" }
