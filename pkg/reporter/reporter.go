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
//
// First you must create a new Reporter
package reporter

import (
	"fmt"
	"io"
)

type Reporter interface {
	SetReport(report Report) error
	AddHeadline(name, value string)
	UpdateTable(rows ...[]string)
	Flush()
	Close()
}

type Report struct {
	Name            string   `mapstructure:"name"`
	Group           string   `mapstructure:"group,omitempty"`
	Columns         []string `mapstructure:"columns,omitempty"`
	ScrambleColumns []string `mapstructure:"scramble-columns,omitempty"`

	// API specific
	EnableForDataview *bool `mapstructure:"enable-for-dataview,omitempty"`

	// xlsx specific
	EnableForXLSX     *bool               `mapstructure:"enable-for-xlsx,omitempty"`
	FreezeColumn      string              `mapstructure:"freeze-to-column"`
	ConditionalFormat []ConditionalFormat `mapstructure:"conditional-format,omitempty"`
}

type ReportOptions func(*reportOptions)

type reportOptions struct {
	scrambleColumns []string
}

func evalReportOptions(options ...ReportOptions) (ro *reportOptions) {
	ro = &reportOptions{}
	for _, opt := range options {
		opt(ro)
	}
	return
}

func ScrambleColumns(columns []string) ReportOptions {
	return func(ro *reportOptions) {
		ro.scrambleColumns = columns
	}
}

func NewReporter(t string, w io.Writer, options ...any) (r Reporter, err error) {
	switch t {
	case "csv", "toolkit":
		r = NewToolkitReporter(w)
	case "api", "dataview":
		var apioptions []APIReporterOptions
		for _, o := range options {
			if a, ok := o.(APIReporterOptions); ok {
				apioptions = append(apioptions, a)
			} else {
				panic("wrong option type")
			}
		}
		r, err = NewAPIReporter(apioptions...)
	case "xlsx":
		var xlsxoptions []XLSXReporterOptions
		for _, o := range options {
			if x, ok := o.(XLSXReporterOptions); ok {
				xlsxoptions = append(xlsxoptions, x)
			} else {
				panic("wrong option type")
			}
		}
		r = NewXLSXReporter(w, xlsxoptions...)
	case "table", "html":
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
		r = NewFormattedReporter(w, fmtoptions...)
	default:
		err = fmt.Errorf("unknown report type %q", t)
		return
	}

	return
}
