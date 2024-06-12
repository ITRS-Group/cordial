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
