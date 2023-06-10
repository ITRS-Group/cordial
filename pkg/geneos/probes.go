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
	ProbeGroups    []ProbeGroup    `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probegroup"`
	Probes         []Probe         `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probe"`
	VirtualProbes  []VirtualProbe  `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"virtualprobe"`
	FloatingProbes []FloatingProbe `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"floatingprobe"`
}

type ProbeGroup struct {
	XMLName        xml.Name `xml:"probeGroup" json:"-" yaml:"-"`
	Name           string   `xml:"name,attr"`
	Disabled       bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ProbeInfo      `yaml:",inline" mapstructure:",squash"`
	ProbeGroups    []ProbeGroup    `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probegroup"`
	Probes         []Probe         `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probe"`
	VirtualProbes  []VirtualProbe  `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"virtualprobe"`
	FloatingProbes []FloatingProbe `xml:",omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"floatingprobe"`
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
	Name      string   `xml:"name,attr"`
	Disabled  bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Hostname  string   `xml:"hostname" json:"hostname"`
	ProbeInfo `yaml:",inline" mapstructure:",squash"`
}

type VirtualProbe struct {
	XMLName  xml.Name `xml:"virtualProbe" json:"-" yaml:"-"`
	Name     string   `xml:"name.attr"`
	Disabled bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
}

type FloatingProbe struct {
	XMLName              xml.Name `xml:"floatingProbe" json:"-" yaml:"-"`
	Name                 string   `xml:"name.attr" json:"name"`
	Disabled             bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ProbeInfoWithoutPort `yaml:",inline" mapstructure:",squash"`
}
