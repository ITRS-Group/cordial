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

type ManagedEntities struct {
	XMLName             xml.Name             `xml:"managedEntities" json:"-" yaml:"-"`
	Entities            []ManagedEntity      `xml:"managedEntity,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentity"`
	ManagedEntityGroups []ManagedEntityGroup `xml:"managedEntityGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentitygroup"`
}

type ManagedEntityGroup struct {
	XMLName             xml.Name `xml:"managedEntityGroup" json:"-" yaml:"-"`
	Name                string   `xml:"name,attr"`
	Disabled            bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ManagedEntityInfo   `yaml:",inline" mapstructure:",squash"`
	Entities            []ManagedEntity      `xml:"managedEntity,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentity"`
	ManagedEntityGroups []ManagedEntityGroup `xml:"managedEntityGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentitygroup"`
}

type ManagedEntity struct {
	XMLName           xml.Name        `xml:"managedEntity" json:"-" yaml:"-"`
	Name              string          `xml:"name,attr"`
	Disabled          bool            `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Probe             *Reference      `xml:"probe" json:",omitempty" yaml:",omitempty"`
	FloatingProbe     *Reference      `xml:"floatingProbe" json:",omitempty" yaml:",omitempty"`
	VirtualProbe      *Reference      `xml:"virtualProbe" json:",omitempty" yaml:",omitempty"`
	Environment       *EnvironmentRef `xml:"environment,omitempty" json:",omitempty" yaml:",omitempty"`
	ManagedEntityInfo `yaml:",inline" mapstructure:",squash"`
	Samplers          []SamplerRef `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
}

type ManagedEntityInfo struct {
	Attributes       []Attribute     `xml:"attribute,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"attribute"`
	RemoveTypes      *RemoveTypes    `xml:"removeTypes,omitempty" json:",omitempty" yaml:",omitempty"`
	RemoveSamplers   *RemoveSamplers `xml:"removeSamplers,omitempty" json:",omitempty" yaml:",omitempty"`
	AddTypes         *AddTypes       `xml:"addTypes,omitempty" json:",omitempty" yaml:",omitempty"`
	Vars             []Vars          `xml:"var,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"var"`
	ResolvedSamplers map[string]bool `xml:"-" json:",omitempty" yaml:",omitempty"` // map of "type:sampler" that exist at this point
}

type RemoveTypes struct {
	XMLName xml.Name  `xml:"removeTypes" json:"-" yaml:"-"`
	Types   []TypeRef `xml:"type,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"type"`
}

type RemoveSamplers struct {
	XMLName  xml.Name          `xml:"removeSamplers" json:"-" yaml:"-"`
	Samplers []SamplerWithType `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty"`
}

type AddTypes struct {
	XMLName xml.Name         `xml:"addTypes" json:"-" yaml:"-"`
	Types   []TypeRefWithEnv `xml:"type,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"type"`
}

type SamplerWithType struct {
	XMLName xml.Name `xml:"sampler" json:"-" yaml:"-"`
	Sampler string   `xml:"ref,attr"`
	Type    TypeRef  `xml:"type"`
}

type Attribute struct {
	XMLName xml.Name `xml:"attribute" json:"-" yaml:"-"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:",innerxml" mapstructure:"#text"`
}

// GetKey satisfies the KeyedObject interface
func (a Attribute) GetKey() string {
	return a.Name
}

type TypeRef struct {
	XMLName xml.Name `xml:"type" json:"-" yaml:"-"`
	Type    string   `xml:"ref,attr" json:",omitempty" yaml:",omitempty" mapstructure:"ref"`
}

type TypeRefWithEnv struct {
	XMLName     xml.Name        `xml:"type" json:"-" yaml:"-"`
	Type        string          `xml:"ref,attr" json:",omitempty" yaml:",omitempty" mapstructure:"ref"`
	Environment *EnvironmentRef `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
}
