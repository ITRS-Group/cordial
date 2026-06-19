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

package responses

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// GeneralResponses is a collection of command responses, the key is typically
// an instance name (from i.String()) but can be any other label that is
// suitable in the circumstances.
type GeneralResponses map[string]*General

// General is a consolidated set of responses from commands
//
// TODO: Cleanup, fields are misused and overlap
type General struct {
	Instance   geneos.Instance
	start      time.Time     // start time of opertation, set by NewResponse() and can be overridden when merging responses
	Duration   time.Duration // duration of operation, set by Do() etc. when operation completes, and is used to calculate elapsed time for reporting (alternative to Start and Finish)
	ResultText []string      // free format text output, typically for human consumption
	Completed  []string      // simple past tense verbs of completed actions, e.g. "stopped", "started" etc.
	Err        error         // error from command, if any
	Value      any           // arbitrary value (typically for JSON output)
	Values     []any         // arbitrary values (typically for JSON output), which are merged into a single slice when merging responses
	Dataview   *Dataview     // dataview for reporting, each should have same columns
}

type Table struct {
	Instance geneos.Instance
	start    time.Time
	Duration time.Duration // duration of operation, set by Do() etc. when operation completes, and is used to calculate elapsed time for reporting (alternative to Start and Finish)
	Err      error
	Dataview *Dataview
}

type Text struct {
	Instance   geneos.Instance
	start      time.Time
	Duration   time.Duration // duration of operation, set by Do() etc. when operation completes, and is used to calculate elapsed time for reporting (alternative to Start and Finish)
	Err        error
	ResultText []string
	Completed  []string
}

type Response interface {
	General | Table | Text
}

// Dataview is a simple struct to hold the data for a dataview, which
// can then be passed to the reporter as a single value for JSON output,
// or converted to rows for CSV output. The Columns field is the column
// headings, and the Table field is the rows of data. The Headlines
// field is a map of headline name to value, which can be used to add
// headlines to the report.
type Dataview struct {
	Columns   []string
	Table     [][]string
	Headlines map[string]string
}

var log = cordial.Logger

func Finished[R Response](resp *R) {
	switch any(*resp).(type) {
	case General:
		r := any(*resp).(General)
		r.Duration = time.Since(r.start)
		*resp = any(r).(R)
	case Table:
		r := any(*resp).(Table)
		r.Duration = time.Since(r.start)
		*resp = any(r).(R)
	case Text:
		r := any(*resp).(Text)
		r.Duration = time.Since(r.start)
		*resp = any(r).(R)
	default:
		log.Error("unsupported response type", slog.Any("type", resp))
		os.Exit(1)
	}
}

func New[R Response](i geneos.Instance) *R {
	var resp *R
	switch any(resp).(type) {
	case *General:
		resp = any(&General{
			Instance: i,
			start:    time.Now(),
			Dataview: &Dataview{},
		}).(*R)
	case *Table:
		resp = any(&Table{
			Instance: i,
			start:    time.Now(),
			Dataview: &Dataview{},
		}).(*R)
	case *Text:
		resp = any(&Text{
			Instance:   i,
			start:      time.Now(),
			ResultText: []string{},
			Completed:  []string{},
		}).(*R)
	default:
		log.Error("unsupported response type", slog.Any("type", resp))
		os.Exit(1)
	}
	return any(resp).(*R)
}

// MergeResponse merges r1 and r2 and returns a single response pointer.
// Instance is set to r1.Instance, and the r2.Instance value is ignored.
// Single value fields are turned into multi-value fields if both r1 and
// r2 have them set. if both Value fields are set them they are turned
// into a slice of both. The Finish time from r2 is copied to r1, the
// Start time is only copied if r1.Start is unset. The Err fields are
// joined using errors.Join()
//
// This should only be used where a sequence of actions are being
// performed and a single response per instance is expected. It should
// not be used across different instances.
func MergeResponse(r1, r2 *General) (resp *General) {
	resp = New[General](r1.Instance)
	resp.Completed = append(r1.Completed, r2.Completed...)
	resp.Dataview.Table = append(r1.Dataview.Table, r2.Dataview.Table...)
	resp.Err = errors.Join(r1.Err, r2.Err)

	resp.start = r1.start
	if resp.start.IsZero() {
		resp.start = r2.start
	}

	resp.Duration = r1.Duration + r2.Duration

	switch {
	case r1.Value == nil:
		resp.Value = r2.Value
	case r1.Value != nil && r2.Value != nil:
		resp.Value = []any{r1.Value, r2.Value}
	default:
		resp.Value = r1.Value
	}

	resp.Values = append(r1.Values, r2.Values...)

	// switch {
	// case r1.Summary != "" && r2.Summary != "":
	// 	resp.Details = append(resp.Details, r1.Summary, r2.Summary)
	// case r1.Summary != "":
	// 	resp.Summary = r1.Summary
	// case r2.Summary != "":
	// 	resp.Summary = r2.Summary
	// }

	r1.ResultText = append(r1.ResultText, r2.ResultText...)
	return
}
