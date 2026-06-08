package cpu

import (
	"fmt"
	"runtime"

	zlog "github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/geneos/plugins"
	"github.com/itrs-group/cordial/pkg/geneos/samplers"
)

func init() {
	// logger.EnableDebugLog()
}

type CPUSampler struct {
	samplers.Samplers
	cpustats cpustat
}

func New(s *plugins.Connection, name string, group string) (*CPUSampler, error) {
	zlog.Debug().Msg("called")
	c := new(CPUSampler)
	c.Plugins = c
	return c, c.New(s, name, group)
}

func (p *CPUSampler) InitSampler() (err error) {
	zlog.Debug().Msg("called")
	p.Headline("OS", runtime.GOOS)
	p.Headline("SampleInterval", fmt.Sprintf("%v", p.Interval()))

	// call internal OS column init
	columns, columnnames, sortcol, err := p.initColumns()
	p.SetColumns(columns)
	p.SetColumnNames(columnnames)
	p.SetSortColumn(sortcol)
	return
}
