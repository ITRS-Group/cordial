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
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Responses is a collection of command responses, the key is typically
// an instance name (from i.String()) but can be any other label that is
// suitable in the circumstances.
type Responses map[string]*General

// General is a consolidated set of responses from commands
//
// TODO: Cleanup, fields are misused and overlap
type General struct {
	Instance   geneos.Instance
	Duration   time.Duration // duration of operation, set by Do() etc. when operation completes, and is used to calculate elapsed time for reporting (alternative to Start and Finish)
	ResultText []string      // free format text output, typically for human consumption
	Completed  []string      // simple past tense verbs of completed actions, e.g. "stopped", "started" etc.
	Err        error         // error from command, if any
	Value      any           // arbitrary value (typically for JSON output)
	Values     []any         // arbitrary values (typically for JSON output), which are merged into a single slice when merging responses
	Dataview   *Dataview     // dataview for reporting, each should have same columns
	start      time.Time     // start time of opertation, set by NewResponse() and can be overridden when merging responses
}

type Table struct {
	Instance geneos.Instance
	Duration time.Duration // duration of operation, set by Do() etc. when operation completes, and is used to calculate elapsed time for reporting (alternative to Start and Finish)
	Err      error
	Dataview *Dataview
	start    time.Time
}

type Text struct {
	Instance   geneos.Instance
	Duration   time.Duration // duration of operation, set by Do() etc. when operation completes, and is used to calculate elapsed time for reporting (alternative to Start and Finish)
	Err        error
	ResultText []string
	Completed  []string
	start      time.Time
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

func SetFinish[R Response](resp *R) {
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
		log.Fatal().Msgf("unsupported response type %T", resp)
	}
}

// NewResponse returns a new Response structure for instance i. The
// Start time is set to time.Now().
func NewResponse(i geneos.Instance) *General {
	return &General{
		Instance: i,
		start:    time.Now(),
		Dataview: &Dataview{},
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
		log.Fatal().Msgf("unsupported response type %T", resp)
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
	resp = NewResponse(r1.Instance)
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

// Formatted outputs the responses as a report in the specified format
// to writer w. headings are the column headings to use. prequel is any
// rows to add before the response rows.
//
// options are any reporter.ReporterOptions to control the output.
//
// If any response has a non-nil Err field then it is skipped.
//
// The response struct field used is Rows for table rows and headlines
// should be passed in as a responses.ReporterOption with
// responses.AddHeadlines()
func (responses Responses) Formatted(w io.Writer, format string, headings []string, prequel [][]string, options ...any) (err error) {
	writerOptions := make([]WriterOption, 0, len(options))
	reporterOptions := make([]any, 0, len(options))

	for _, o := range options {
		if ro, ok := o.(WriterOption); ok {
			writerOptions = append(writerOptions, ro)
		} else {
			reporterOptions = append(reporterOptions, o)
		}
	}

	opts := evalWriterOptions(writerOptions...)

	// special case "json", merge multiple Values and Value into a single slice and output
	if format == "json" {
		var data []any
		j := json.NewEncoder(w)
		j.SetEscapeHTML(false)
		if opts.indentJSON {
			j.SetIndent("", "    ")
		}
		for _, k := range slices.Sorted(maps.Keys(responses)) {
			resp := responses[k]
			for _, i := range opts.ignoreerr {
				if errors.Is(resp.Err, i) {
					continue
				}
			}
			if resp.Err != nil {
				continue
			}
			if resp.Value != nil {
				data = append(data, resp.Value)
			}
			if len(resp.Values) > 0 {
				data = append(data, resp.Values...)
			}
		}

		j.Encode(data)
		return nil
	}

	r, err := reporter.NewReporter(format, w, reporterOptions...)
	if err != nil {
		return err
	}

	if err = r.Prepare(reporter.Report{
		Columns:         headings,
		ScrambleColumns: []string{},
	}); err != nil {
		return err
	}

	var rows [][]string
	if len(prequel) > 0 {
		rows = prequel
	}

RESPONSES:
	for _, k := range slices.Sorted(maps.Keys(responses)) {
		resp := responses[k]
		for _, i := range opts.ignoreerr {
			if errors.Is(resp.Err, i) {
				continue RESPONSES
			}
		}
		if resp.Err != nil {
			continue
		}
		rows = append(rows, resp.Dataview.Table...)
	}

	// for name, value := range opts.headlines {
	// 	r.AddHeadline(name, value)
	// }

	r.AddHeadlines(opts.headlines)

	r.UpdateTable(headings, rows)
	r.Render()
	return nil
}

// Report iterates over responses and outputs a formatted response to
// writer.
//
// If responses.WriterSkipOnErr(true) is set then any response with a
// non-nil Err field, where errors are not ignored with
// responses.WriterIgnoreErr() or responses.WriterIgnoreErrs(), then the
// other outputs are skipped (even if the error writer is the default
// io.Discard). Errors then written as described below.
//
// If writer is a [*tabwriter.Writer] String and Strings are written
// with a trailing newline.
//
// If writer is a [*csv.Writer] then Strings and Rows are written.
//
// Otherwise if Value is not nil then it is treated as a slice of any
// values which are marshalled as a JSON array and written to writer. If
// Value is a slice then it is unrolled and each element is instead
// written as a top-level array element, allowing Value to contain
// an arrays of responses. Any non-empty String or any Strings
// elements are output with a trailing newline. Any newline already
// present is removed to ensure only one newline between lines.
//
// If an error writer is set with responses.WriteStderr() then all
// non-ignored errors are written out, prefixed with the
// Instance.String() and a colon. Note that this format may change if
// and when structured logging is introduced.
//
// Report calls Flush() after writing to CSV or Tab writers.
func (responses Responses) Report(writer any, options ...WriterOption) {
	var rows [][]string

	if len(responses) == 0 {
		return
	}
	opts := evalWriterOptions(options...)

	startedJSON := false

OUTER:
	for _, k := range slices.Sorted(maps.Keys(responses)) {
		resp := responses[k]
		if resp.Err != nil && opts.skiponerr {
			for _, i := range opts.ignoreerr {
				if errors.Is(resp.Err, i) {
					continue OUTER
				}
			}
		}

		switch w := writer.(type) {
		case *reporter.TabWriterReporter:
			rows = append(rows, resp.Dataview.Table...)
		case *tabwriter.Writer:
			// if resp.Summary != "" {
			// 	fmt.Fprintf(w, "%s\n", resp.Summary)
			// }
			for _, line := range resp.ResultText {
				if line != "" {
					fmt.Fprintf(w, "%s\n", line)
				}
			}
		case *csv.Writer:
			w.WriteAll(resp.Dataview.Table)
			// w.WriteAll(resp.Rows) // WriteAll calls Flush()
		case io.Writer:
			// json from values, a bit painful - fix later if Values is
			// not nil then format - but no merging, yet. Use Formatted
			// instead.
			if len(resp.Values) > 0 {
				var b bytes.Buffer
				j := json.NewEncoder(&b)
				j.SetEscapeHTML(false)
				if opts.indentJSON {
					j.SetIndent("    ", "    ")
				}
				j.Encode(resp.Values)
				b.WriteTo(w)
				continue
			}

			// only support for an array of "Values", which is unrolled
			if resp.Value != nil && (opts.outputFields == 0 || opts.outputFields&outputFieldValue != 0) {
				if opts.asJSON {
					// encode to a buffer so we can strip the trailing newline
					var b bytes.Buffer
					j := json.NewEncoder(&b)
					j.SetEscapeHTML(false)
					if opts.indentJSON {
						j.SetIndent("    ", "    ")
					}

					if reflect.TypeOf(resp.Value).Kind() == reflect.Slice {
						s := reflect.ValueOf(resp.Value)
						for i := 0; i < s.Len(); i++ {
							if s.Index(i).IsValid() {
								if !startedJSON {
									fmt.Fprint(w, "[")
									startedJSON = true
								} else {
									fmt.Fprint(w, ",")
								}
								if opts.indentJSON {
									fmt.Fprint(w, "\n    ")
								}
								j.Encode(s.Index(i).Interface())
								if b.Len() > 1 {
									b.Truncate(b.Len() - 1)
									b.WriteTo(w)
								}
							}
						}
					} else {
						if !startedJSON {
							fmt.Fprint(w, "[")
							startedJSON = true
						} else {
							fmt.Fprint(w, ",")
						}
						if opts.indentJSON {
							fmt.Fprint(w, "\n    ")
						}
						j.Encode(resp.Value)

						if b.Len() > 1 {
							b.Truncate(b.Len() - 1)
							b.WriteTo(w)
						}
					}
				} else {
					fmt.Fprintf(w, opts.prefixformat, resp.Instance)
					fmt.Fprintf(w, "%s", resp.Value)
					fmt.Fprint(w, opts.suffix)
				}
			}

			if len(resp.Completed) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldCompleted != 0) {
				fmt.Fprintf(w, opts.prefixformat, resp.Instance)
				fmt.Fprint(w, joinNatural(resp.Completed...))
				fmt.Fprint(w, opts.suffix)
			}

			if len(resp.ResultText) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldDetails != 0) {
				for _, s := range resp.ResultText {
					fmt.Fprintln(w, strings.TrimSuffix(s, "\n"))
				}
			}

		default:
			log.Fatal().Msgf("unknown writer type %T", writer)
		}
	}

	if startedJSON {
		if opts.indentJSON {
			fmt.Fprint(writer.(io.Writer), "\n")
		}
		fmt.Fprintln(writer.(io.Writer), "]")
	}

	if w, ok := writer.(*reporter.TabWriterReporter); ok {
		w.UpdateTable(w.Columns, rows)
		w.Render()
		w.Close()
	}

	if w, ok := writer.(*tabwriter.Writer); ok {
		w.Flush()
	}

	if opts.stderr != io.Discard {
		for _, k := range slices.Sorted(maps.Keys(responses)) {
			r := responses[k]
			errored := false
			ignored := false
			if r.Err != nil && opts.skiponerr {
				for _, i := range opts.ignoreerr {
					if errors.Is(r.Err, i) {
						ignored = true
						break
					}
				}
				if !ignored {
					fmt.Fprintf(opts.stderr, "%s: %s\n", r.Instance, r.Err)
					errored = true
				}
			}

			if !errored && !ignored && opts.showtimes {
				s := r.Duration.Seconds()
				fmt.Fprintf(opts.stderr, opts.timesformat, r.Instance, s)
			}
		}
	}
}

// Report writes a single response to the writer w given the options.
func (resp General) Report(writer any, options ...WriterOption) {
	opts := evalWriterOptions(options...)

	if resp.Err != nil && opts.skiponerr {
		var ignored bool
		for _, i := range opts.ignoreerr {
			if errors.Is(resp.Err, i) {
				ignored = true
			}
		}
		if !ignored {
			return
		}
	}

	switch w := writer.(type) {
	case *reporter.TabWriterReporter:
		w.UpdateTable(w.Columns, resp.Dataview.Table)
		// w.UpdateTable(w.Columns, resp.Rows)
		w.Render()
		w.Close()
	case *tabwriter.Writer:
		// if resp.Summary != "" {
		// 	fmt.Fprintf(w, "%s\n", resp.Summary)
		// }
		for _, line := range resp.ResultText {
			if line != "" {
				fmt.Fprintf(w, "%s\n", line)
			}
		}
		w.Flush()
	case *csv.Writer:
		w.WriteAll(resp.Dataview.Table)
		// w.WriteAll(resp.Rows) // WriteAll calls Flush()
	case io.Writer:
		if resp.Value != nil && (opts.outputFields == 0 || opts.outputFields&outputFieldValue != 0) {
			if opts.asJSON {
				b, err := json.MarshalIndent(resp.Value, "    ", "    ")
				if err != nil {
					log.Error().Err(err).Msg("failed to marshal value to JSON")
					return
				}
				fmt.Fprint(w, string(b))
			} else {
				fmt.Fprintf(w, opts.prefixformat, resp.Instance)
				fmt.Fprintf(w, "%s", resp.Value)
				fmt.Fprint(w, opts.suffix)
			}
		}

		if len(resp.Completed) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldCompleted != 0) {
			fmt.Fprintf(w, opts.prefixformat, resp.Instance)
			fmt.Fprint(w, joinNatural(resp.Completed...))
			fmt.Fprint(w, opts.suffix)
		}

		if len(resp.ResultText) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldDetails != 0) {
			for _, s := range resp.ResultText {
				fmt.Fprintln(w, strings.TrimSuffix(s, "\n"))
			}
		}

	default:
		log.Fatal().Msgf("unknown writer type %T", writer)
	}

	if opts.stderr != io.Discard {
		errored := false
		ignored := false
		if resp.Err != nil {
			for _, i := range opts.ignoreerr {
				if errors.Is(resp.Err, i) {
					ignored = true
					break
				}
			}
			if !ignored {
				fmt.Fprintf(opts.stderr, "%s: %s\n", resp.Instance, resp.Err)
				errored = true
			}
		}

		if !errored && !ignored && opts.showtimes {
			s := resp.Duration.Seconds()
			fmt.Fprintf(opts.stderr, opts.timesformat, resp.Instance, s)
		}
	}
}

// joinNatural joins words with commas except the last pair, which are
// joined with an 'and'. No words results in empty string, one word is
// returned as-is and two words with 'and' etc.
func joinNatural(words ...string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	// case 2:
	// 	return words[0] + " and " + words[1]
	default:
		return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
	}
}
