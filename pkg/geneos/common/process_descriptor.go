package common

type ProcessDescriptors struct {
	ProcessDescriptors      []ProcessDescriptor      `xml:"data,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessDescriptorGroups []ProcessDescriptorGroup `xml:"processDescriptorGroup,omitempty" json:",omitempty" yaml:",omitempty"`
}

type ProcessDescriptorGroup struct {
	Name                    string                   `xml:"name,attr" json:"name" yaml:"name"`
	Disabled                bool                     `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessDescriptors      []ProcessDescriptor      `xml:"processDescriptor,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessDescriptorGroups []ProcessDescriptorGroup `xml:"processDescriptorGroup,omitempty" json:",omitempty" yaml:",omitempty"`
}

type ProcessDescriptor struct {
	Name     string               `xml:"name,attr" json:"name" yaml:"name"`
	Disabled bool                 `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Alias    *SingleLineStringVar `xml:"alias" json:"alias" yaml:"alias"`
	Start    *SingleLineStringVar `xml:"start,omitempty" json:"start,omitempty" yaml:"start,omitempty"`
	Stop     *SingleLineStringVar `xml:"stop,omitempty" json:"stop,omitempty" yaml:"stop,omitempty"`
	LogFile  *SingleLineStringVar `xml:"logFile,omitempty" json:"logfile,omitempty" yaml:"logfile,omitempty"`
}
