package cpu

import (
	"fmt"
	"runtime"

	"github.com/itrs-group/cordial/pkg/geneos/plugins"
	"github.com/itrs-group/cordial/pkg/geneos/samplers"
)

func init() {
	// logger.EnableDebugLog()
}

// CPUSampler TODO: Describe its purpose and intended use.
type CPUSampler struct {
	samplers.Samplers
	cpustats cpustat
}

// New TODO: Document its behavior, inputs, and effects.
func New(s *plugins.Connection, name string, group string) (*CPUSampler, error) {
	log.Debug("called")
	c := new(CPUSampler)
	c.Plugins = c
	return c, c.New(s, name, group)
}

// InitSampler TODO: Document its behavior, inputs, and effects.
func (p *CPUSampler) InitSampler() (err error) {
	log.Debug("called")
	p.Headline("OS", runtime.GOOS)
	p.Headline("SampleInterval", fmt.Sprintf("%v", p.Interval()))

	// call internal OS column init
	columns, columnnames, sortcol, err := p.initColumns()
	p.SetColumns(columns)
	p.SetColumnNames(columnnames)
	p.SetSortColumn(sortcol)
	return
}
