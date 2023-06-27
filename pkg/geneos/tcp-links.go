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
