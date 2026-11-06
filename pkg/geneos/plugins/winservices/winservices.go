package winservices

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName     xml.Name                     `xml:"win-services" json:"-" yaml:"-"`
	ServicesOld []common.SingleLineStringVar `xml:"services,omitempty" json:"servicesOld,omitempty" yaml:"servicesOld,omitempty"`
	Services    []Service                    `xml:"servicesNew>service,omitempty" json:"services,omitempty" yaml:"services,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }

type Service struct {
	ServiceName        *ServiceName        `xml:"serviceName,omitempty" json:"name,omitempty" yaml:"name,omitempty"`
	ServiceDescription *ServiceDescription `xml:"serviceDescription,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
}

type ServiceName struct {
	Basic                *common.SingleLineStringVar `xml:"basic>pattern,omitempty" json:"basic,omitempty" yaml:"basic,omitempty"`
	BasicShowUnavailable *bool                       `xml:"basic>showIfUnavailable,omitempty" json:"showIfUnavailable,omitempty" yaml:"showIfUnavailable,omitempty"`
	Regex                *common.Value               `xml:"regex,omitempty" json:"regex,omitempty" yaml:"regex,omitempty"`
}

type ServiceDescription struct {
	Basic *common.SingleLineStringVar `xml:"basic>pattern,omitempty" json:"basic,omitempty" yaml:"basic,omitempty"`
	Regex *common.Regex               `xml:"regex,omitempty" json:"regex,omitempty" yaml:"regex,omitempty"`
}
