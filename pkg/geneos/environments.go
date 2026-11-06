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

// EnvironmentRef TODO: Describe its purpose and intended use.
type EnvironmentRef struct {
	XMLName xml.Name `xml:"environment" json:"-" yaml:"-"`
	Comment string   `xml:",comment"`
	Name    string   `xml:"ref,attr" json:",omitempty" yaml:",omitempty"`
}

// Environments TODO: Describe its purpose and intended use.
type Environments struct {
	XMLName      xml.Name           `xml:"environments" json:"-" yaml:"-"`
	Comment      string             `xml:",comment"`
	Groups       []EnvironmentGroup `xml:"environmentGroup,omitempty" json:",omitempty" yaml:",omitempty"`
	Environments []Environment      `xml:"environment,omitempty" json:",omitempty" yaml:",omitempty"`
}

// EnvironmentGroup TODO: Describe its purpose and intended use.
type EnvironmentGroup struct {
	XMLName      xml.Name      `xml:"environmentGroup" json:"-" yaml:"-"`
	Comment      string        `xml:",comment"`
	Name         string        `xml:"name,attr"`
	Environments []Environment `xml:"environment,omitempty" json:",omitempty" yaml:",omitempty"`
}

// Environment TODO: Describe its purpose and intended use.
type Environment struct {
	XMLName      xml.Name      `xml:"environment" json:"-" yaml:"-"`
	Comment      string        `xml:",comment"`
	Name         string        `xml:"name,attr"`
	Environments []Environment `xml:"environment,omitempty" json:",omitempty" yaml:",omitempty"`
	Vars         []Vars        `xml:"var,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"var"`
}
