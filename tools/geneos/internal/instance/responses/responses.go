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
	"os"
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
type Responses map[string]*Response

// Response is a consolidated set of responses from commands
type Response struct {
	Instance  geneos.Instance
	Summary   string     // single line response
	Details   []string   // multiple lines of output
	Completed []string   // simple past tense verbs of completed actions, e.g. "stopped", "started" etc.
	Value     any        // arbitrary value (typically for JSON output)
	Rows      [][]string // rows of values (for CSV)
	Start     time.Time
	Finish    time.Time
	Err       error
}

// NewResponse returns a pointer to an initialised Response structure,
// using instance i. The Start time is set to time.Now().
func NewResponse(i geneos.Instance) *Response {
	return &Response{
		Instance: i,
		Start:    time.Now(),
	}
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
// performed and a single response is expected.
func MergeResponse(r1, r2 *Response) (resp *Response) {
	resp = NewResponse(r1.Instance)
	resp.Completed = append(r1.Completed, r2.Completed...)
	resp.Rows = append(r1.Rows, r2.Rows...)
	resp.Err = errors.Join(r1.Err, r2.Err)

	resp.Start = r1.Start
	if resp.Start.IsZero() {
		resp.Start = r2.Start
	}

	resp.Finish = r2.Finish
	if resp.Finish.IsZero() {
		resp.Finish = r1.Finish
	}

	switch {
	case r1.Value == nil:
		resp.Value = r2.Value
	case r1.Value != nil && r2.Value != nil:
		resp.Value = []any{r1.Value, r2.Value}
	default:
		resp.Value = r1.Value
	}

	switch {
	case r1.Summary != "" && r2.Summary != "":
		resp.Details = append(resp.Details, r1.Summary, r2.Summary)
	case r1.Summary != "":
		resp.Summary = r1.Summary
	case r2.Summary != "":
		resp.Summary = r2.Summary
	}

	r1.Details = append(r1.Details, r2.Details...)
	return
}

// Formatted outputs the responses as a report in the specified format to
// writer w. headings are the column headings to use. prequel is any
// rows to add before the response rows.
//
// options are any reporter.ReporterOptions to control the output.
//
// If any response has a non-nil Err field then it is skipped.
func (responses Responses) Formatted(w io.Writer, format string, headings []string, prequel [][]string, options ...any) (err error) {
	r, err := reporter.NewReporter(format, w, options...)
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

	for _, k := range slices.Sorted(maps.Keys(responses)) {
		resp := responses[k]
		if resp.Err != nil {
			continue
		}
		rows = append(rows, resp.Rows...)
	}

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
func (responses Responses) Report(writer any, options ...WriterOptions) {
	var rows [][]string

	if len(responses) == 0 {
		return
	}
	opts := evalWriterOptions(options...)

	startedJSON := false

	for _, k := range slices.Sorted(maps.Keys(responses)) {
		resp := responses[k]
		if resp.Err != nil && opts.skiponerr {
			var ignored bool
			for _, i := range opts.ignoreerr {
				if errors.Is(resp.Err, i) {
					ignored = true
				}
			}
			if !ignored {
				continue
			}
		}

		switch w := writer.(type) {
		case *reporter.TabWriterReporter:
			rows = append(rows, resp.Rows...)
		case *tabwriter.Writer:
			if resp.Summary != "" {
				fmt.Fprintf(w, "%s\n", resp.Summary)
			}
			for _, line := range resp.Details {
				if line != "" {
					fmt.Fprintf(w, "%s\n", line)
				}
			}
		case *csv.Writer:
			w.WriteAll(resp.Rows) // WriteAll calls Flush()
		case io.Writer:
			// json from values, a bit painful - fix later
			// only support for an array of "Values", which is unrolled
			if resp.Value != nil {
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

			// string(s) - append a newline unless one is present
			if resp.Summary != "" {
				fmt.Fprintf(w, opts.prefixformat, resp.Instance)
				fmt.Fprint(w, strings.TrimSuffix(resp.Summary, "\n"))
				fmt.Fprint(w, opts.suffix)
			}

			if len(resp.Completed) > 0 {
				fmt.Fprintf(w, opts.prefixformat, resp.Instance)
				fmt.Fprint(w, joinNatural(resp.Completed...))
				fmt.Fprint(w, opts.suffix)
			}

			for _, s := range resp.Details {
				fmt.Fprintln(w, strings.TrimSuffix(s, "\n"))
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
			if r.Err != nil {
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
				s := r.Finish.Sub(r.Start).Seconds()
				fmt.Fprintf(opts.stderr, opts.timesformat, r.Instance, s)
			}
		}
	}
}

// Report writes a single response to the writer w given the options.
func (resp Response) Report(writer any, options ...WriterOptions) {
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
		w.UpdateTable(w.Columns, resp.Rows)
		w.Render()
		w.Close()
	case *tabwriter.Writer:
		if resp.Summary != "" {
			fmt.Fprintf(w, "%s\n", resp.Summary)
		}
		for _, line := range resp.Details {
			if line != "" {
				fmt.Fprintf(w, "%s\n", line)
			}
		}
		w.Flush()
	case *csv.Writer:
		w.WriteAll(resp.Rows) // WriteAll calls Flush()
	case io.Writer:
		switch {
		case resp.Value != nil && (opts.outputFields == 0 || opts.outputFields&outputFieldValue != 0):
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

		// string(s) - append a newline unless one is present
		case resp.Summary != "" && (opts.outputFields == 0 || opts.outputFields&outputFieldSummary != 0):
			fmt.Fprintf(w, opts.prefixformat, resp.Instance)
			fmt.Fprint(w, strings.TrimSuffix(resp.Summary, "\n"))
			fmt.Fprint(w, opts.suffix)

		case len(resp.Completed) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldCompleted != 0):
			fmt.Fprintf(w, opts.prefixformat, resp.Instance)
			fmt.Fprint(w, joinNatural(resp.Completed...))
			fmt.Fprint(w, opts.suffix)

		case len(resp.Details) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldDetails != 0):
			for _, s := range resp.Details {
				fmt.Fprintln(w, strings.TrimSuffix(s, "\n"))
			}

		default:
			// nothing to write
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
			s := resp.Finish.Sub(resp.Start).Seconds()
			fmt.Fprintf(opts.stderr, opts.timesformat, resp.Instance, s)
		}
	}
}

// WriteHTML will structure the responses in a way that can be displayed
// well in an HTML container. Currently does nothing.
func (responses Responses) WriteHTML(writer any, options ...WriterOptions) {
	if len(responses) == 0 {
		return
	}
	// opts := evalWriterOptions(options...)

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

// bitmap of types of output to limit to
type outputFields int

const (
	outputFieldSummary outputFields = 1 << iota
	outputFieldDetails
	outputFieldCompleted
	outputFieldValue
)

type writeOptions struct {
	stderr       io.Writer
	outputFields outputFields
	ignoreerr    []error
	skiponerr    bool
	showtimes    bool
	timesformat  string // first arg instance, second arg duration
	prefixformat string // prefix plain output with this format, parameter is instance name
	suffix       string // trailing suffix after each response, default "\n"
	asJSON       bool   // output each value as (unrolled) JSON. false is output using plain Print()
	indentJSON   bool
}

var globalWriteOptions = writeOptions{
	stderr:       os.Stderr,
	ignoreerr:    []error{os.ErrProcessDone, geneos.ErrNotSupported},
	skiponerr:    true,
	timesformat:  "%s: command finished in %.3fs\n",
	prefixformat: "%s ",
	suffix:       "\n",
	asJSON:       true,
}

// WriterOptions controls to behaviour of the responses.Write method
type WriterOptions func(*writeOptions)

func evalWriterOptions(options ...WriterOptions) *writeOptions {
	opts := globalWriteOptions
	for _, o := range options {
		o(&opts)
	}
	return &opts
}

// IndentJSON sets the JSON indentation to true or false for the output
// of Values in responses.Write
func IndentJSON(indent bool) WriterOptions {
	return func(wo *writeOptions) {
		wo.indentJSON = indent
	}
}

// StderrWriter sets the writer to use for errors. It defaults to
// os.StderrWriter
func StderrWriter(stderr io.Writer) WriterOptions {
	return func(wo *writeOptions) {
		wo.stderr = stderr
	}
}

// IgnoreErr adds err to the list of errors for responses.Write to skip.
func IgnoreErr(err error) WriterOptions {
	return func(wo *writeOptions) {
		wo.ignoreerr = append(wo.ignoreerr, err)
	}
}

// IgnoreErrs sets the errors that the responses.Write method will
// skip outputting. It replaces any existing set.
func IgnoreErrs(errs ...error) WriterOptions {
	return func(wo *writeOptions) {
		wo.ignoreerr = errs
	}
}

// SkipOnErr sets the behaviour of responses.Write regarding the
// output of other responses data if an error is present. If skip is
// true then any response that has a non-ignored error will output the
// error (subject to WriterStderr) and skip other returned data.
func SkipOnErr(skip bool) WriterOptions {
	return func(wo *writeOptions) {
		wo.skiponerr = skip
	}
}

// ShowTimes enables the output of the duration of each call. The
// format of the output can be changed using WriterTimingFormat.
func ShowTimes() WriterOptions {
	return func(wo *writeOptions) {
		wo.showtimes = true
	}
}

// TimingFormat sets the output format of any timing information.
// It is a Printf-style format with the instance (as a geneos.Instance)
// and the duration (as a time.Duration) as the two arguments.
func TimingFormat(format string) WriterOptions {
	return func(wo *writeOptions) {
		wo.timesformat = format
	}
}

// Prefix is the Printf-style format to prefix plain text output
// (only once per Lines). It can have one argument, the instance as a
// geneos.Instance. The default is `"%s "`.
func Prefix(prefix string) WriterOptions {
	return func(wo *writeOptions) {
		wo.prefixformat = prefix
	}
}

// Suffix is the suffix added to plain text output. The default is
// a single newline (`\n`).
func Suffix(suffix string) WriterOptions {
	return func(wo *writeOptions) {
		wo.suffix = suffix
	}
}

// PlainValue overrides the output of Value as JSON and instead it
// is written as a string, in the format `prefix + value as %s +
// suffix`, where prefix and suffix can be set using Prefix and
// Suffix respectively, if the defaults are not suitable.
func PlainValue() WriterOptions {
	return func(wo *writeOptions) {
		wo.asJSON = false
	}
}

// SummaryOnly makes responses.Write only output the Summary field.
func SummaryOnly() WriterOptions {
	return func(wo *writeOptions) {
		wo.outputFields = outputFieldSummary
	}
}

// DetailsOnly makes responses.Write only output the Details field.
func DetailsOnly() WriterOptions {
	return func(wo *writeOptions) {
		wo.outputFields = outputFieldDetails
	}
}

// CompletedOnly makes responses.Write only output the Completed field.
func CompletedOnly() WriterOptions {
	return func(wo *writeOptions) {
		wo.outputFields = outputFieldCompleted
	}
}

func ValueOnly() WriterOptions {
	return func(wo *writeOptions) {
		wo.outputFields = outputFieldValue
	}
}
