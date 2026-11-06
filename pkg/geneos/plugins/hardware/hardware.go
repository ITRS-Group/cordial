package hardware

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type CPUPlugin struct {
	XMLName     xml.Name        `xml:"cpu" json:"-" yaml:"-"`
	Temperature *CPUTemperature `xml:"temperature,omitempty" geneos:"deprecated"`
	Solaris     *CPUSolaris     `xml:"solaris,omitempty" geneos:"deprecated"`
}

type CPUTemperature struct {
	Show    *common.Value               `xml:"show,omitempty"`
	Command *common.SingleLineStringVar `xml:"command,omitempty"`
}

type CPUSolaris struct {
	ShowPhysicalSummaryRows *common.EmptyStruct `xml:"showPhysicalSummaryRows,omitempty"`
}

func (p *CPUPlugin) String() string { return p.XMLName.Local }

type DeviceIOPlugin struct {
	XMLName    xml.Name                     `xml:"deviceio" json:"-" yaml:"-"`
	Command    *common.SingleLineStringVar  `xml:"command,omitempty" geneos:"deprecated"`
	DeviceList []common.SingleLineStringVar `xml:"deviceList>device,omitempty"`
}

func (p *DeviceIOPlugin) String() string { return p.XMLName.Local }

type DiskPlugin struct {
	XMLName              xml.Name           `xml:"disk" json:"-" yaml:"-"`
	Partitions           []Partition        `xml:"partitions>partition,omitempty"`
	ExcludePartitions    []ExcludePartition `xml:"excludePartitions>excludePartition,omitempty"`
	AutoDetect           *common.Value      `xml:"autoDetect,omitempty"`
	MaximumPartitions    *common.Value      `xml:"maximumPartitions,omitempty"`
	CheckNFSPartitions   *common.Value      `xml:"checkNFSPartitions,omitempty"`
	MatchExactPartitions *common.Value      `xml:"matchExactPartitions,omitempty"`
}

func (p *DiskPlugin) String() string { return p.XMLName.Local }

type Partition struct {
	Path  *common.SingleLineStringVar `xml:"path"`
	Alias *common.SingleLineStringVar `xml:"alias"`
}

type ExcludePartition struct {
	Path       string `xml:"path"`
	FileSystem string `xml:"fileSystem"`
}

type HardwarePlugin struct {
	XMLName xml.Name `xml:"hardware" json:"-" yaml:"-"`
}

func (p *HardwarePlugin) String() string { return p.XMLName.Local }

type NetworkPlugin struct {
	XMLName              xml.Name      `xml:"network" json:"-" yaml:"-"`
	ShowVirtualInterface *common.Value `xml:"showVirtualInterface,omitempty"`
	ShowLoopback         *common.Value `xml:"showLoopback,omitempty"`
	ShowMBits            *common.Value `xml:"showMBits,omitempty"`
	ShowAbsolute         *common.Value `xml:"showAbsolute,omitempty"`
	ShowUnconfiguredNics *common.Value `xml:"showUnconfiguredNics,omitempty"`
}

func (p *NetworkPlugin) String() string { return p.XMLName.Local }
