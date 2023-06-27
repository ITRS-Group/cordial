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

import (
	"encoding/xml"
)

type XPingPlugin struct {
	XMLName           xml.Name   `xml:"x-ping" json:"-" yaml:"-"`
	TargetNodes       []Host     `xml:"targetNodes>targetNode,omitempty" json:"targetNode,omitempty" yaml:"targetNode,omitempty"`
	RecvInterfacesVar *Reference `xml:"recvInterfaces>var,omitempty" json:"recvInterfaceVar,omitempty" yaml:"recvInterfaceVar,omitempty"`
	RecvInterfaces    []Value    `xml:"recvInterfaces>data>recvInterface,omitempty" json:"recvInterface,omitempty" yaml:"recvInterface,omitempty"`
	SendInterface     *Value     `xml:"sendInterface" json:"sendInterface" yaml:"sendInterface"`
}

func (p *XPingPlugin) String() string {
	return p.XMLName.Local
}
