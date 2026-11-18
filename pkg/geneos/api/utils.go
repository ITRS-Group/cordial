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
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Columns is a common type for the map of rows for output.
type Columns map[string]columndetails

// columndetails has to be it's own type so that it can be used as values in maps
type columndetails struct {
	tags   string // copy of tags for now
	name   string // display name of column. name="OMIT" (or "-") means not rendered
	number int    // index of the column in rendered rows (OMIT columns are not rendered), -1 if not rendered
	format string // alterative Printf format, default is %v
	// TODO: Revisit unused field convfunc; restore it only when needed.
	// 	convfunc func(any) string // this may happen - not yet used
	sort sortType // if this is the sorting column then what type from above
}

const (
	// sort=[+|-][num] = sort by this column optionally asc/desc and optionally numeric, one or the other
	sorting = "sort"
	// format is a fmt.Printf format string for the data and defaults to %v
	format = "format"
)

// omitted returns true if a column name marks it as not rendered
func omitted(name string) bool {
	return name == "OMIT" || name == "-"
}

type sortType int

const (
	sortNone sortType = iota
	sortAsc
	sortDesc
	sortAscNum
	sortDescNum
)

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
func ColumnInfo(rowdata any) (cols Columns, columnnames []string, sorting string, err error) {
	if rowdata == nil {
		return
	}
	rv := reflect.Indirect(reflect.ValueOf(rowdata))
	if rv.Kind() != reflect.Struct {
		err = fmt.Errorf("rowdata is not a struct")
		return
	}

	rt := rv.Type()
	cols = make(Columns, rt.NumField())
	sorting = rt.Field(0).Name

	for i := range rt.NumField() {
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
		} else {
			column.name = fieldname
			column.format = "%v"
		}
		// A column marked "OMIT" is still useable but is not included
		// in the column names or the rendered rows, so its index
		// is that of the rendered cells and not of the struct field
		column.number = -1
		if !omitted(column.name) {
			column.number = len(columnnames)
			columnnames = append(columnnames, column.name)
		}
		cols[fieldname] = column
	}

	return
}

func (c Columns) sortRows(rows [][]string, sortcol string) [][]string {
	sorttype, sortby := c[sortcol].sort, c[sortcol].number
	if sortby < 0 {
		// the sort column is not rendered, leave the rows as they are
		return rows
	}

	// cells shorter than expected sort as empty rather than panic
	cell := func(row []string) string {
		if sortby < len(row) {
			return row[sortby]
		}
		return ""
	}

	slices.SortFunc(rows, func(a, b []string) int {
		as, bs := cell(a), cell(b)
		switch sorttype {
		case sortDesc:
			return cmp.Compare(bs, as)
		case sortAscNum:
			// err is ignored, zero is a valid answer if the
			// contents are not a float
			an, _ := strconv.ParseFloat(as, 64)
			bn, _ := strconv.ParseFloat(bs, 64)
			if an == bn {
				return strings.Compare(as, bs)
			}
			return cmp.Compare(an, bn)
		case sortDescNum:
			// err is ignored, zero is a valid answer if the
			// contents are not a float
			an, _ := strconv.ParseFloat(as, 64)
			bn, _ := strconv.ParseFloat(bs, 64)
			if an == bn {
				return strings.Compare(bs, as)
			}
			return cmp.Compare(bn, an)
		// case sortNone, sortAsc: - the default
		default:
			return strings.Compare(as, bs)
		}
	})
	return rows
}

// pivot the struct members to a slice of their values ready to be
// processed to a slice of strings
func rowFromStruct(c Columns, rv reflect.Value) (row []any, err error) {
	rv = reflect.Indirect(rv)
	if rv.Kind() != reflect.Struct {
		err = fmt.Errorf("row data not a struct")
		return
	}

	rt := rv.Type()

	for i := range rt.NumField() {
		row = append(row, rv.Field(i).Interface())
	}

	return
}

func parseTags(fieldname string, tag string) (cols columndetails, err error) {
	// non "zero" default
	cols.tags = tag
	cols.name = fieldname
	cols.format = "%v"

	var nameSet bool
	for t := range strings.SplitSeq(tag, ",") {
		prefix, value, found := strings.Cut(t, "=")
		if !found {
			if t == "" {
				continue
			}
			if nameSet {
				// err, already defined
				err = fmt.Errorf("column name %q redefined more than once", cols.name)
				return
			}
			cols.name = t
			nameSet = true
			continue
		}

		switch prefix {
		case sorting:
			cols.sort = sortAsc
			if strings.HasPrefix(value, "-") {
				cols.sort = sortDesc
			}
			if strings.HasSuffix(value, "num") {
				if cols.sort == sortAsc {
					cols.sort = sortAscNum
				} else {
					cols.sort = sortDescNum
				}
			}

		case format:
			// no validation
			cols.format = value
		}
	}
	return
}
