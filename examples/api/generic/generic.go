package generic

import (
	zlog "github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/geneos/plugins"
	"github.com/itrs-group/cordial/pkg/geneos/samplers"
)

type GenericData struct {
	RowName string
	Column1 string
	Column2 string
}

type GenericSampler struct {
	samplers.Samplers
	localdata string
}

func New(s *plugins.Connection, name string, group string) (*GenericSampler, error) {
	c := new(GenericSampler)
	c.Plugins = c
	return c, c.New(s, name, group)
}

func (g *GenericSampler) InitSampler() error {
	zlog.Debug().Msg("called")
	example, err := g.Parameter("EXAMPLE")
	longparameter, err := g.Parameter("DIRS")
	zlog.Printf("long param len: %d\n%s", len(longparameter), longparameter)
	if err != nil {
		zlog.Error().Err(err).Msg("")
		return nil
	}
	g.localdata = example

	columns, columnnames, sortcol, err := g.ColumnInfo(GenericData{})
	g.SetColumns(columns)
	g.SetColumnNames(columnnames)
	g.SetSortColumn(sortcol)
	return g.Headline("example", g.localdata)
}

func (p *GenericSampler) DoSample() error {
	zlog.Debug().Msg("called")
	var rowdata = []GenericData{
		{"row4", "data1", "data2"},
		{"row2", "data1", "data2"},
		{"row3", "data1", "data2"},
		{"row1", "data1", "data2"},
	}
	return p.UpdateTableFromSlice(rowdata)
}
