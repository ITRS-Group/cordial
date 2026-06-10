package generic

import (
	"log/slog"

	"github.com/itrs-group/cordial/pkg/geneos/plugins"
	"github.com/itrs-group/cordial/pkg/geneos/samplers"
	"github.com/itrs-group/cordial/pkg/logger"
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

var log = logger.Logger

func New(s *plugins.Connection, name string, group string) (*GenericSampler, error) {
	c := new(GenericSampler)
	c.Plugins = c
	return c, c.New(s, name, group)
}

func (g *GenericSampler) InitSampler() error {
	log.Debug("called")
	example, err := g.Parameter("EXAMPLE")
	longparameter, err := g.Parameter("DIRS")
	log.Debug("long param len", slog.Int("len", len(longparameter)), slog.String("value", longparameter))
	if err != nil {
		log.Error("error retrieving parameter", slog.Any("error", err))
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
	log.Debug("called")
	var rowdata = []GenericData{
		{"row4", "data1", "data2"},
		{"row2", "data1", "data2"},
		{"row3", "data1", "data2"},
		{"row1", "data1", "data2"},
	}
	return p.UpdateTableFromSlice(rowdata)
}
