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

// Probes TODO: Describe its purpose and intended use.
type Probes struct {
	XMLName        xml.Name        `xml:"probes" json:"-" yaml:"-"`
	Comment        string          `xml:",comment"`
	ProbeGroups    []ProbeGroup    `xml:"probeGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probegroup"`
	Probes         []Probe         `xml:"probe,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"probe"`
	VirtualProbes  []VirtualProbe  `xml:"virtualProbe,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"virtualprobe"`
	FloatingProbes []FloatingProbe `xml:"floatingProbe,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"floatingprobe"`
}

// ProbeGroup TODO: Describe its purpose and intended use.
type ProbeGroup struct {
	XMLName        xml.Name `xml:"probeGroup" json:"-" yaml:"-"`
	Comment        string   `xml:",comment"`
	Name           string   `xml:"name,attr"`
	Disabled       *bool    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ProbeInfo      `yaml:",inline" mapstructure:",squash"`
	LicensingGroup *string         `xml:"licensingGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"licensinggroup"`
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
	Port                 *int    `xml:"port,omitempty" json:"port,omitempty" yaml:",omitempty" mapstructure:"port"`
	Secure               *bool   `xml:"secure,omitempty" json:"secure,omitempty" yaml:",omitempty" mapstructure:"secure" geneos:"deprecated"`
	ConnectionSecurity   *string `xml:"connectionSecurity,omitempty" json:"connectionSecurity,omitempty" yaml:",omitempty" mapstructure:"connectionsecurity"`
	ProbeInfoWithoutPort `yaml:",inline" mapstructure:",squash"`
	Comment              string `xml:",comment"`
}

// ProbeInfoWithoutPort TODO: Describe its purpose and intended use.
type ProbeInfoWithoutPort struct {
	MaxLogFileSize          *int    `xml:"maxLogFileSize,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"maxlogfilesize"`
	ProcessListCommand      *string `xml:"processListCommand,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"processlistcommand"`
	WideProcessListCommand  *string `xml:"wideProcessListCommand,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"wideprocesslistcommand"`
	CacheProcessNames       *bool   `xml:"cacheProcessNames,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"cacheprocessnames"`
	MaxDatabaseConnections  *int    `xml:"maxDatabaseConnections,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"maxdatabaseconnections"`
	DisableSelfAnnounce     *bool   `xml:"disableSelfAnnounce,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"disableselfannounce"`
	CaseInsensitiveFilename *bool   `xml:"caseInsensitiveFilename,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"caseinsensitivefilename"`
	CommandTimeOut          *int    `xml:"commandTimeOut,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"commandtimeout"`
	FKMLockFilePath         *string `xml:"fkmLockFilePath,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"fkmlockfilepath"`
	MaxToolkitProcesses     *int    `xml:"maxToolkitProcesses,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"maxtoolkitprocesses"`
	RestAPIHTTPPort         *int    `xml:"restApiHttpPort,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"restapihttpport"`
	RestAPIHTTPSPort        *int    `xml:"restApiHttpsPort,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"restapihttpsport"`
	Timezone                *string `xml:"timezone,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"timezone"`
	CyberArkApplicationID   *string `xml:"cyberArkApplicationID,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"cyberarkapplicationid"`
	CyberArkSDKPathUnix     *string `xml:"cyberArkSdkPathUnix,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"cyberarksdkpathunix"`
	CyberArkSDKPathWindows  *string `xml:"cyberArkSdkPathWindows,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"cyberarksdkpathwindows"`
	Comment                 string  `xml:",comment"`
}

const (
	// ProbeTypeProbe TODO: Describe its purpose and intended use.
	ProbeTypeProbe int = iota
	// ProbeTypeFloating TODO: Describe its purpose and intended use.
	ProbeTypeFloating
	// ProbeTypeVirtual TODO: Describe its purpose and intended use.
	ProbeTypeVirtual
)

// Probe TODO: Describe its purpose and intended use.
type Probe struct {
	XMLName        xml.Name `xml:"probe" json:"-" yaml:"-"`
	Comment        string   `xml:",comment"`
	Type           int      `xml:"-" json:"-" yaml:"-"`
	Name           string   `xml:"name,attr" json:"name" yaml:"name"`
	Disabled       *bool    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Group          *string  `xml:"group,omitempty" json:",omitempty" yaml:",omitempty"`
	Description    *string  `xml:"description,omitempty" json:",omitempty" yaml:",omitempty"`
	DisplayName    *string  `xml:"displayName,omitempty" json:",omitempty" yaml:",omitempty"`
	LicensingGroup *string  `xml:"licensingGroup,omitempty" json:",omitempty" yaml:",omitempty"`
	Hostname       string   `xml:"hostname" json:"hostname" yaml:"hostname"`
	ProbeInfo      `json:",inline" yaml:",inline" mapstructure:",squash"`
}

// VirtualProbe TODO: Describe its purpose and intended use.
type VirtualProbe struct {
	XMLName     xml.Name `xml:"virtualProbe" json:"-" yaml:"-"`
	Comment     string   `xml:",comment"`
	Name        string   `xml:"name,attr" json:"name" yaml:"name"`
	Disabled    *bool    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Group       *string  `xml:"group,omitempty" json:",omitempty" yaml:",omitempty"`
	Description *string  `xml:"description,omitempty" json:",omitempty" yaml:",omitempty"`
	DisplayName *string  `xml:"displayName,omitempty" json:",omitempty" yaml:",omitempty"`
}

// FloatingProbe TODO: Describe its purpose and intended use.
type FloatingProbe struct {
	XMLName              xml.Name `xml:"floatingProbe" json:"-" yaml:"-"`
	Comment              string   `xml:",comment"`
	Name                 string   `xml:"name,attr" json:"name" yaml:"name"`
	Disabled             *bool    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Group                *string  `xml:"group,omitempty" json:",omitempty" yaml:",omitempty"`
	Description          *string  `xml:"description,omitempty" json:",omitempty" yaml:",omitempty"`
	DisplayName          *string  `xml:"displayName,omitempty" json:",omitempty" yaml:",omitempty"`
	LicensingGroup       *string  `xml:"licensingGroup,omitempty" json:",omitempty" yaml:",omitempty"`
	ProbeInfoWithoutPort `yaml:",inline" mapstructure:",squash"`
}
