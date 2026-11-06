package fixanalyser2

import "encoding/xml"

type Plugin struct {
	XMLName xml.Name `xml:"fix-analyser2" json:"-" yaml:"-"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
