package geneos

import "encoding/xml"

type ProcessesPlugin struct {
	// XMLName xml.Name `xml:"processes"`
	// ...
	AdjustForLogicalCPUs *Value    `xml:"adjustForLogicalCPUs,omitempty" json:",omitempty" yaml:",omitempty"`
	Processes            []Process `xml:"processes>processes>process,omitempty" json:",omitempty" yaml:",omitempty"`
	Rest                 string    `xml:",any"`
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
	Alias *SingleLineString `xml:"alias"`
}
