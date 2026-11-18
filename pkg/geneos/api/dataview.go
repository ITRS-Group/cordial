/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

// ErrNoColumns is returned when rows are built or a table is updated
// for a Dataview before its columns have been set with
// [Dataview.SetColumns].
var ErrNoColumns = errors.New("dataview columns not set")

// Dataview TODO: Describe its purpose and intended use.
type Dataview struct {
	APIClient

	Entity  string
	Sampler string
	Name    string

	columns     Columns
	columnnames []string
	sortcolumn  string
}

// SetColumns derives the column names, the display formats and the sort
// column from the `column` struct tags of rowdata, which is a struct or
// pointer to a struct and only its type is used. It must be called
// before any of the methods that update a table from structs or build
// rows from them.
func (view *Dataview) SetColumns(rowdata any) error {
	columns, columnnames, sortcolumn, err := ColumnInfo(rowdata)
	if err != nil {
		return err
	}
	if columns == nil {
		return fmt.Errorf("%w: no struct supplied", ErrNoColumns)
	}
	view.columns, view.columnnames, view.sortcolumn = columns, columnnames, sortcolumn
	return nil
}

// NewDataview returns a Dataview view. If the connection to the API
// fails then a nil pointer is returned. If the dataview does not exist
// in the Netprobe it is also created. When a groupHeading is given the
// view is created using separate group and view names, if the client
// supports it, so that hyphens in either name are not ambiguous.
func NewDataview(c APIClient, entity, sampler, typeName, groupHeading, viewName string) (view *Dataview, err error) {
	if typeName != "" {
		sampler += "(" + typeName + ")"
	}
	var group, groupedView string
	if groupHeading != "" {
		group, groupedView = groupHeading, viewName
		viewName = groupHeading + "-" + viewName
	}

	view = &Dataview{
		APIClient: c,
		Entity:    entity,
		Sampler:   sampler,
		Name:      viewName,
	}
	exists, err := view.Exists()
	if err != nil && !errors.Is(err, errors.ErrUnsupported) {
		return nil, err
	}
	if err != nil || !exists {
		if creator, ok := c.(interface {
			CreateView(entity, sampler, viewName, groupHeading string) error
		}); ok && group != "" {
			err = creator.CreateView(entity, sampler, groupedView, group)
		} else {
			err = view.CreateDataview(entity, sampler, viewName)
		}
		if err != nil {
			return nil, err
		}
	}
	return
}

// Exists TODO: Document its behavior, inputs, and effects.
func (view *Dataview) Exists() (exists bool, err error) {
	if view == nil {
		return
	}
	return view.DataviewExists(view.Entity, view.Sampler, view.Name)

}

// Update TODO: Document its behavior, inputs, and effects.
func (view *Dataview) Update() (err error) {
	return
}

// UpdateTableFromMap - Given a map of structs representing rows of data,
// render a simple table update by converting all data and sorting the rows
// by the sort column in the initialised ColumnNames member of the Sampler

// Sorting the data is only to define the "natural sort order" of the data
// as it appears in a Geneos Dataview without further client-side sorting.
func (view *Dataview) UpdateTableFromMap(data any) error {
	table, err := view.RowsFromMap(data)
	if err != nil {
		return err
	}
	return view.UpdateDataview(view.Entity, view.Sampler, view.Name, append([][]string{view.columnnames}, table...))
}

// RowFromMap is a helper function that takes a tagged (flat) struct as input
// and formats a row (slice of strings) using tags. TBD, but selecting the
// rowname, the sorting, the format and type conversion, scaling and labels (percent,
// MB etc.)

// The data passed should NOT include column heading slice as it will be
// regenerated from the Columns data
func (view *Dataview) RowsFromMap(rowdata any) (rows [][]string, err error) {
	if rowdata == nil {
		return
	}
	c := view.columns
	if c == nil {
		err = ErrNoColumns
		return
	}
	r := reflect.Indirect(reflect.ValueOf(rowdata))
	if r.Kind() != reflect.Map {
		err = fmt.Errorf("non Map passed")
		return
	}

	for _, k := range r.MapKeys() {
		var cells []string
		rawcells, err := rowFromStruct(c, r.MapIndex(k))
		if err != nil {
			return nil, err
		}
		t := reflect.Indirect(r.MapIndex(k)).Type()
		for i := range rawcells {
			fieldname := t.Field(i).Name
			format := c[fieldname].format
			if omitted(c[fieldname].name) {
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
func (view *Dataview) UpdateTableFromSlice(rowdata any) error {
	table, err := view.RowsFromSlice(rowdata)
	if err != nil {
		return err
	}
	return view.UpdateDataview(view.Entity, view.Sampler, view.Name, append([][]string{view.columnnames}, table...))
}

// RowsFromSlice - results are not resorted, they are assumed to be in the order
// required
func (view *Dataview) RowsFromSlice(rowdata any) (rows [][]string, err error) {
	if rowdata == nil {
		return
	}

	c := view.columns
	if c == nil {
		err = ErrNoColumns
		return
	}

	rd := reflect.Indirect(reflect.ValueOf(rowdata))
	if rd.Kind() != reflect.Slice {
		err = fmt.Errorf("non slice passed")
		return
	}

	for i := range rd.Len() {
		v := reflect.Indirect(rd.Index(i))
		t := v.Type()

		rawcells, err := rowFromStruct(c, v)
		if err != nil {
			return nil, err
		}
		var cells []string
		for i := range rawcells {
			fieldname := t.Field(i).Name
			format := c[fieldname].format
			if omitted(c[fieldname].name) {
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
func (view *Dataview) UpdateTableFromMapDelta(newdata, olddata any, interval time.Duration) error {
	table, err := view.RowsFromMapDelta(newdata, olddata, interval)
	if err != nil {
		return err
	}
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
func (view Dataview) RowsFromMapDelta(newrowdata, oldrowdata any,
	interval time.Duration) (rows [][]string, err error) {
	if newrowdata == nil || oldrowdata == nil {
		return
	}

	c := view.columns
	if c == nil {
		err = ErrNoColumns
		return
	}

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
		if !rOld.MapIndex(k).IsValid() {
			err = fmt.Errorf("key %v missing from old data", k)
			return nil, err
		}
		rawOld, err := rowFromStruct(c, rOld.MapIndex(k))
		if err != nil {
			return nil, err
		}
		rawCells, err := rowFromStruct(c, rNew.MapIndex(k))
		if err != nil {
			return nil, err
		}
		var cells []string
		t := reflect.Indirect(rNew.MapIndex(k)).Type()
		for i := range rawCells {
			fieldname := t.Field(i).Name
			format := c[fieldname].format
			if omitted(c[fieldname].name) {
				continue
			}

			// calc diff here
			oldCell, newCell := rawOld[i], rawCells[i]
			if reflect.TypeOf(oldCell) != reflect.TypeOf(newCell) {
				return nil, fmt.Errorf("non-matching types in data")
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

func toFloat(f any) (float64, error) {
	var ft = reflect.TypeFor[float64]()
	v := reflect.ValueOf(f)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(ft) {
		return 0, fmt.Errorf("cannot convert %v to float", v.Type())
	}
	fv := v.Convert(ft)
	return fv.Float(), nil
}
