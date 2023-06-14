package geneos

type ControlMPlugin struct {
	Rest string `xml:",any"`
}

func (f *ControlMPlugin) String() string {
	return "control-m"
}
