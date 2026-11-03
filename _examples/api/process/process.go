package process

import (
	"fmt"
	"runtime"

	"github.com/itrs-group/cordial/pkg/geneos/plugins"
	"github.com/itrs-group/cordial/pkg/geneos/samplers"
)

// ProcessSampler TODO: Describe its purpose and intended use.
type ProcessSampler struct {
	samplers.Samplers
}

// New TODO: Document its behavior, inputs, and effects.
func New(s *plugins.Connection, name string, group string) (*ProcessSampler, error) {
	c := new(ProcessSampler)
	c.Plugins = c
	return c, c.New(s, name, group)
}

// InitSampler TODO: Document its behavior, inputs, and effects.
func (p *ProcessSampler) InitSampler() (err error) {
	p.Headline("OS", runtime.GOOS)
	p.Headline("SampleInterval", fmt.Sprintf("%v", p.Interval()))

	columns, columnnames, sortcol, err := p.initColumns()
	if err == nil {
		p.SetColumns(columns)
		p.SetColumnNames(columnnames)
		p.SetSortColumn(sortcol)
	}
	return
}
