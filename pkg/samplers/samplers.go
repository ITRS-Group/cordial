/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package samplers

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/plugins"
	"github.com/itrs-group/cordial/pkg/xmlrpc"
)

func init() {
	// logger.EnableDebugLog()
}

var (
	log      = logger.Log
	logDebug = logger.Debug
	logError = logger.Error
)

type SamplerInstance interface {
	New(plugins.Connection, string, string) *SamplerInstance
	InitSampler() (err error)
	DoSample() (err error)
}

// All plugins share common settings
type Samplers struct {
	plugins.Plugins
	*xmlrpc.Dataview
	name        string
	group       string
	interval    time.Duration
	columns     Columns
	columnnames []string
	sortcolumn  string
}

// Columns is a common type for the map of rows for output.
type Columns map[string]columndetails

// columndetails has to be it's own type so that it can be used as values in maps
type columndetails struct {
	tags     string                   // copy of tags for now
	name     string                   // display name of column. name="OMIT" mean not rendered
	number   int                      // column index - convenience for now
	format   string                   // alterative Printf format, default is %v
	convfunc func(interface{}) string // this may happen - not yet used
	sort     sortType                 // if this is the sorting column then what type from above
}

const (
	// sort=[+|-][num] = sort by this column optionally asc/desc and optionally numeric, one or the other
	sorting = "sort"
	// format is a fmt.Printf format string for the data and defaults to %v
	format = "format"
)

type sortType int

const (
	sortNone sortType = iota
	sortAsc
	sortDesc
	sortAscNum
	sortDescNum
)

// these two internal functions implement the redirection required to
// call initialisation and sample routines from plugins. without
// these, using direct calls, the process will crash if one of the functions
// isn't defined and there is no way to check before calling. this also
// allows for future shared initialisation code

func (p *Samplers) initSamplerInternal() error {
	if v, ok := interface{}(p.Plugins).(interface{ InitSampler() error }); ok {
		return v.InitSampler()
	}
	log.Print("no InitSampler() found in plugin")
	return nil
}

func (p *Samplers) doSampleInterval() error {
	if v, ok := interface{}(p.Plugins).(interface{ DoSample() error }); ok {
		return v.DoSample()
	}
	log.Print("no DoSample() found in plugin")
	return nil
}

func (s *Samplers) New(p plugins.Connection, name string, group string) error {
	logDebug.Print("called")
	s.name, s.group = name, group
	return s.initDataviews(p)
}

func (p *Samplers) SetInterval(interval time.Duration) {
	p.interval = interval
}

func (p Samplers) Interval() time.Duration {
	return p.interval
}

func (p *Samplers) SetColumnNames(columnnames []string) {
	p.columnnames = columnnames
}

func (p Samplers) ColumnNames() []string {
	return p.columnnames
}

func (p *Samplers) SetColumns(columns Columns) {
	p.columns = columns
}

func (p Samplers) Columns() Columns {
	return p.columns
}

func (p *Samplers) SetSortColumn(column string) {
	p.sortcolumn = column
}

func (p Samplers) SortColumn() string {
	return p.sortcolumn
}

func (s *Samplers) initDataviews(p plugins.Connection) (err error) {
	d, err := p.NewDataview(s.name, s.group)
	if err != nil {
		return
	}
	s.Dataview = d
	return
}

func (p *Samplers) Start(wg *sync.WaitGroup) (err error) {
	if !p.IsValid() {
		err = fmt.Errorf("Start(): Dataview not defined")
		return
	}
	err = p.initSamplerInternal()
	if err != nil {
		return
	}
	wg.Add(1)
	go func() {
		tick := time.NewTicker(p.Interval())
		defer tick.Stop()
		for {
			<-tick.C
			err := p.doSampleInterval()
			if err != nil {
				break
			}
		}
		wg.Done()
		log.Printf("sampler %q exiting\n", p)

	}()
	return
}

func (s *Samplers) Close() error {
	if !s.IsValid() {
		return nil
	}
	return s.Dataview.Close()
}

// the methods below are helpers for common cases of needing to render a struct of data as
// a row etc.

/*
ColumnInfo is a helper function that takes a (flat) struct as input
and returns an ordered slice of column names ready to update a dataview.
Normally called once per sampler during initialisation.

The column names are the display names in the struct tags or the field name
otherwise. The internal method parsetags() is where the valid options are
defined in detail. More docs to follow.

The input is a type or an zero-ed struct as this method only checks the struct
tags and doesn't care about the data

*/
func (s Samplers) ColumnInfo(rowdata interface{}) (cols Columns,
	columnnames []string, sorting string, err error) {
	rv := reflect.Indirect(reflect.ValueOf(rowdata))
	if rv.Kind() != reflect.Struct {
		err = fmt.Errorf("rowdata is not a struct")
		return
	}

	rt := rv.Type()
	cols = make(Columns, rt.NumField())
	sorting = rt.Field(0).Name

	for i := 0; i < rt.NumField(); i++ {
		column := columndetails{}
		fieldname := rt.Field(i).Name
		if tags, ok := rt.Field(i).Tag.Lookup("column"); ok {
			column, err = parseTags(fieldname, tags)
			if err != nil {
				return
			}
			// check for already set values and error
			if column.sort != sortNone {
				sorting = fieldname
			}
			column.number = i
		} else {
			column.name = fieldname
			column.number = i
			column.format = "%v"
		}
		// A column marked "OMIT" is still useable but is not included
		// in the column names
		if column.name != "OMIT" {
			columnnames = append(columnnames, column.name)
		}
		cols[fieldname] = column
	}

	return
}

/*
UpdateTableFromMap - Given a map of structs representing rows of data,
render a simple table update by converting all data and sorting the rows
by the sort column in the initialised ColumnNames member of the Sampler

Sorting the data is only to define the "natural sort order" of the data
as it appears in a Geneos Dataview without further client-side sorting.
*/
func (s *Samplers) UpdateTableFromMap(data interface{}) error {
	table, _ := s.RowsFromMap(data)
	return s.UpdateTable(s.ColumnNames(), table...)
}

/*
RowFromMap is a helper function that takes a tagged (flat) struct as input
and formats a row (slice of strings) using tags. TBD, but selecting the
rowname, the sorting, the format and type conversion, scaling and labels (percent,
MB etc.)

The data passed should NOT include column heading slice as it will be
regenerated from the Columns data
*/
func (s Samplers) RowsFromMap(rowdata interface{}) (rows [][]string, err error) {
	c := s.Columns()
	r := reflect.Indirect(reflect.ValueOf(rowdata))
	if r.Kind() != reflect.Map {
		err = fmt.Errorf("non Map passed")
		return
	}

	for _, k := range r.MapKeys() {
		var cells []string
		rawcells, _ := rowFromStruct(c, r.MapIndex(k))
		t := reflect.Indirect(r.MapIndex(k)).Type()
		for i := range rawcells {
			fieldname := t.Field(i).Name
			format := c[fieldname].format
			if c[fieldname].name == "OMIT" {
				continue
			}
			cells = append(cells, fmt.Sprintf(format, rawcells[i]))
		}
		rows = append(rows, cells)
	}

	rows = c.sortRows(rows, s.SortColumn())

	return
}

/*
UpdateTableFromSlice - Given an ordered slice of structs of data the
method renders a simple table of data as defined in the Columns
part of Samplers

*/
func (s Samplers) UpdateTableFromSlice(rowdata interface{}) error {
	table, _ := s.RowsFromSlice(rowdata)
	return s.UpdateTable(s.ColumnNames(), table...)
}

// RowsFromSlice - results are not resorted, they are assumed to be in the order
// required
func (s Samplers) RowsFromSlice(rowdata interface{}) (rows [][]string, err error) {
	c := s.Columns()

	rd := reflect.Indirect(reflect.ValueOf(rowdata))
	if rd.Kind() != reflect.Slice {
		err = fmt.Errorf("non Slice passed")
		return
	}

	for i := 0; i < rd.Len(); i++ {
		v := rd.Index(i)
		t := v.Type()

		rawcells, _ := rowFromStruct(c, v)
		var cells []string
		for i := range rawcells {
			fieldname := t.Field(i).Name
			format := c[fieldname].format
			if c[fieldname].name == "OMIT" {
				continue
			}
			cells = append(cells, fmt.Sprintf(format, rawcells[i]))
		}
		rows = append(rows, cells)
	}

	return
}

/*
UpdateTableFromMapDelta
*/
func (s *Samplers) UpdateTableFromMapDelta(newdata, olddata interface{}, interval time.Duration) error {
	table, _ := s.RowsFromMapDelta(newdata, olddata, interval)
	return s.UpdateTable(s.ColumnNames(), table...)
}

// RowsFromMapDelta takes two sets of data and calculates the difference between them.
// Only numeric data is changed, any non-numeric fields are left
// unchanges and taken from newrowdata only. If an interval is supplied (non-zero) then that is used as
// a scaling value otherwise the straight numeric difference is calculated
//
// This is for data like sets of counters that are absolute values over time
func (s Samplers) RowsFromMapDelta(newrowdata, oldrowdata interface{},
	interval time.Duration) (rows [][]string, err error) {

	c := s.Columns()

	// if no interval is supplied - the same as an interval of zero
	// then set 1 second as the interval as the divisor below takes
	// the number of seconds as the value, hence cancelling itself out
	if interval == 0 {
		interval = 1 * time.Second
	}

	rNew := reflect.Indirect(reflect.ValueOf(newrowdata))
	if rNew.Kind() != reflect.Map {
		err = fmt.Errorf("non map passed")
		return
	}

	rOld := reflect.Indirect(reflect.ValueOf(oldrowdata))
	if rOld.Kind() != reflect.Map {
		err = fmt.Errorf("non map passed")
		return
	}

	for _, k := range rNew.MapKeys() {
		rawOld, _ := rowFromStruct(c, rOld.MapIndex(k))
		rawCells, _ := rowFromStruct(c, rNew.MapIndex(k))
		var cells []string
		t := reflect.Indirect(rNew.MapIndex(k)).Type()
		for i := range rawCells {
			fieldname := t.Field(i).Name
			format := c[fieldname].format
			if c[fieldname].name == "OMIT" {
				continue
			}

			// calc diff here
			oldCell, newCell := rawOld[i], rawCells[i]
			if reflect.TypeOf(oldCell) != reflect.TypeOf(newCell) {
				err = fmt.Errorf("non-matching types in data")
				return
			}
			// can these fields be converted to float (the concrete value)
			// this is not the same as parsing a string as float, but the
			// actual struct field being numeric
			newFloat, nerr := toFloat(newCell)
			oldFloat, oerr := toFloat(oldCell)
			if nerr == nil && oerr == nil {
				cells = append(cells, fmt.Sprintf(format, (newFloat-oldFloat)/interval.Seconds()))
			} else {
				// if we fail to convert then just render the new values directly
				cells = append(cells, fmt.Sprintf(format, newCell))
			}
		}
		rows = append(rows, cells)
	}

	rows = c.sortRows(rows, s.SortColumn())

	return
}

func toFloat(f interface{}) (float64, error) {
	var ft = reflect.TypeOf(float64(0))
	v := reflect.ValueOf(f)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(ft) {
		return 0, fmt.Errorf("cannot convert %v to float", v.Type())
	}
	fv := v.Convert(ft)
	return fv.Float(), nil
}

func (c Columns) sortRows(rows [][]string, sortcol string) [][]string {
	sorttype, sortby := c[sortcol].sort, c[sortcol].number

	sort.Slice(rows, func(a, b int) bool {
		switch sorttype {
		case sortDesc:
			return rows[a][sortby] >= rows[b][sortby]
		case sortAscNum:
			// err is ignored, zero is a valid answer if the
			// contents are not a float
			an, _ := strconv.ParseFloat(rows[a][sortby], 64)
			bn, _ := strconv.ParseFloat(rows[b][sortby], 64)
			if an == bn {
				return rows[a][sortby] < rows[b][sortby]
			} else {
				return an < bn
			}
		case sortDescNum:
			// err is ignored, zero is a valid answer if the
			// contents are not a float
			an, _ := strconv.ParseFloat(rows[a][sortby], 64)
			bn, _ := strconv.ParseFloat(rows[b][sortby], 64)
			if an == bn {
				return rows[a][sortby] >= rows[b][sortby]
			} else {
				return an >= bn
			}
		// case sortNone, sortAsc: - the default
		default:
			return rows[a][sortby] < rows[b][sortby]
		}
	})
	return rows
}

// pivot the struct members to a slice of their values ready to be
// processed to a slice of strings
func rowFromStruct(c Columns, rv reflect.Value) (row []interface{}, err error) {
	rv = reflect.Indirect(rv)
	if rv.Kind() != reflect.Struct {
		err = fmt.Errorf("row data not a struct")
		return
	}

	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		row = append(row, rv.Field(i).Interface())
	}

	return
}

func parseTags(fieldname string, tag string) (cols columndetails, err error) {
	// non "zero" default
	cols.tags = tag
	cols.name = fieldname
	cols.format = "%v"

	tags := strings.Split(tag, ",")
	for _, t := range tags {
		i := strings.IndexByte(t, '=')
		if i == -1 {
			if cols.name != fieldname {
				// err, already defined
				err = fmt.Errorf("column name %q redefined more than once", cols.name)
				return
			}
			cols.name = t
			continue
		}
		prefix := t[:i]

		switch prefix {
		case sorting:
			cols.sort = sortAsc
			if t[i+1] == '-' {
				cols.sort = sortDesc
			}
			if strings.HasSuffix(t[i+1:], "num") {
				if cols.sort == sortAsc {
					cols.sort = sortAscNum
				} else {
					cols.sort = sortDescNum
				}
			}

		case format:
			// no validation
			cols.format = t[i+1:]
		}
	}
	return
}
