package process

import (
	"fmt"
	"runtime"

	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/plugins"
	"github.com/itrs-group/cordial/pkg/samplers"
)

func init() {
	logger.EnableDebugLog()
}

var (
	log      = logger.Logger
	logDebug = logger.DebugLogger
	logError = logger.ErrorLogger
)

type ProcessSampler struct {
	samplers.Samplers
}

func New(s plugins.Connection, name string, group string) (*ProcessSampler, error) {
	c := new(ProcessSampler)
	c.Plugins = c
	return c, c.New(s, name, group)
}

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
