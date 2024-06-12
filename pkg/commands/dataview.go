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

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
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
// [commands.Snapshot]. Name and XPath are populated from the request to
// Snapshot while the three "Order" slices are constructed from the order
// of the received JSON data.
type Dataview struct {
	Name             string       `json:"name"`
	XPath            *xpath.XPath `json:"xpath"`
	SampleTime       time.Time    `json:"sample-time"`
	Snoozed          bool         `json:"snoozed"`
	SnoozedAncestors bool         `json:"snoozed-ancestors"`

	// Headlines is a map of headline names to data items
	Headlines map[string]DataItem `json:"headlines,omitempty"`

	// Table is a map of row names to column names to data items, the
	// first column (row name) not included in the map. A specific
	// DataItem is Table["row"]["column"].
	Table map[string]map[string]DataItem `json:"table,omitempty"`

	// HeadlineOrder, ColumnOrder and RowOrder are slice of the
	// respective names based on the order in the Gateway REST response
	//
	// While JSON does not support fixed orders for objects, the Geneos
	// Gateway responds with the "natural" (i.e. internal) order for
	// these object maps
	HeadlineOrder []string `json:"-"`
	ColumnOrder   []string `json:"-"`
	RowOrder      []string `json:"-"`
}

// dataviewRaw contains json.RawMessage fields for further processing
// for ordered objects
type dataviewRaw struct {
	Name             string          `json:"name"`
	SampleTime       time.Time       `json:"sample-time,omitempty"`
	Snoozed          bool            `json:"snoozed,omitempty"`
	SnoozedAncestors bool            `json:"snoozed-ancestors,omitempty"`
	Headlines        json.RawMessage `json:"headlines,omitempty"`
	Table            json.RawMessage `json:"table,omitempty"`
}

// UnmarshalJSON preserves the order of the Headline, Rows and Columns
// from a snapshot in dv Order fields. If the input is empty an
// os.ErrInvalid is returned as an empty Dataview object should not be
// used.
func (dv *Dataview) UnmarshalJSON(d []byte) (err error) {
	if len(d) == 0 {
		return os.ErrInvalid
	}

	dvr := dataviewRaw{}
	if err = json.Unmarshal(d, &dvr); err != nil {
		return
	}

	dv.SampleTime = dvr.SampleTime
	dv.Snoozed = dvr.Snoozed
	dv.SnoozedAncestors = dvr.SnoozedAncestors
	dv.ColumnOrder = []string{"rowname"}

	// decode the headlines object
	dv.Headlines = map[string]DataItem{}
	hdec := json.NewDecoder(bytes.NewReader(dvr.Headlines))
	for hdec.More() {
		t, err := hdec.Token()
		if err != nil {
			return err
		}
		switch v := t.(type) {
		case json.Delim:
			// skip opening and closing
			continue
		case string:
			dv.HeadlineOrder = append(dv.HeadlineOrder, v)

			var di DataItem
			if err = hdec.Decode(&di); err != nil {
				return err
			}
			dv.Headlines[v] = di
		default:
			return &json.UnmarshalTypeError{
				Value:  "",
				Type:   reflect.TypeOf(dv),
				Offset: hdec.InputOffset(),
				Struct: "Dataview",
				Field:  "Headlines",
			}
		}
	}

	// decode table, grab column order from first row, just decode the reset directly
	dv.Table = map[string]map[string]DataItem{}
	tdec := json.NewDecoder(bytes.NewReader(dvr.Table))
	first := true

NEXTROW:
	for {
		t, err := tdec.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch v := t.(type) {
		case json.Delim:
			if v == '}' {
				break NEXTROW
			}
			continue
		case string:
			dv.RowOrder = append(dv.RowOrder, v)

			if first {
				first = false
				dv.Table[v] = map[string]DataItem{}
			NEXTCELL:
				for {
					t, err := tdec.Token()
					if err != nil {
						if err == io.EOF {
							return nil
						}
						return err
					}
					switch c := t.(type) {
					case json.Delim:
						if c == '}' {
							break NEXTCELL
						}
						continue
					case string:
						dv.ColumnOrder = append(dv.ColumnOrder, c)
						var di DataItem
						if err = tdec.Decode(&di); err != nil {
							return err
						}
						dv.Table[v][c] = di
					default:
						return &json.UnmarshalTypeError{
							Value:  "",
							Type:   reflect.TypeOf(dv),
							Offset: tdec.InputOffset(),
							Struct: "Dataview",
							Field:  "Table",
						}
					}
				}
				continue
			}

			// just decode further rows, we already have ColumnOrder
			// after the first row
			var di map[string]DataItem
			if err = tdec.Decode(&di); err != nil {
				return err
			}
			dv.Table[v] = di
		default:
			return &json.UnmarshalTypeError{
				Value:  "",
				Type:   reflect.TypeOf(dv),
				Offset: tdec.InputOffset(),
				Struct: "Dataview",
				Field:  "Table",
			}
		}
	}
	return nil
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

	// no error but no dataview? return a dataview with no content
	if cr.Dataview == nil {
		dataview = &Dataview{
			Headlines: map[string]DataItem{},
			Table:     map[string]map[string]DataItem{},
		}
		return
	}

	dataview = cr.Dataview
	dataview.Name = target.Dataview.Name
	dataview.XPath = target
	return
}
