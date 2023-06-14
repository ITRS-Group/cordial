package geneos

type StateTrackerPlugin struct {
	Display *FKMDisplay `xml:"fkm>display,omitempty" json:",omitempty" yaml:",omitempty"`
	Files   FKMFiles    `xml:"files,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"files"`
}

func (f *StateTrackerPlugin) String() string {
	return "stateTracker"
}
