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
		return false
	}
	return res
}

// Close - currently a no-op. To remove a dataview use [Remove]
func (d *Dataview) Close() (err error) {
	return
}

// Remove a dataview
func (d *Dataview) Remove() (err error) {
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
func (d Dataview) UpdateCell(rowname string, column string, value interface{}) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	cellname := rowname + "." + column
	s := fmt.Sprintf("%v", value)
	return d.updateTableCell(d.entityName, d.samplerName, d.String(), cellname, s)
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
		return
	}
	var table [][]string = append([][]string{columns}, values...)
	return d.updateEntireTable(d.entityName, d.samplerName, d.String(), table)
}

func (d Dataview) RowExists(rowname string) bool {
	if !d.Exists() {
		return false
	}
	exists, _ := d.rowExists(d.entityName, d.samplerName, d.viewName, rowname)
	return exists
}

func (d Dataview) AddRow(rowname string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	return d.addTableRow(d.entityName, d.samplerName, d.String(), rowname)
}

func (d Dataview) UpdateRow(rowname string, args ...interface{}) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	var s []string
	for _, v := range args {
		s = append(s, fmt.Sprintf("%v", v))
	}
	return d.updateTableRow(d.entityName, d.samplerName, d.String(), rowname, s)
}

func (d Dataview) RowNames() (rownames []string, err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	return d.getRowNames(d.entityName, d.samplerName, d.String())
}

func (d Dataview) RowNamesOlderThan(datetime time.Time) (rownames []string, err error) {
	unixtime := datetime.Unix()
	return d.getRowNamesOlderThan(d.entityName, d.samplerName, d.String(), unixtime)
}

func (d Dataview) CountRows() (int, error) {
	if !d.Exists() {
		err := err_dataview_exists
		return 0, err
	}
	return d.getRowCount(d.entityName, d.samplerName, d.String())
}

func (d Dataview) RemoveRow(rowname string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	return d.removeTableRow(d.entityName, d.samplerName, d.String(), rowname)
}

func (d Dataview) ColumnExists(column string) bool {
	if !d.Exists() {
		return false
	}
	exists, _ := d.columnExists(d.entityName, d.samplerName, d.viewName, column)
	return exists
}

func (d Dataview) AddColumn(column string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	return d.addTableColumn(d.entityName, d.samplerName, d.String(), column)
}

// You cannot remove an existing column. You have to recreate the Dataview

func (d Dataview) ColumnNames() (columnnames []string, err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	return d.getColumnNames(d.entityName, d.samplerName, d.String())
}

func (d Dataview) CountColumns() (int, error) {
	if !d.Exists() {
		err := err_dataview_exists
		return 0, err
	}
	return d.getColumnCount(d.entityName, d.samplerName, d.String())
}

// create and optional populate headline
// this is also the entry point to update the value of a headline

func (d Dataview) Headline(headline string, args ...string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	res, err := d.headlineExists(d.entityName, d.samplerName, d.String(), headline)
	if err != nil {
		return
	}
	if !res {
		err = d.addHeadline(d.entityName, d.samplerName, d.String(), headline)
	}
	if err != nil {
		return
	}
	if len(args) > 0 {
		s := fmt.Sprintf("%v", args[0])
		return d.updateHeadline(d.entityName, d.samplerName, d.String(), headline, s)
	}
	return
}

func (d Dataview) HeadlineExists(headline string) bool {
	if !d.Exists() {
		return false
	}
	exists, _ := d.headlineExists(d.entityName, d.samplerName, d.viewName, headline)
	return exists
}

func (d Dataview) CountHeadlines() (int, error) {
	if !d.Exists() {
		err := err_dataview_exists
		return 0, err
	}
	return d.getHeadlineCount(d.entityName, d.samplerName, d.String())
}

func (d Dataview) HeadlineNames() (headlinenames []string, err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	return d.getHeadlineNames(d.entityName, d.samplerName, d.String())
}

func (d Dataview) RemoveHeadline(headline string) (err error) {
	if !d.Exists() {
		err = err_dataview_exists
		return
	}
	res, err := d.headlineExists(d.entityName, d.samplerName, d.String(), headline)
	if !res {
		return
	}
	return d.removeHeadline(d.entityName, d.samplerName, d.String(), headline)
}
