/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
	Regex                *Regex               `xml:"regex,omitempty" json:"regex,omitempty" yaml:"regex,omitempty"`
}

type ServiceDescription struct {
	Basic *SingleLineStringVar `xml:"basic>pattern,omitempty" json:"basic,omitempty" yaml:"basic,omitempty"`
	Regex *Regex               `xml:"regex,omitempty" json:"regex,omitempty" yaml:"regex,omitempty"`
}
