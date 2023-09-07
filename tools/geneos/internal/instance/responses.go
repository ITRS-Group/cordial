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

package instance

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/rs/zerolog/log"
)

type Responses []*Response

// Response is a consolidated set of responses from commands
type Response struct {
	Instance  geneos.Instance
	Line      string     // single line response,
	Lines     []string   // Lines of output
	Rows      [][]string // rows of values (for CSV)
	Value     any
	Start     time.Time
	Finish    time.Time
	Completed []string // simple past tense verbs of completed actions, e.g. "stopped", "started" etc.
	Err       error
}

// NewResponse returns a pointer to an intialised Response structure,
// using instance i. The Start time is set to time.Now().
func NewResponse(i geneos.Instance) *Response {
	return &Response{
		Instance: i,
		Start:    time.Now(),
	}
}

var _ sort.Interface = (Responses)(nil)

func (r Responses) Len() int { return len(r) }

func (r Responses) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Responses) Less(i, j int) bool {
	ci := r[i].Instance
	cj := r[j].Instance

	switch {
	case ci.Host().String() != cj.Host().String():
		return ci.Host().String() < cj.Host().String()
	case ci.Type().String() != cj.Type().String():
		return ci.Type().String() < cj.Type().String()
	case ci.Name() != cj.Name():
		return ci.Name() < cj.Name()
	default:
		return false
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
	case r1.Line != "" && r2.Line != "":
		resp.Lines = append(resp.Lines, r1.Line, r2.Line)
	case r1.Line != "":
		resp.Line = r1.Line
	case r2.Line != "":
		resp.Line = r2.Line
	}

	r1.Lines = append(r1.Lines, r2.Lines...)
	return
}

type SortInstanceResponses struct {
	Instances []geneos.Instance
	Results   []interface{}
}

func (s SortInstanceResponses) Len() int { return len(s.Instances) }

func (s SortInstanceResponses) Swap(i, j int) {
	s.Instances[i], s.Instances[j] = s.Instances[j], s.Instances[i]
	s.Results[i], s.Results[j] = s.Results[j], s.Results[i]
}

func (s SortInstanceResponses) Less(i, j int) bool {
	ci := s.Instances[i]
	cj := s.Instances[j]

	switch {
	case ci.Host().String() != cj.Host().String():
		return ci.Host().String() < cj.Host().String()
	case ci.Type().String() != cj.Type().String():
		return ci.Type().String() < cj.Type().String()
	case ci.Name() != cj.Name():
		return ci.Name() < cj.Name()
	default:
		return false
	}
}

// Write iterates over responses and outputs a formatted response to
// writer.
//
// If instance.WriterSkipOnErr(true) is set then any response with a
// non-nil Err field, where errors are not ignored with
// instance.WriterIgnoreErr() or instance.WriterIgnoreErrs(), then the
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
// any value is a slice then it is unrolled and each element is instead
// written as a top-level array element, allowing values to contain
// their own arrays of responses. Any non-empty String or any Strings
// elements are output with a trailing newline. Any newline already
// present is removed to ensure only one newline between lines.
//
// If an error writer is set with instance.WriteStderr() then all
// non-ignored errors are written out, prefixed with the
// Instance.String() and a colon. Note that this format may change if
// and when structured logging is introduced.
//
// Write calls Flush() after writing to CSV or Tab writers.
func (responses Responses) Write(writer any, options ...WriterOptions) {
	if len(responses) == 0 {
		return
	}
	opts := evalWriterOptions(options...)

	startedJSON := false

	for _, r := range responses {
		if r.Err != nil && opts.skiponerr {
			var ignored bool
			for _, i := range opts.ignoreerr {
				if errors.Is(r.Err, i) {
					ignored = true
				}
			}
			if !ignored {
				continue
			}
		}

		switch w := writer.(type) {
		case *tabwriter.Writer:
			if r.Line != "" {
				fmt.Fprintf(w, "%s\n", r.Line)
			}
			for _, line := range r.Lines {
				if line != "" {
					fmt.Fprintf(w, "%s\n", line)
				}
			}
		case *csv.Writer:
			w.WriteAll(r.Rows) // WriteAll calls Flush()
		case io.Writer:
			// json from values, a bit painful - fix later
			// only support for an array of "Values", which is unrolled
			if r.Value != nil {
				if opts.valuesasJSON {
					// encode to a buffer so we can strip the trailing newline
					var b bytes.Buffer
					j := json.NewEncoder(&b)
					j.SetEscapeHTML(false)
					if opts.indent {
						j.SetIndent("    ", "    ")
					}

					if reflect.TypeOf(r.Value).Kind() == reflect.Slice {
						s := reflect.ValueOf(r.Value)
						for i := 0; i < s.Len(); i++ {
							if s.Index(i).IsValid() {
								if !startedJSON {
									fmt.Fprint(w, "[")
									startedJSON = true
								} else {
									fmt.Fprint(w, ",")
								}
								if opts.indent {
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
						if opts.indent {
							fmt.Fprint(w, "\n    ")
						}
						j.Encode(r.Value)

						if b.Len() > 1 {
							b.Truncate(b.Len() - 1)
							b.WriteTo(w)
						}
					}
				} else {
					fmt.Fprintf(w, opts.prefixformat, r.Instance)
					fmt.Fprintf(w, "%s", r.Value)
					fmt.Fprint(w, opts.suffix)
				}
			}

			// string(s) - append a newline unless one is present
			if r.Line != "" {
				fmt.Fprintf(w, opts.prefixformat, r.Instance)
				fmt.Fprint(w, strings.TrimSuffix(r.Line, "\n"))
				fmt.Fprint(w, opts.suffix)
			}

			if len(r.Completed) > 0 {
				fmt.Fprintf(w, opts.prefixformat, r.Instance)
				fmt.Fprint(w, joinNatural(r.Completed...))
				fmt.Fprint(w, opts.suffix)
			}

			for _, s := range r.Lines {
				fmt.Fprintln(w, strings.TrimSuffix(s, "\n"))
			}

		default:
			log.Fatal().Msgf("unknown writer type %T", writer)
		}
	}

	if startedJSON {
		if opts.indent {
			fmt.Fprint(writer.(io.Writer), "\n")
		}
		fmt.Fprintln(writer.(io.Writer), "]")
	}

	if w, ok := writer.(*tabwriter.Writer); ok {
		w.Flush()
	}

	if opts.stderr != io.Discard {
		for _, r := range responses {
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

type writeOptions struct {
	indent       bool
	stderr       io.Writer
	ignoreerr    []error
	skiponerr    bool
	showtimes    bool
	timesformat  string // first arg instance, second arg duration
	prefixformat string // prefix plain output with this format, parameter is instance name
	suffix       string // trailing suffix after each response, default "\n"
	valuesasJSON bool   // output each value as (unrolled) JSON. false is output using plain Print()
}

var globalWriteOptions = writeOptions{
	stderr:       os.Stderr,
	ignoreerr:    []error{os.ErrProcessDone, geneos.ErrNotSupported},
	skiponerr:    true,
	timesformat:  "%s: command finished in %.3fs\n",
	prefixformat: "%s ",
	suffix:       "\n",
	valuesasJSON: true,
}

// WriterOptions controls to behaviour of the instance.Write method
type WriterOptions func(*writeOptions)

func evalWriterOptions(options ...WriterOptions) *writeOptions {
	opts := globalWriteOptions
	for _, o := range options {
		o(&opts)
	}
	return &opts
}

// WriterDefaultOptions sets and defaults for calls to instance.Write
//
// The defaults, unless otherwise set are to write errors to os.Stderr
// and to ignore os.ErrProcessDone and geneos.ErrNotSupported errors and
// to skip other outputs for each response on non-ignored errors.
func WriterDefaultOptions(options ...WriterOptions) {
	for _, o := range options {
		o(&globalWriteOptions)
	}
}

// WriterIndent sets the JSON indentation to true or false for the
// output of Values in instance.Write
func WriterIndent(indent bool) WriterOptions {
	return func(wo *writeOptions) {
		wo.indent = indent
	}
}

// WriteStderr sets the writer to use for errors. It defaults to
// os.Stderr
func WriterStderr(stderr io.Writer) WriterOptions {
	return func(wo *writeOptions) {
		wo.stderr = stderr
	}
}

// WriteIgnoreErr adds err to the list of errors for instance.Write to
// skip.
func WriterIgnoreErr(err error) WriterOptions {
	return func(wo *writeOptions) {
		wo.ignoreerr = append(wo.ignoreerr, err)
	}
}

// WriterIgnoreErrs sets the errors that the instance.Write method will
// skip outputting. It replaces any existing set.
func WriterIgnoreErrs(errs ...error) WriterOptions {
	return func(wo *writeOptions) {
		wo.ignoreerr = errs
	}
}

// WriterSkipOnErr sets the behaviour of instance.Write regarding the
// output of other responses data if an error is present. If skip is
// true then any response that has a non-ignored error will output the
// error (subject to WriterStderr) and skip other returned data.
func WriterSkipOnErr(skip bool) WriterOptions {
	return func(wo *writeOptions) {
		wo.skiponerr = skip
	}
}

// WriterShowTimes enables the output of the duration of each call. The
// format of the output can be changed using WriterTimingFormat.
func WriterShowTimes() WriterOptions {
	return func(wo *writeOptions) {
		wo.showtimes = true
	}
}

// WriterTimingFormat sets the output format of any timing information.
// It is a Printf-style format with the instance (as a geneos.Instance)
// and the duration (as a time.Duration) as the two arguments.
func WriterTimingFormat(format string) WriterOptions {
	return func(wo *writeOptions) {
		wo.timesformat = format
	}
}

// WriterPrefix is the Printf-style format to prefix plain text output
// (only once per Lines). It can have one argument, the instance asd a
// geneos.Instance. The default is `"%s "`.
func WriterPrefix(prefix string) WriterOptions {
	return func(wo *writeOptions) {
		wo.prefixformat = prefix
	}
}

// WriterSuffix is the suffix added to plain text output. The default is
// a single newline (`\n`).
func WriterSuffix(suffix string) WriterOptions {
	return func(wo *writeOptions) {
		wo.suffix = suffix
	}
}

// WriterPlainValue overrides the output of Value as JSON and instead it
// is written as a string, in the format `prefix + value as %s +
// suffix`, where prefix and suffix can be set ising WriterPrefix and
// WriterSuffix respectively, if the defaults are not suitable.
func WriterPlainValue() WriterOptions {
	return func(wo *writeOptions) {
		wo.valuesasJSON = false
	}
}
