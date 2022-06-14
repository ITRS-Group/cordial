package cpu

import (
	"fmt"
	"runtime"

	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/plugins"
	"github.com/itrs-group/cordial/pkg/samplers"
)

func init() {
	// logger.EnableDebugLog()
}

var (
	log      = logger.Log
	logDebug = logger.Debug
	logError = logger.Error
)

type CPUSampler struct {
	samplers.Samplers
	cpustats cpustat
}

func New(s plugins.Connection, name string, group string) (*CPUSampler, error) {
	logDebug.Println("called")
	c := new(CPUSampler)
	c.Plugins = c
	return c, c.New(s, name, group)
}

func (p *CPUSampler) InitSampler() (err error) {
	logDebug.Println("called")
	p.Headline("OS", runtime.GOOS)
	p.Headline("SampleInterval", fmt.Sprintf("%v", p.Interval()))

	// call internal OS column init
	columns, columnnames, sortcol, err := p.initColumns()
	p.SetColumns(columns)
	p.SetColumnNames(columnnames)
	p.SetSortColumn(sortcol)
	return
}
