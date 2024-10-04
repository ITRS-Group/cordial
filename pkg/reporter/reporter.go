/*
Copyright © 2024 ITRS Group

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

// The reporter package provides a simple interface to generating Geneos
// dataviews, with headlines and a data table, either though the XML-RPC
// API, as Toolkit compatible CSV, XLSX workbooks or a number of other
// formats.
package reporter

import (
	"fmt"
	"io"
	"slices"
)

type Reporter interface {
	// Prepare initialises the current report and must be called before AddHeadline() or UpdateTable()
	Prepare(report Report) error

	// AddHeadline adds a single headline to the current report dataview / sheet etc.
	AddHeadline(name, value string)

	// UpdateTable sets the main data table to the rows given. The first row must be the column names.
	UpdateTable(headings []string, rows [][]string)

	// Remove deletes an existing report, e.g. an existing Dataview from a previous run
	Remove(report Report) error

	// Flush writes the current report to the destination selected with Prepare()
	Flush()

	// Close releases any resources for the whole reporter
	Close()
}

type ReporterCommon struct {
	scrambleFunc func(in string) string
}

type Report struct {
	Name            string   `mapstructure:"report"`
	Title           string   `mapstructure:"name"`
	Columns         []string `mapstructure:"columns,omitempty"`
	ScrambleColumns []string `mapstructure:"scramble-columns,omitempty"`

	// Report format specific settings

	Dataview struct {
		Group  string `mapstructure:"group,omitempty"`
		Enable *bool  `mapstructure:"enable,omitempty"`
	} `mapstructure:"dataview,omitempty"`

	XLSX struct {
		// xlsx specific
		Enable            *bool               `mapstructure:"enable,omitempty"`
		FreezeColumn      string              `mapstructure:"freeze-to-column"`
		ConditionalFormat []ConditionalFormat `mapstructure:"conditional-format,omitempty"`
	} `mapstructure:"xlsx,omitempty"`
}

type ReporterOptions func(*reporterOptions)

type reporterOptions struct {
	scrambleFunc    func(string) string
	scrambleColumns []string
}

func evalReporterOptions(options ...ReporterOptions) (ro *reporterOptions) {
	ro = &reporterOptions{
		scrambleFunc: scrambleWords,
	}
	for _, opt := range options {
		opt(ro)
	}
	return
}

func ScrambleFunc(fn func(in string) string) ReporterOptions {
	return func(ro *reporterOptions) {
		ro.scrambleFunc = fn
	}
}

func ScrambleColumns(columns []string) ReporterOptions {
	return func(ro *reporterOptions) {
		ro.scrambleColumns = columns
	}
}

// NewReporter returns a reporter for type t, which must be one of
// "toolkit", "csv", "api", "dataview", "xlsx", "table" or "html". If a
// destination writer is appropriate for the reporter type, then w
// should be the io.Writer to use. options are a list of options of
// either ReporterOptions or the options for the selected reporter type.
func NewReporter(t string, w io.Writer, options ...any) (r Reporter, err error) {
	// pull out general reporter options, which are passed to each
	// reporter factory method
	var ro []ReporterOptions
	options = slices.DeleteFunc(options, func(o any) bool {
		if a, ok := o.(ReporterOptions); ok {
			ro = append(ro, a)
			return true
		}
		return false
	})
	opts := evalReporterOptions(ro...)

	switch t {
	case "csv", "toolkit":
		r = newToolkitReporter(w, opts)
	case "api", "dataview":
		var apioptions []APIReporterOptions
		for _, o := range options {
			if a, ok := o.(APIReporterOptions); ok {
				apioptions = append(apioptions, a)
			} else {
				panic("wrong option type")
			}
		}
		r, err = newAPIReporter(opts, apioptions...)
	case "xlsx":
		var xlsxoptions []XLSXReporterOptions
		for _, o := range options {
			if x, ok := o.(XLSXReporterOptions); ok {
				xlsxoptions = append(xlsxoptions, x)
			} else {
				panic("wrong option type")
			}
		}
		r = newXLSXReporter(w, opts, xlsxoptions...)
	case "table", "html", "markdown", "md", "tsv":
		var fmtoptions = []FormattedReporterOptions{
			RenderAs(t),
		}
		for _, o := range options {
			if f, ok := o.(FormattedReporterOptions); ok {
				fmtoptions = append(fmtoptions, f)
			} else {
				panic("wrong option type")
			}
		}
		r = newFormattedReporter(w, opts, fmtoptions...)
	default:
		err = fmt.Errorf("unknown report type %q", t)
		return
	}

	return
}
