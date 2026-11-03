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

// ControlMPlugin TODO: Describe its purpose and intended use.
type ControlMPlugin struct {
	XMLName   xml.Name           `xml:"control-m" json:"-" yaml:"-"`
	Host      *Host              `xml:"host"`
	Port      *Value             `xml:"port"`
	Username  *Value             `xml:"username"`
	Password  *Value             `xml:"password"`
	Timeout   *Value             `xml:"timeout,omitempty"`
	Dataviews []ControlMDataview `xml:"dataviews>dataview,omitempty"`
}

// String TODO: Document its behavior, inputs, and effects.
func (_ *ControlMPlugin) String() string {
	return "control-m"
}

// ControlMDataview TODO: Describe its purpose and intended use.
type ControlMDataview struct {
	Name       string              `xml:"name"`
	Parameters []ControlMParameter `xml:"parameters,omitempty"`
	Columns    []string            `xml:"columns>column,omitempty"`
}

// ControlMParameter TODO: Describe its purpose and intended use.
type ControlMParameter struct {
	Parameter string `xml:"parameter"`
	Criteria  string `xml:"criteria"`
}
