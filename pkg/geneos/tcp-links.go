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

// TCP-Links

type TCPLinksPlugin struct {
	XMLName      xml.Name              `xml:"tcp-links" json:"-" yaml:"-"`
	LocalPorts   []SingleLineStringVar `xml:"localPorts>port,omitempty" json:"localPorts,omitempty" yaml:"localPorts,omitempty"`
	RemotePorts  []SingleLineStringVar `xml:"remotePorts>port,omitempty" json:"remotePorts,omitempty" yaml:"remotePorts,omitempty"`
	Command      *SingleLineStringVar  `xml:"command,omitempty" json:"command,omitempty" xml:"command,omitempty"`
	NameContents *NameContents         `xml:"nameContents,omitempty" json:"nameContents,omitempty" yaml:"nameContents,omitempty"`
}

func (p *TCPLinksPlugin) String() string {
	return p.XMLName.Local
}

type NameContents struct {
	ShowLocalAddress  *Value `xml:"showLocalAddress,omitempty" json:"showLocalAddress,omitempty" yaml:"showLocalAddress,omitempty"`
	ShowLocalPort     *Value `xml:"showLocalPort,omitempty" json:"showLocalPort,omitempty" yaml:"showLocalPort,omitempty"`
	ShowRemoteAddress *Value `xml:"showRemoteAddress,omitempty" json:"showRemoteAddress,omitempty" yaml:"showRemoteAddress,omitempty"`
	ShowRemotePort    *Value `xml:"showRemotePort,omitempty" json:"showRemotePort,omitempty" yaml:"showRemotePort,omitempty"`
}
