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

type Types struct {
	XMLName    xml.Name    `xml:"types" json:"-" yaml:"-"`
	Types      []Type      `xml:"type,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"type"`
	TypeGroups []TypeGroup `xml:"typeGroup,omitempty" json:"typeGroup,omitempty" yaml:",omitempty" mapstructure:"typegroup"`
}

type TypeGroup struct {
	XMLName    xml.Name    `xml:"typeGroup" json:"-" yaml:"-"`
	Name       string      `xml:"name,attr"`
	Disabled   bool        `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Types      []Type      `xml:"type,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"type"`
	TypeGroups []TypeGroup `xml:"typeGroup,omitempty" json:"typeGroup,omitempty" yaml:",omitempty" mapstructure:"typegroup"`
}

type Type struct {
	XMLName     xml.Name     `xml:"type" json:"-" yaml:"-"`
	Name        string       `xml:"name,attr"`
	Disabled    bool         `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Environment *Reference   `xml:"environment,omitempty" json:",omitempty" yaml:",omitempty"`
	Vars        []Vars       `xml:"var,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"var"`
	Samplers    []SamplerRef `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
}

type SamplerRef struct {
	XMLName  xml.Name `xml:"sampler" json:"-" yaml:"-"`
	Name     string   `xml:"ref,attr" json:",omitempty" yaml:",omitempty"`
	Disabled bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
}
