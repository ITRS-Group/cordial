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

type CPUPlugin struct {
	XMLName xml.Name `xml:"cpu" json:"-" yaml:"-"`
}

func (p *CPUPlugin) String() string {
	return p.XMLName.Local
}

type DeviceIOPlugin struct {
	XMLName xml.Name `xml:"deviceio" json:"-" yaml:"-"`
}

func (p *DeviceIOPlugin) String() string {
	return p.XMLName.Local
}

type DiskPlugin struct {
	XMLName              xml.Name           `xml:"disk" json:"-" yaml:"-"`
	Partitions           []Partition        `xml:"partitions>partition,omitempty"`
	ExcludePartitions    []ExcludePartition `xml:"excludePartitions>excludePartition,omitempty"`
	AutoDetect           *Value             `xml:"autoDetect,omitempty"`
	MaximumPartitions    *Value             `xml:"maximumPartitions,omitempty"`
	CheckNFSPartitions   *Value             `xml:"checkNFSPartitions,omitempty"`
	MatchExactPartitions *Value             `xml:"matchExactPartitions,omitempty"`
}

func (p *DiskPlugin) String() string {
	return p.XMLName.Local
}

type Partition struct {
	Path  *SingleLineStringVar `xml:"path"`
	Alias *SingleLineStringVar `xml:"alias"`
}

type ExcludePartition struct {
	Path       string `xml:"path"`
	FileSystem string `xml:"fileSystem"`
}

type HardwarePlugin struct {
	XMLName xml.Name `xml:"hardware" json:"-" yaml:"-"`
}

func (p *HardwarePlugin) String() string {
	return p.XMLName.Local
}

type NetworkPlugin struct {
	XMLName xml.Name `xml:"network" json:"-" yaml:"-"`
}

func (p *NetworkPlugin) String() string {
	return p.XMLName.Local
}
