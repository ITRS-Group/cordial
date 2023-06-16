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

type CPUPlugin struct {
	XMLName xml.Name `xml:"cpu" json:"-" yaml:"-"`
}

func (_ *CPUPlugin) String() string {
	return "cpu"
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

type Partition struct {
	Path  *SingleLineStringVar `xml:"path"`
	Alias *SingleLineStringVar `xml:"alias"`
}

type ExcludePartition struct {
	Path       string `xml:"path"`
	FileSystem string `xml:"fileSystem"`
}

func (_ *DiskPlugin) String() string {
	return "disk"
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
