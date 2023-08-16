package api

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

type Dataview struct {
	APIClient

	Entity  string
	Sampler string
	Name    string

	columns     Columns
	columnnames []string
	sortcolumn  string
}

// NewDataview returns a Dataview view. If the connection to the API
// fails then a nil pointer is returned. If the dataview does not exist
// in the Netprobe it is also created.
func NewDataview(c Client, entity, sampler, typeName, viewName, groupHeading string) (view *Dataview, err error) {
	if typeName != "" {
		sampler += "(" + typeName + ")"
	}
	if groupHeading != "" {
		viewName = groupHeading + "-" + viewName
	}

	view = &Dataview{
		Entity:  entity,
		Sampler: sampler,
		Name:    viewName,
	}
	exists, err := view.Exists()
	if err != nil && !errors.Is(err, errors.ErrUnsupported) {
		return nil, err
	}
	if err != nil || !exists {
		if err = view.CreateDataview(entity, sampler, viewName); err != nil {
			return nil, err
		}
	}
	return
}

func (view *Dataview) Exists() (exists bool, err error) {
	if view == nil {
		return
	}
	return view.DataviewExists(view.Entity, view.Sampler, view.Name)

}

func (view *Dataview) Update() (err error) {
	return
}

/*
UpdateTableFromMap - Given a map of structs representing rows of data,
render a simple table update by converting all data and sorting the rows
by the sort column in the initialised ColumnNames member of the Sampler

Sorting the data is only to define the "natural sort order" of the data
as it appears in a Geneos Dataview without further client-side sorting.
*/
func (view *Dataview) UpdateTableFromMap(data interface{}) error {
	table, _ := view.RowsFromMap(data)
	return view.UpdateDataview(view.Entity, view.Sampler, view.Name, append([][]string{view.columnnames}, table...))
}

/*
RowFromMap is a helper function that takes a tagged (flat) struct as input
and formats a row (slice of strings) using tags. TBD, but selecting the
rowname, the sorting, the format and type conversion, scaling and labels (percent,
MB etc.)

The data passed should NOT include column heading slice as it will be
regenerated from the Columns data
*/
func (view *Dataview) RowsFromMap(rowdata interface{}) (rows [][]string, err error) {
	c := view.columns
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
			if c[fieldname].name == "-" {
				continue
			}
			cells = append(cells, fmt.Sprintf(format, rawcells[i]))
		}
		rows = append(rows, cells)
	}

	rows = c.sortRows(rows, view.sortcolumn)

	return
}

// UpdateTableFromSlice - Given an ordered slice of structs of data the
// method renders a simple table of data as defined in the Columns part
// of Samplers
func (view *Dataview) UpdateTableFromSlice(rowdata interface{}) error {
	table, _ := view.RowsFromSlice(rowdata)
	return view.UpdateDataview(view.Entity, view.Sampler, view.Name, append([][]string{view.columnnames}, table...))
}

// RowsFromSlice - results are not resorted, they are assumed to be in the order
// required
func (view *Dataview) RowsFromSlice(rowdata interface{}) (rows [][]string, err error) {
	c := view.columns

	rd := reflect.Indirect(reflect.ValueOf(rowdata))
	if rd.Kind() != reflect.Slice {
		err = fmt.Errorf("non slice passed")
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
			if c[fieldname].name == "-" {
				continue
			}
			cells = append(cells, fmt.Sprintf(format, rawcells[i]))
		}
		rows = append(rows, cells)
	}

	return
}

// UpdateTableFromMapDelta calculates the difference between the
// previous values and current, scaled by the interval
func (view *Dataview) UpdateTableFromMapDelta(newdata, olddata interface{}, interval time.Duration) error {
	table, _ := view.RowsFromMapDelta(newdata, olddata, interval)
	return view.UpdateDataview(view.Entity, view.Sampler, view.Name, append([][]string{view.columnnames}, table...))
}

// RowsFromMapDelta takes two sets of data and calculates the difference
// between them. Only numeric data is changed, any non-numeric fields
// are left unchanged and taken from newrowdata only. If an interval is
// supplied (non-zero) then that is used as a scaling value otherwise
// the straight numeric difference is calculated
//
// This is for data like sets of counters that are absolute values over
// time
func (view Dataview) RowsFromMapDelta(newrowdata, oldrowdata interface{},
	interval time.Duration) (rows [][]string, err error) {

	c := view.columns

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
			if c[fieldname].name == "-" {
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

	rows = c.sortRows(rows, view.sortcolumn)

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
