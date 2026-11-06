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

// ManagedEntities TODO: Describe its purpose and intended use.
type ManagedEntities struct {
	XMLName             xml.Name             `xml:"managedEntities" json:"-" yaml:"-"`
	Comment             string               `xml:",comment"`
	Entities            []ManagedEntity      `xml:"managedEntity,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentity"`
	ManagedEntityGroups []ManagedEntityGroup `xml:"managedEntityGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentitygroup"`
}

// ManagedEntityGroup TODO: Describe its purpose and intended use.
type ManagedEntityGroup struct {
	XMLName             xml.Name `xml:"managedEntityGroup" json:"-" yaml:"-"`
	Comment             string   `xml:",comment"`
	Name                string   `xml:"name,attr"`
	Group               *string  `xml:"group,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Disabled            *bool    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ManagedEntityInfo   `yaml:",inline" mapstructure:",squash"`
	Entities            []ManagedEntity      `xml:"managedEntity,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentity"`
	ManagedEntityGroups []ManagedEntityGroup `xml:"managedEntityGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"managedentitygroup"`
}

// ManagedEntity TODO: Describe its purpose and intended use.
type ManagedEntity struct {
	XMLName           xml.Name        `xml:"managedEntity" json:"-" yaml:"-"`
	Comment           string          `xml:",comment"`
	Name              string          `xml:"name,attr"`
	Disabled          *bool           `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Group             *string         `xml:"group,omitempty" json:",omitempty" yaml:",omitempty"`
	Description       *string         `xml:"description,omitempty" json:",omitempty" yaml:",omitempty"`
	DisplayName       *string         `xml:"displayName,omitempty" json:",omitempty" yaml:",omitempty"`
	Probe             *Reference      `xml:"probe,omitempty" json:",omitempty" yaml:",omitempty"`
	FloatingProbe     *Reference      `xml:"floatingProbe,omitempty" json:",omitempty" yaml:",omitempty"`
	VirtualProbe      *Reference      `xml:"virtualProbe,omitempty" json:",omitempty" yaml:",omitempty"`
	Environment       *EnvironmentRef `xml:"environment,omitempty" json:",omitempty" yaml:",omitempty"`
	ManagedEntityInfo `yaml:",inline" mapstructure:",squash"`
	Samplers          []SamplerRef        `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerIncludes   []SamplerIncludeRef `xml:"samplerInclude,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplerinclude"`
	Banner            *string             `xml:"banner,omitempty" json:",omitempty" yaml:",omitempty" geneos:"deprecated"`
}

// ManagedEntityInfo TODO: Describe its purpose and intended use.
type ManagedEntityInfo struct {
	Attributes       []Attribute     `xml:"attribute,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"attribute"`
	RemoveTypes      *RemoveTypes    `xml:"removeTypes,omitempty" json:",omitempty" yaml:",omitempty"`
	RemoveSamplers   *RemoveSamplers `xml:"removeSamplers,omitempty" json:",omitempty" yaml:",omitempty"`
	AddTypes         *AddTypes       `xml:"addTypes,omitempty" json:",omitempty" yaml:",omitempty"`
	Vars             []Vars          `xml:"var,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"var"`
	ResolvedSamplers map[string]bool `xml:"-" json:",omitempty" yaml:",omitempty"` // map of "type:sampler" that exist at this point
	Comment          string          `xml:",comment"`
}

// RemoveTypes TODO: Describe its purpose and intended use.
type RemoveTypes struct {
	XMLName xml.Name  `xml:"removeTypes" json:"-" yaml:"-"`
	Comment string    `xml:",comment"`
	Types   []TypeRef `xml:"type,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"type"`
}

// RemoveSamplers TODO: Describe its purpose and intended use.
type RemoveSamplers struct {
	XMLName  xml.Name          `xml:"removeSamplers" json:"-" yaml:"-"`
	Comment  string            `xml:",comment"`
	Samplers []SamplerWithType `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty"`
}

// AddTypes TODO: Describe its purpose and intended use.
type AddTypes struct {
	XMLName xml.Name         `xml:"addTypes" json:"-" yaml:"-"`
	Comment string           `xml:",comment"`
	Types   []TypeRefWithEnv `xml:"type,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"type"`
}

// SamplerWithType TODO: Describe its purpose and intended use.
type SamplerWithType struct {
	XMLName xml.Name `xml:"sampler" json:"-" yaml:"-"`
	Comment string   `xml:",comment"`
	Sampler string   `xml:"ref,attr"`
	Type    TypeRef  `xml:"type"`
}

// Attribute TODO: Describe its purpose and intended use.
type Attribute struct {
	XMLName xml.Name `xml:"attribute" json:"-" yaml:"-"`
	Comment string   `xml:",comment"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:",innerxml" mapstructure:"#text"`
}

// GetKey satisfies the KeyedObject interface
func (a Attribute) GetKey() string {
	return a.Name
}

// TypeRef TODO: Describe its purpose and intended use.
type TypeRef struct {
	XMLName xml.Name `xml:"type" json:"-" yaml:"-"`
	Comment string   `xml:",comment"`
	Type    string   `xml:"ref,attr" json:",omitempty" yaml:",omitempty" mapstructure:"ref"`
}

// TypeRefWithEnv TODO: Describe its purpose and intended use.
type TypeRefWithEnv struct {
	XMLName     xml.Name        `xml:"type" json:"-" yaml:"-"`
	Comment     string          `xml:",comment"`
	Type        string          `xml:"ref,attr" json:",omitempty" yaml:",omitempty" mapstructure:"ref"`
	Environment *EnvironmentRef `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
}

// SamplerIncludeRef references a sampler include with an optional disabled override.
type SamplerIncludeRef struct {
	XMLName  xml.Name `xml:"samplerInclude" json:"-" yaml:"-"`
	Comment  string   `xml:",comment"`
	Name     string   `xml:"ref,attr" json:",omitempty" yaml:",omitempty" mapstructure:"ref"`
	Disabled *bool    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
}
