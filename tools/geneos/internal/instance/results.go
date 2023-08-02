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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/rs/zerolog/log"
)

type Results []Result

type Result struct {
	Instance geneos.Instance
	String   string
	Strings  []string
	Value    any
	Err      error
}

var _ sort.Interface = (Results)(nil)

func (r Results) Len() int { return len(r) }

func (r Results) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Results) Less(i, j int) bool {
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

type SortInstanceResults struct {
	Instances []geneos.Instance
	Results   []interface{}
}

func (s SortInstanceResults) Len() int { return len(s.Instances) }

func (s SortInstanceResults) Swap(i, j int) {
	s.Instances[i], s.Instances[j] = s.Instances[j], s.Instances[i]
	s.Results[i], s.Results[j] = s.Results[j], s.Results[i]
}

func (s SortInstanceResults) Less(i, j int) bool {
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

// appendUnrolledResults checks the type of result and appends to results
// appropriately.
//
// If result is a slice of a basic type it is appended as-is, as are non
// slice types
//
// If result is a slice of any other type then each element is appended
// individually
//
// Any nil values (or invalid values) are skipped
func appendUnrolledResults(in []any, result any) (out []any) {
	t := reflect.TypeOf(result)
	v := reflect.ValueOf(result)
	if !v.IsValid() {
		return
	}
	if t.Kind() != reflect.Slice {
		return append(in, result)
	}
	if t.Elem().Kind() == reflect.String {
		return append(in, result)
	}
	if v.IsZero() {
		return
	}
	out = in
	for i := 0; i < v.Len(); i++ {
		if !v.Index(i).IsValid() || v.Index(i).IsZero() {
			continue
		}
		out = append(out, v.Index(i).Interface())
	}
	return
}

// WriteResultsToCSVWriter sends all slices of strings to the writer w
//
// results can be a slice or a slice of slices or a mix of both, it will
// recurse into those
func WriteResultsToCSVWriter(w *csv.Writer, results []any) (err error) {
	for _, result := range results {
		switch row := result.(type) {
		case []string:
			if err = w.Write(row); err != nil {
				return
			}
		case [][]string:
			for _, r := range row {
				if len(r) == 0 {
					continue
				}
				if err = w.Write(r); err != nil {
					return
				}
			}
		default:
			log.Error().Msgf("unexpected row type %T", result)
		}
	}
	return
}

// WriteResultsAsJSON writes the JSON encoding of results to writer w
//
// HTML escaping is turned off. If indent is true then the output is
// indented by four spaces per level for human presentation.
func WriteResultsAsJSON(w io.Writer, results any, indent bool) (err error) {
	j := json.NewEncoder(w)
	j.SetEscapeHTML(false)
	if indent {
		j.SetIndent("", "    ")
	}
	return j.Encode(results)
}

// WriteResultsStrings writes the elements of results to the writer w.
// If the element is a plain string then it is written with a trailing
// newline unless the string is only a newline, in which case only the
// newline is written. If the element is a slice of strings then each
// one is written out using the same rules.
func WriteResultsStrings(w io.Writer, results []any) (err error) {
	for _, result := range results {

		switch row := result.(type) {
		case string:
			if row == "\n" {
				fmt.Fprintln(w, "")
			} else {
				fmt.Fprintln(w, row)
			}
		case []string:
			for _, r := range row {
				if len(r) == 0 {
					continue
				}
				if r == "\n" {
					fmt.Fprintln(w, "")
				} else {
					fmt.Fprintln(w, r)
				}
			}
		default:
			log.Error().Msgf("unexpected result type %T", result)
		}
	}
	return
}
