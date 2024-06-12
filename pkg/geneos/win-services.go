/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package geneos

import "encoding/xml"

type WinServicesPlugin struct {
	XMLName     xml.Name              `xml:"win-services" json:"-" yaml:"-"`
	ServicesOld []SingleLineStringVar `xml:"services,omitempty" json:"servicesOld,omitempty" yaml:"servicesOld,omitempty"`
	Services    []Service             `xml:"servicesNew>service,omitempty" json:"services,omitempty" yaml:"services,omitempty"`
}

func (p *WinServicesPlugin) String() string {
	return p.XMLName.Local
}

type Service struct {
	ServiceName        *ServiceName        `xml:"serviceName,omitempty" json:"name,omitempty" yaml:"name,omitempty"`
	ServiceDescription *ServiceDescription `xml:"serviceDescription,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
}

type ServiceName struct {
	Basic                *SingleLineStringVar `xml:"basic>pattern,omitempty" json:"basic,omitempty" yaml:"basic,omitempty"`
	BasicShowUnavailable bool                 `xml:"basic>showIfUnavailable,omitempty" json:"showIfUnavailable,omitempty" yaml:"showIfUnavailable,omitempty"`
	Regex                *Value               `xml:"regex,omitempty" json:"regex,omitempty" yaml:"regex,omitempty"`
}

type ServiceDescription struct {
	Basic *SingleLineStringVar `xml:"basic>pattern,omitempty" json:"basic,omitempty" yaml:"basic,omitempty"`
	Regex *Regex               `xml:"regex,omitempty" json:"regex,omitempty" yaml:"regex,omitempty"`
}
