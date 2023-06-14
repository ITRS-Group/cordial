package geneos

type XPingPlugin struct {
	Display *FKMDisplay `xml:"fkm>display,omitempty" json:",omitempty" yaml:",omitempty"`
	Files   FKMFiles    `xml:"files,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"files"`
}

func (f *XPingPlugin) String() string {
	return "x-ping"
}
