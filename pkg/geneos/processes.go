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
	AdjustForLogicalCPUs *Value    `xml:"adjustForLogicalCPUs,omitempty" json:",omitempty" yaml:",omitempty"`
	Processes            []Process `xml:"processes>processes>process,omitempty" json:",omitempty" yaml:",omitempty"`
	Rest                 string    `xml:",any"`
}

func (_ *ProcessesPlugin) String() string {
	return "processes"
}

type Process struct {
	Data              *ProcessDescriptor    `xml:"data,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessDescriptor *ProcessDescriptorRef `xml:"processDescriptor,omitempty" json:",omitempty" yaml:",omitempty"`
}

type ProcessDescriptorRef struct {
	Name string `xml:"ref,attr"`
}

type ProcessDescriptors struct {
	XMLName xml.Name `xml:"processDescriptors"`
}

type ProcessDescriptorGroup struct {
	XMLName            xml.Name            `xml:"processDescriptorGroup"`
	ProcessDescriptors []ProcessDescriptor `xml:"processDescriptor,omitempty" json:",omitempty" yaml:",omitempty"`
}

type ProcessDescriptor struct {
	Alias *SingleLineStringVar `xml:"alias"`
}
