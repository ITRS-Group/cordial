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

type Responses []Response

type Response struct {
	Instance geneos.Instance
	String   string
	Strings  []string
	Rows     [][]string
	Value    any
	Start    time.Time
	Finish   time.Time
	Err      error
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

// Write iterates over responses and outputs the response to writer.
//
// If instance.WriterSkipOnErr(true) is set then any response with a
// non-nil Err field, where errors are not ignored with
// instance.WriterIgnoreErr() or instance.WriterIgnoreErrs(), then the
// other outputs are skipped (even if the error writer is the default
// io.Discard). Errors then written as described below.
//
// If writer is a [*tabwriter.Writer] String and Strings are written with a
// trailing newline.
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
func (responses Responses) Write(writer any, options ...WriterOptions) {
	if len(responses) == 0 {
		return
	}
	opts := evalWriterOptions(options...)

	started := false

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
			if r.String != "" {
				fmt.Fprintf(w, "%s\n", r.String)
			}
			for _, line := range r.Strings {
				if line != "" {
					fmt.Fprintf(w, "%s\n", line)
				}
			}
		case *csv.Writer:
			if len(r.Strings) > 0 {
				w.Write(r.Strings)
			}
			if len(r.Rows) > 0 {
				w.WriteAll(r.Rows)
			}
		case io.Writer:
			// json from values, a bit painful - fix later
			// only support for an array of "Values", which is unrolled
			if r.Value != nil {
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
							if !started {
								fmt.Fprint(w, "[")
								started = true
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
					if !started {
						fmt.Fprint(w, "[")
						started = true
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
			}

			// string(s) - append a newline unless one is present
			if r.String != "" {
				fmt.Fprintln(w, strings.TrimSuffix(r.String, "\n"))
			}

			for _, s := range r.Strings {
				fmt.Fprintln(w, strings.TrimSuffix(s, "\n"))
			}

		default:
			log.Fatal().Msgf("unknown writer type %T", writer)
		}
	}
	if started {
		if opts.indent {
			fmt.Fprint(writer.(io.Writer), "\n")
		}
		fmt.Fprintln(writer.(io.Writer), "]")
	}

	if opts.stderr != io.Discard {
	ERRORS:
		for _, r := range responses {
			if r.Err != nil {
				for _, i := range opts.ignoreerr {
					if errors.Is(r.Err, i) {
						continue ERRORS
					}
				}
				fmt.Fprintf(opts.stderr, "%s: %s\n", r.Instance, r.Err)
			}
		}
	}
}

type writeOptions struct {
	indent    bool
	stderr    io.Writer
	ignoreerr []error
	skiponerr bool
}

var globalWriteOptions = writeOptions{
	stderr:    os.Stderr,
	ignoreerr: []error{os.ErrProcessDone, geneos.ErrNotSupported},
	skiponerr: true,
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
