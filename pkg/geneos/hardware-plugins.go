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

// CPUPlugin TODO: Describe its purpose and intended use.
type CPUPlugin struct {
	XMLName xml.Name `xml:"cpu" json:"-" yaml:"-"`
}

// String TODO: Document its behavior, inputs, and effects.
func (p *CPUPlugin) String() string {
	return p.XMLName.Local
}

// DeviceIOPlugin TODO: Describe its purpose and intended use.
type DeviceIOPlugin struct {
	XMLName xml.Name `xml:"deviceio" json:"-" yaml:"-"`
}

// String TODO: Document its behavior, inputs, and effects.
func (p *DeviceIOPlugin) String() string {
	return p.XMLName.Local
}

// DiskPlugin TODO: Describe its purpose and intended use.
type DiskPlugin struct {
	XMLName              xml.Name           `xml:"disk" json:"-" yaml:"-"`
	Partitions           []Partition        `xml:"partitions>partition,omitempty"`
	ExcludePartitions    []ExcludePartition `xml:"excludePartitions>excludePartition,omitempty"`
	AutoDetect           *Value             `xml:"autoDetect,omitempty"`
	MaximumPartitions    *Value             `xml:"maximumPartitions,omitempty"`
	CheckNFSPartitions   *Value             `xml:"checkNFSPartitions,omitempty"`
	MatchExactPartitions *Value             `xml:"matchExactPartitions,omitempty"`
}

// String TODO: Document its behavior, inputs, and effects.
func (p *DiskPlugin) String() string {
	return p.XMLName.Local
}

// Partition TODO: Describe its purpose and intended use.
type Partition struct {
	Path  *SingleLineStringVar `xml:"path"`
	Alias *SingleLineStringVar `xml:"alias"`
}

// ExcludePartition TODO: Describe its purpose and intended use.
type ExcludePartition struct {
	Path       string `xml:"path"`
	FileSystem string `xml:"fileSystem"`
}

// HardwarePlugin TODO: Describe its purpose and intended use.
type HardwarePlugin struct {
	XMLName xml.Name `xml:"hardware" json:"-" yaml:"-"`
}

// String TODO: Document its behavior, inputs, and effects.
func (p *HardwarePlugin) String() string {
	return p.XMLName.Local
}

// NetworkPlugin TODO: Describe its purpose and intended use.
type NetworkPlugin struct {
	XMLName xml.Name `xml:"network" json:"-" yaml:"-"`
}

// String TODO: Document its behavior, inputs, and effects.
func (p *NetworkPlugin) String() string {
	return p.XMLName.Local
}
