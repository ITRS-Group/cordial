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
	"maps"
	"os"
	"slices"
	"time"

	"github.com/itrs-group/cordial/pkg/logger"
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
	Audited    bool          // whether the action has been audited
	Err        error         // error from command, if any
	Value      any           // arbitrary value (typically for JSON output)
	Values     []any         // arbitrary values (typically for JSON output), which are merged into a single slice when merging responses
	Dataview   *Dataview     // dataview for reporting, each should have same columns
}

// Response TODO: Describe its purpose and intended use.
type Response interface {
	General
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

var log = logger.Logger

// ErrReported is returned by [GeneralResponses.Report] and
// [GeneralResponses.Formatted] if one or more of the responses failed.
// The failures have already been reported to the user, per response, so
// the command should return this error unchanged, to cause a non-zero
// exit status, and it is not written again.
var ErrReported = errors.New("one or more operations failed")

// label returns the name to use for a response in messages. It is the
// instance, or the key of the response if there is no instance, for
// example for a name that did not match any instance.
func (r *General) label(key string) string {
	if r.Instance == nil {
		return key
	}
	return r.Instance.String()
}

// NewFailed returns a response, not associated with an instance, for
// something that failed with err, such as a name that did not match any
// instance. The name should be used as the key for the response.
func NewFailed(err error) *General {
	return &General{Err: err, start: time.Now(), Dataview: &Dataview{}}
}

// Finished TODO: Document its behavior, inputs, and effects.
func Finished[R Response](resp *R) {
	switch r := any(resp).(type) {
	case *General:
		r.Duration = time.Since(r.start)
	default:
		log.Error("unsupported response type", slog.Any("type", resp))
		os.Exit(1)
	}
}

// New TODO: Document its behavior, inputs, and effects.
func New[R Response](i geneos.Instance) *R {
	var resp *R
	switch any(resp).(type) {
	case *General:
		resp = any(&General{
			Instance: i,
			start:    time.Now(),
			Dataview: &Dataview{},
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
//
// Completed, ResultText, Values and the Dataview rows are concatenated,
// the Dataview Columns are those of r1 or, if none, r2 and the
// Headlines are merged with those of r2 taking precedence. Audited is
// set if either response has been audited. Neither r1 nor r2 are
// changed.
func MergeResponse(r1, r2 *General) (resp *General) {
	resp = New[General](r1.Instance)
	resp.Completed = slices.Concat(r1.Completed, r2.Completed)
	resp.ResultText = slices.Concat(r1.ResultText, r2.ResultText)
	resp.Err = errors.Join(r1.Err, r2.Err)
	resp.Audited = r1.Audited || r2.Audited

	d1, d2 := r1.Dataview, r2.Dataview
	if d1 == nil {
		d1 = &Dataview{}
	}
	if d2 == nil {
		d2 = &Dataview{}
	}
	resp.Dataview.Table = slices.Concat(d1.Table, d2.Table)
	resp.Dataview.Columns = d1.Columns
	if len(resp.Dataview.Columns) == 0 {
		resp.Dataview.Columns = d2.Columns
	}
	if len(d1.Headlines) > 0 || len(d2.Headlines) > 0 {
		resp.Dataview.Headlines = make(map[string]string, len(d1.Headlines)+len(d2.Headlines))
		maps.Copy(resp.Dataview.Headlines, d1.Headlines)
		maps.Copy(resp.Dataview.Headlines, d2.Headlines)
	}

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

	resp.Values = slices.Concat(r1.Values, r2.Values)

	return
}
