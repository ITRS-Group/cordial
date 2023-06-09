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

type ManagedEntities struct {
	XMLName             xml.Name             `xml:"managedEntities" json:"-" yaml:"-"`
	Entities            []ManagedEntity      `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentity"`
	ManagedEntityGroups []ManagedEntityGroup `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentitygroup"`
}

type ManagedEntityGroup struct {
	XMLName             xml.Name             `xml:"managedEntityGroup" json:"-" yaml:"-"`
	Name                string               `xml:"name,attr"`
	Disabled            bool                 `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Entities            []ManagedEntity      `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentity"`
	ManagedEntityGroups []ManagedEntityGroup `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentitygroup"`
	ManagedEntityInfo   `yaml:",inline" mapstructure:",squash"`
}

type ManagedEntity struct {
	XMLName           xml.Name        `xml:"managedEntity" json:"-" yaml:"-"`
	Name              string          `xml:"name,attr"`
	Disabled          bool            `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Probe             *Reference      `xml:"probe" json:",omitempty" yaml:",omitempty"`
	FloatingProbe     *Reference      `xml:"floatingprobe" json:",omitempty" yaml:",omitempty"`
	VirtualProbe      *Reference      `xml:"virtualprobe" json:",omitempty" yaml:",omitempty"`
	Environment       *EnvironmentRef `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
	ManagedEntityInfo `yaml:",inline" mapstructure:",squash"`
	Samplers          []Reference `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
}

type ManagedEntityInfo struct {
	Attributes []Attribute `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"attribute"`
	Vars       []Vars      `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"var"`
	AddTypes   struct {
		XMLName xml.Name  `xml:"addTypes" json:"-" yaml:"-"`
		Types   []TypeRef `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
	} `json:",omitempty" yaml:",omitempty"`
}

type Attribute struct {
	XMLName xml.Name `xml:"attribute" json:"-" yaml:"-"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:",innerxml" mapstructure:"#text"`
}

type TypeRef struct {
	XMLName     xml.Name        `xml:"type" json:"-" yaml:"-"`
	Type        string          `xml:"ref,attr"`
	Environment *EnvironmentRef `xml:",omitempty"`
}