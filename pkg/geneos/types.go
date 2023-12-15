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
