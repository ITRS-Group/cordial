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

type ToolkitPlugin struct {
	XMLName              xml.Name               `xml:"toolkit" json:"-" yaml:"-"`
	SamplerScript        *SingleLineStringVar   `xml:"samplerScript"`
	SamplerTimeout       *Value                 `xml:"scriptTimeout,omitempty"`
	EnvironmentVariables *[]EnvironmentVariable `xml:"environmentVariables>variable"`
}

func (p *ToolkitPlugin) String() string {
	return p.XMLName.Local
}

type EnvironmentVariable struct {
	XMLName xml.Name             `xml:"variable"  json:"-" yaml:"-"`
	Name    string               `xml:"name"`
	Value   *SingleLineStringVar `xml:"value"`
}
