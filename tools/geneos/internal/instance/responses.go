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
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
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

// WriteResponsesToCSVWriter sends all slices of strings to the writer w
//
// results can be one row in Strings or multiple rows in Rows, if both
// are set then Strings is output first
func WriteResponsesToCSVWriter(w *csv.Writer, results Responses) (err error) {
	for _, result := range results {
		if len(result.Strings) > 0 {
			w.Write(result.Strings)
		}
		if len(result.Rows) > 0 {
			w.WriteAll(result.Rows)
		}
	}
	return
}

// WriteResponsesToTabWriter sends all slices of strings to the writer
// w, terminating each line with a newline (`\n`)
//
// responses can contain both a String and a Strings slice, both are
// written if the contents are not empty
func WriteResponsesToTabWriter(w *tabwriter.Writer, responses Responses) (err error) {
	for _, result := range responses {
		if result.String != "" {
			fmt.Fprintf(w, "%s\n", result.String)
		}
		for _, line := range result.Strings {
			if line != "" {
				fmt.Fprintf(w, "%s\n", line)
			}
		}
	}
	return
}

// WriteResponsesAsJSON writes the JSON encoding of the Value field of
// each response in responses to writer w. If Value is a slice then it
// is unrolled, so it is not possible to JSON encode arrays (beyond the
// responses slice itself) with this method.
//
// HTML escaping is turned off. If indent is true then the output is
// indented by four spaces per level for human presentation.
func WriteResponsesAsJSON(w io.Writer, results Responses, indent bool) (err error) {
	values := []any{}
	for _, v := range results {
		if v.Value == nil {
			continue
		}

		// unroll any slices to the underlying elements
		if reflect.TypeOf(v.Value).Kind() == reflect.Slice {
			s := reflect.ValueOf(v.Value)
			for i := 0; i < s.Len(); i++ {
				if s.Index(i).IsValid() {
					values = append(values, s.Index(i).Interface())
				}

			}
		} else {
			values = append(values, v.Value)
		}
	}

	j := json.NewEncoder(w)
	j.SetEscapeHTML(false)
	if indent {
		j.SetIndent("", "    ")
	}
	return j.Encode(values)
}

// WriteResponseStrings writes the elements of results to the writer w.
// If the element is a plain string then it is written with a trailing
// newline unless the string is only a newline, in which case only the
// newline is written. If the element is a slice of strings then each
// one is written out using the same rules.
func WriteResponseStrings(w io.Writer, responses Responses) (err error) {
	for _, result := range responses {
		if result.String == "\n" {
			fmt.Fprintln(w, "")
		} else if result.String != "" {
			fmt.Fprintln(w, result.String)
		}

		for _, r := range result.Strings {
			// if len(r) == 0 {
			// 	continue
			// }
			if r == "\n" {
				fmt.Fprintln(w, "")
			} else {
				fmt.Fprintln(w, r)
			}
		}
	}
	return
}
