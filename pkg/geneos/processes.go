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
