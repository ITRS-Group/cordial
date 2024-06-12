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

type ProcessesPlugin struct {
	XMLName                     xml.Name  `xml:"processes" json:"-" yaml:"-"`
	AdjustForLogicalCPUs        *Value    `xml:"adjustForLogicalCPUs,omitempty" json:",omitempty" yaml:",omitempty"`
	AdjustForLogicalCPUsSummary *Value    `xml:"adjustForLogicalCPUsSummary,omitempty" json:",omitempty" yaml:",omitempty"`
	Processes                   []Process `xml:"processes>process,omitempty" json:",omitempty" yaml:",omitempty"`
}

func (p *ProcessesPlugin) String() string {
	return p.XMLName.Local
}

type Process struct {
	XMLName           xml.Name              `xml:"process" json:"-" yaml:"-"`
	Data              *ProcessDescriptor    `xml:"data,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessDescriptor *ProcessDescriptorRef `xml:"processDescriptor,omitempty" json:",omitempty" yaml:",omitempty"`
}

type ProcessDescriptorRef struct {
	Name string `xml:"ref,attr" json:"name" yaml:"name"`
}

type ProcessDescriptors struct {
	XMLName                 xml.Name                 `xml:"processDescriptors" json:"-" yaml:"-"`
	ProcessDescriptors      []ProcessDescriptor      `xml:"data,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessDescriptorGroups []ProcessDescriptorGroup `xml:"processDescriptorGroup,omitempty" json:",omitempty" yaml:",omitempty"`
}

type ProcessDescriptorGroup struct {
	XMLName                 xml.Name                 `xml:"processDescriptorGroup" json:"-" yaml:"-"`
	Name                    string                   `xml:"name,attr" json:"name" yaml:"name"`
	Disabled                bool                     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessDescriptors      []ProcessDescriptor      `xml:"processDescriptor,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessDescriptorGroups []ProcessDescriptorGroup `xml:"processDescriptorGroup,omitempty" json:",omitempty" yaml:",omitempty"`
}

type ProcessDescriptor struct {
	// no struct XMLName as we can get here via two routes - "data" or "processDescriptor"
	// XMLName  xml.Name             `xml:"processDescriptor" json:"-" yaml:"-"`
	Name     string               `xml:"name,attr" json:"name" yaml:"name"`
	Disabled bool                 `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Alias    *SingleLineStringVar `xml:"alias" json:"alias" yaml:"alias"`
	Start    *SingleLineStringVar `xml:"start,omitempty" json:"start,omitempty" yaml:"start,omitempty"`
	Stop     *SingleLineStringVar `xml:"stop,omitempty" json:"stop,omitempty" yaml:"stop,omitempty"`
	LogFile  *SingleLineStringVar `xml:"logFile,omitempty" json:"logfile,omitempty" yaml:"logfile,omitempty"`
}
