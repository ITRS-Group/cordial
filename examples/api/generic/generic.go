package generic

import (
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/plugins"
	"github.com/itrs-group/cordial/pkg/samplers"
)

func init() {
	logger.EnableDebugLog()
}

var (
	log      = logger.Log
	logDebug = logger.Debug
	logError = logger.Error
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

func New(s plugins.Connection, name string, group string) (*GenericSampler, error) {
	c := new(GenericSampler)
	c.Plugins = c
	return c, c.New(s, name, group)
}

func (g *GenericSampler) InitSampler() error {
	logDebug.Println("called")
	example, err := g.Parameter("EXAMPLE")
	if err != nil {
		logError.Println(err)
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
	logDebug.Print("called")
	var rowdata = []GenericData{
		{"row4", "data1", "data2"},
		{"row2", "data1", "data2"},
		{"row3", "data1", "data2"},
		{"row1", "data1", "data2"},
	}
	return p.UpdateTableFromSlice(rowdata)
}
