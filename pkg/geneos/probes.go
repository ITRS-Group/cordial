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

// These definitions for the probe are for the gateway setup
//
// The number of tags on each fields is needed for:
//
//   * xml and mapstructure for unmarshalling from source and config respectively
//   * json and yaml for marshalling for output
//
// Any (bool) value that can be inherited and then overridden hence
// needs a concept of "unset" must be a pointer. This is mostly for
// booleans and ints that can be zero, in groups.

type Probes struct {
	XMLName        xml.Name        `xml:"probes" json:"-" yaml:"-"`
	ProbeGroups    []ProbeGroup    `xml:"probeGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probegroup"`
	Probes         []Probe         `xml:"probe,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probe"`
	VirtualProbes  []VirtualProbe  `xml:"virtualProbe,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"virtualprobe"`
	FloatingProbes []FloatingProbe `xml:"floatingProbe,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"floatingprobe"`
}

type ProbeGroup struct {
	XMLName        xml.Name `xml:"probeGroup" json:"-" yaml:"-"`
	Name           string   `xml:"name,attr"`
	Disabled       bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ProbeInfo      `yaml:",inline" mapstructure:",squash"`
	ProbeGroups    []ProbeGroup    `xml:"probeGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probegroup"`
	Probes         []Probe         `xml:"probe,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probe"`
	VirtualProbes  []VirtualProbe  `xml:"virtualProbe,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"virtualprobe"`
	FloatingProbes []FloatingProbe `xml:"floatingProbe,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"floatingprobe"`
}

// ProbeInfo is embedded in groups and probes
//
// The mapstructure tags are needed otherwise the embedding uses the
// structure name, not the field
type ProbeInfo struct {
	Port                 int   `xml:"port,omitempty" json:"port,omitempty" yaml:",omitempty" mapstructure:"port"`
	Secure               *bool `xml:"secure,omitempty" json:"secure,omitempty" yaml:",omitempty" mapstructure:"secure"`
	ProbeInfoWithoutPort `yaml:",inline" mapstructure:",squash"`
}

type ProbeInfoWithoutPort struct {
	CommandTimeOut         int `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"commandtimeout"`
	MaxDatabaseConnections int `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"maxdatabaseconnections"`
	MaxToolkitProcesses    int `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"maxtoolkitprocesses"`
}

const (
	ProbeTypeProbe int = iota
	ProbeTypeFloating
	ProbeTypeVirtual
)

type Probe struct {
	XMLName   xml.Name `xml:"probe" json:"-" yaml:"-"`
	Type      int      `xml:"-" json:"-" yaml:"-"`
	Name      string   `xml:"name,attr" json:"name" yaml:"name"`
	Disabled  bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Hostname  string   `xml:"hostname" json:"hostname" yaml:"hostname"`
	ProbeInfo `json:",inline" yaml:",inline" mapstructure:",squash"`
}

type VirtualProbe struct {
	XMLName  xml.Name `xml:"virtualProbe" json:"-" yaml:"-"`
	Name     string   `xml:"name,attr" json:"name" yaml:"name"`
	Disabled bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
}

type FloatingProbe struct {
	XMLName              xml.Name `xml:"floatingProbe" json:"-" yaml:"-"`
	Name                 string   `xml:"name.attr" json:"name" yaml:"name"`
	Disabled             bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ProbeInfoWithoutPort `yaml:",inline" mapstructure:",squash"`
}
