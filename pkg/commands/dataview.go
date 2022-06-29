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

/*
Dataview Snapshot Support

Only valid in GA5.14 and above

In GA5.14.x the first column name is not exported and set to "rowname"
*/
package commands

import (
	"sort"
	"time"

	"github.com/itrs-group/cordial/pkg/xpath"
)

type DataItem struct {
	Value    string `json:"value,omitempty"`
	Severity string `json:"severity,omitempty"`
	Snoozed  bool   `json:"snoozed,omitempty"`
	Assigned bool   `json:"assigned,omitempty"`
}

type Dataview struct {
	SampleTime       time.Time                      `json:"sample-time,omitempty"`
	Snoozed          bool                           `json:"snoozed,omitempty"`
	SnoozedAncestors bool                           `json:"snoozed-ancestors,omitempty"`
	Headlines        map[string]DataItem            `json:"headlines,omitempty"`
	Table            map[string]map[string]DataItem `json:"table,omitempty"`
	Columns          []string                       `json:"-"`
}

type Scope struct {
	Value          bool `json:"value,omitempty"`
	Severity       bool `json:"severity,omitempty"`
	Snooze         bool `json:"snooze,omitempty"`
	UserAssignment bool `json:"user-assignment,omitempty"`
}

type Fetch struct {
	Target string `json:"target"`
	Scope  Scope  `json:"scope,omitempty"`
}

// Return the snapshot of the Dataview target with an option Scope. If
// no Scope is given then only Values are requested. If more than one
// Scope is given then only the first is used.
func (c *Connection) Snapshot(target *xpath.XPath, scope ...Scope) (dataview *Dataview, err error) {
	// override endpoint for snapshots
	const endpoint = "/rest/snapshot/dataview"
	s := Scope{Value: true}
	if len(scope) > 0 {
		s = scope[0]
	}
	cr, err := c.Do(endpoint, Fetch{
		Target: target.String(),
		Scope:  s,
	})
	if err != nil {
		return
	}

	if cr.Dataview != nil {
		dataview = cr.Dataview
		var row map[string]DataItem
		for k := range dataview.Table {
			row = dataview.Table[k]
			break
		}

		for k := range row {
			dataview.Columns = append(dataview.Columns, k)
		}
		sort.Strings(dataview.Columns)
		// XXX until the first column is supplied, prepend a constant
		dataview.Columns = append([]string{"rowname"}, dataview.Columns...)
	}

	return
}
