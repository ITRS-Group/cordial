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

package commands

import (
	"fmt"
	"time"

	"github.com/itrs-group/cordial/pkg/xpath"
)

// DataItem is a Geneos data item and normally represents a headline or
// table cell.
type DataItem struct {
	Value    string `json:"value,omitempty"`
	Severity string `json:"severity,omitempty"`
	Snoozed  bool   `json:"snoozed,omitempty"`
	Assigned bool   `json:"assigned,omitempty"`
}

// Dataview represents the contents of a Geneos dataview as returned by
// [Snapshot].
type Dataview struct {
	Name             string       `json:"name"`
	XPath            *xpath.XPath `json:"xpath"`
	SampleTime       time.Time    `json:"sample-time,omitempty"`
	Snoozed          bool         `json:"snoozed,omitempty"`
	SnoozedAncestors bool         `json:"snoozed-ancestors,omitempty"`

	// Headlines is a map of headline names to data items
	Headlines map[string]DataItem `json:"headlines,omitempty"`

	// Table is a map of row names to column names to data items, the
	// first column (row name) not included in the map
	Table map[string]map[string]DataItem `json:"table,omitempty"`

	// Columns is an ordered slice of column names obtained from the
	// JSON data returned from the REST endpoint to allow the Table
	// field to be iterated over in the same order as the Geneos
	// dataview table
	Columns []string `json:"-"`

	// The Rows field is an unordered slice of rownames. The caller can
	// order this in anyway desired for use as a range loop.
	Rows []string `json:"-"`
}

// Snapshot fetches the contents of the dataview identified by the
// target XPath. The target must match exactly one dataview. Only
// headline and cell values are requested unless an optional scope is
// passed, which can request severity, snooze and user assignment
// information. If the underlying REST call fails then the error is
// returned along with any stderr output.
//
// Snapshot support is only available in Geneos GA5.14 and above
// Gateways and requires the REST command API to be enabled.
//
// In existing releases the first column name is not exported and is set
// to rowname, which defaults to the literal "rowname" if passed an
// empty rowname.
func (c *Connection) Snapshot(target *xpath.XPath, rowname string, scope ...Scope) (dataview *Dataview, err error) {
	if rowname == "" {
		rowname = "rowname"
	}

	// override endpoint for snapshots
	const endpoint = "/rest/snapshot/dataview"
	s := Scope{Value: true}
	if len(scope) > 0 {
		s = scope[0]
	}
	cr, err := c.Do(endpoint, &Command{
		Target: target,
		Scope:  s,
	})
	if err != nil {
		if cr.Stderr != "" {
			err = fmt.Errorf("%w (%s)", err, cr.Stderr)
		}
		return
	}

	// no error but no dataview?
	if cr.Dataview == nil {
		dataview = &Dataview{
			Headlines: map[string]DataItem{},
			Table:     map[string]map[string]DataItem{},
			// Columns:   []string{},
			// Rows:      []string{},
		}
		return
	}

	dataview = cr.Dataview
	dataview.Columns = []string{rowname}

	// grab any row, it's the keys we actually want
	for k := range dataview.Table {
		rowmap := dataview.Table[k]
		// get the column names from the first row returned
		for k := range rowmap {
			dataview.Columns = append(dataview.Columns, k)
		}
		// exit after one iteration
		break
	}

	// XXX until the first column is supplied, prepend a constant
	// dataview.Columns = append([]string{rowname}, dataview.Columns...)

	for rowname := range dataview.Table {
		dataview.Rows = append(dataview.Rows, rowname)
	}

	dataview.Name = target.Dataview.Name
	dataview.XPath = target

	return
}
