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
	XMLName       xml.Name        `xml:"probes" json:"-" yaml:"-"`
	ProbeGroup    []ProbeGroup    `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
	Probe         []Probe         `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
	VirtualProbe  []VirtualProbe  `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
	FloatingProbe []FloatingProbe `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
}

type ProbeGroup struct {
	XMLName       xml.Name `xml:"probeGroup" json:"-" yaml:"-"`
	Name          string   `xml:"name,attr" json:"name"`
	Disabled      bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ProbeInfo     `yaml:",inline" mapstructure:",squash"`
	ProbeGroup    []ProbeGroup    `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
	Probe         []Probe         `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
	VirtualProbe  []VirtualProbe  `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
	FloatingProbe []FloatingProbe `xml:",omitempty" json:",omitempty" yaml:",omitempty"`
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

type Probe struct {
	XMLName   xml.Name `xml:"probe" json:"-" yaml:"-"`
	Name      string   `xml:"name,attr" json:"name"`
	Disabled  bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Hostname  string   `xml:"hostname" json:"hostname"`
	ProbeInfo `yaml:",inline" mapstructure:",squash"`
}

type VirtualProbe struct {
	XMLName  xml.Name `xml:"virtualProbe" json:"-" yaml:"-"`
	Name     string   `xml:"name.attr" json:"name"`
	Disabled bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
}

type FloatingProbe struct {
	XMLName              xml.Name `xml:"floatingProbe" json:"-" yaml:"-"`
	Name                 string   `xml:"name.attr" json:"name"`
	Disabled             bool     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ProbeInfoWithoutPort `yaml:",inline" mapstructure:",squash"`
}
