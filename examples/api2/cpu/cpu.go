package cpu

import (
	"fmt"
	"runtime"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/geneos/api"
)

func init() {
	api.RegisterPlugin(cpu)
}

var cpu = api.Plugin{
	Name: "cpu",
	Start: InitSampler,
	SampleNow: SampleNow,
}

type Sampler struct {
	api.Sampler
	cpustats cpustat
}

func New(c *api.XMLRPCClient, name string, group string) (s *Sampler, error) {
	log.Debug().Msg("called")
	// s = api.NewSampler(c, )
	s = &Sampler{
		Sampler: api.NewSampler(x, ...),		
	}
	c.Plugins = c
	return c, c.New(s, name, group)
}

func (p *Sampler) InitSampler() (err error) {
	log.Debug().Msg("called")
	p.Headline("OS", runtime.GOOS)
	p.Headline("SampleInterval", fmt.Sprintf("%v", p.Interval()))

	// call internal OS column init
	columns, columnnames, sortcol, err := p.initColumns()
	p.SetColumns(columns)
	p.SetColumnNames(columnnames)
	p.SetSortColumn(sortcol)
	return
}
