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

package api

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

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
func ColumnInfo(rowdata interface{}) (cols Columns, columnnames []string, sorting string, err error) {
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
