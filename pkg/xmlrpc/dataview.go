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

package xmlrpc

import (
	"errors"
	"fmt"
	"time"
)

// Dataview struct encapsulates the Sampler it belongs to and adds the
// name. The name is the aggregated for of [group]-name the "-" is always
// present
type Dataview struct {
	Sampler
	// dataviewName string // [group]-name
	viewName  string
	groupName string
}

var (
	err_dataview_exists = errors.New("dataview doesn't exist")
)

// String returns a formatted view name
func (d Dataview) String() string {
	return fmt.Sprintf("%s-%s", d.groupName, d.viewName)
}

// Exists checks if the dataview exists
func (d Dataview) Exists() bool {
	res, err := d.viewExists(d.entityName, d.samplerName, d.String())
	if err != nil {
		logError.Print(err)
		return false
	}
	return res
}

// Close - currently a no-op
func (d *Dataview) Close() (err error) {
	logDebug.Print("called")
	return
}

func (d *Dataview) Remove() (err error) {
	logDebug.Print("called")
	if !d.Exists() {
		return
	}
	err = d.removeView(d.entityName, d.samplerName, d.viewName, d.groupName)
	return
}

// UpdateCell sets the value of an existing dataview cell given the row and column name
// The value is formatted using %v so this can be passed any concrete value
//
// No validation is done on args
func (d Dataview) UpdateCell(rowname string, columnname string, value interface{}) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	cellname := rowname + "." + columnname
	s := fmt.Sprintf("%v", value)
	err = d.updateTableCell(d.entityName, d.samplerName, d.String(), cellname, s)
	return
}

// UpdateTable replaces the contents of the dataview table but will not work if
// the column names have changed. The underlying API requires the caller to remove the
// original dataview unless you are simply adding new columns
//
// The arguments are a mandatory slice of column names followed by any number
// of rows in the form of a variadic list of slices of strings
func (d Dataview) UpdateTable(columns []string, values ...[]string) (err error) {
	if !d.Exists() {
		err = fmt.Errorf("UpdateTable(%q): dataview doesn't exist", d)
		logError.Print(err)
		return
	}
	var table [][]string = append([][]string{columns}, values...)
	err = d.updateEntireTable(d.entityName, d.samplerName, d.String(), table)
	return
}

func (d Dataview) AddRow(name string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	err = d.addTableRow(d.entityName, d.samplerName, d.String(), name)
	return
}

func (d Dataview) UpdateRow(name string, args ...interface{}) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	var s []string
	for _, v := range args {
		s = append(s, fmt.Sprintf("%v", v))
	}
	err = d.updateTableRow(d.entityName, d.samplerName, d.String(), name, s)
	return
}

func (d Dataview) RowNames() (rownames []string, err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	rownames, err = d.getRowNames(d.entityName, d.samplerName, d.String())
	if err != nil {
		return
	}
	return
}

func (d Dataview) RowNamesOlderThan(datetime time.Time) (rownames []string, err error) {
	unixtime := datetime.Unix()
	rownames, err = d.getRowNamesOlderThan(d.entityName, d.samplerName, d.String(), unixtime)
	if err != nil {
		logError.Print(err)
		return
	}
	return
}

func (d Dataview) CountRows() (int, error) {
	if !d.Exists() {
		err := err_dataview_exists
		logError.Print(err)
		return 0, err
	}
	return d.getRowCount(d.entityName, d.samplerName, d.String())
}

func (d Dataview) RemoveRow(name string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	err = d.removeTableRow(d.entityName, d.samplerName, d.String(), name)
	return
}

func (d Dataview) AddColumn(name string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	err = d.addTableColumn(d.entityName, d.samplerName, d.String(), name)
	return
}

// You cannot remove an existing column. You have to recreate the Dataview

func (d Dataview) ColumnNames() (columnnames []string, err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	columnnames, err = d.getColumnNames(d.entityName, d.samplerName, d.String())
	if err != nil {
		return
	}
	return
}

func (d Dataview) CountColumns() (int, error) {
	if !d.Exists() {
		err := err_dataview_exists
		logError.Print(err)
		return 0, err
	}
	return d.getColumnCount(d.entityName, d.samplerName, d.String())
}

// create and optional populate headline
// this is also the entry point to update the value of a headline

func (d Dataview) Headline(name string, args ...string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	res, err := d.headlineExists(d.entityName, d.samplerName, d.String(), name)
	if err != nil {
		logError.Print(err)
		return
	}
	if !res {
		err = d.addHeadline(d.entityName, d.samplerName, d.String(), name)
	}
	if err != nil {
		logError.Print(err)
		return
	}
	if len(args) > 0 {
		s := fmt.Sprintf("%v", args[0])
		err = d.updateHeadline(d.entityName, d.samplerName, d.String(), name, s)
		if err != nil {
			logError.Print(err)
			return
		}
	}
	return
}

func (d Dataview) CountHeadlines() (int, error) {
	if !d.Exists() {
		err := err_dataview_exists
		logError.Print(err)
		return 0, err
	}
	return d.getHeadlineCount(d.entityName, d.samplerName, d.String())
}

func (d Dataview) HeadlineNames() (headlinenames []string, err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	headlinenames, err = d.getHeadlineNames(d.entityName, d.samplerName, d.String())
	if err != nil {
		return
	}
	return
}

func (d Dataview) RemoveHeadline(name string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		logError.Print(err)
		return
	}
	res, err := d.headlineExists(d.entityName, d.samplerName, d.String(), name)
	if !res {
		logError.Print(err)
		return
	}
	return d.removeHeadline(d.entityName, d.samplerName, d.String(), name)
}
