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
	"maps"
	"slices"

	"github.com/itrs-group/cordial/pkg/logger"
)

// Reporter is the interface implemented by all the report formats.
// Methods are not safe for concurrent use. A Reporter never modifies
// the tables passed to UpdateTable.
type Reporter interface {
	// Prepare initialises the current report and must be called before AddHeadline() or UpdateTable()
	Prepare(report Report) error

	// AddHeadline adds a single headline to the current report dataview / sheet etc.
	AddHeadline(name, value string)

	// AddHeadlines adds multiple headlines to the current report dataview / sheet etc.
	AddHeadlines(headlines map[string]string)

	// UpdateTable sets the main data table to the rows given. The first row must be the column names.
	UpdateTable(columns []string, rows [][]string)

	// Reset deletes an existing report, e.g. an existing Dataview from a previous run
	Reset(report Report) error

	// Render writes the current report to the destination selected with Prepare()
	Render() error

	// Close releases any resources for the whole reporter
	Close() error

	// Extension returns the file extension appropriate for this reporter type
	Extension() string
}

// Report TODO: Describe its purpose and intended use.
type Report struct {
	Name            string   `mapstructure:"report"`
	Title           string   `mapstructure:"name"`
	Columns         []string `mapstructure:"columns,omitempty"`
	ScrambleColumns []string `mapstructure:"scramble-columns,omitempty"`

	// FilePath is the pattern used to generate a file path for reports
	// when being written to a directory or a zip archive. It may
	// contain placeholders that will be replaced with report-specific
	// values.
	//
	// When not set the file path is generated from the report name,
	// lowercased and with spaces replaced by '-' (dash) and an extension
	// appropriate to the report format.
	//
	// Example: "reports/${value}_report.${extension}"
	// The ${value} placeholder is replaced with the split value
	// (e.g. gateway name) and ${extension} with the appropriate file
	// extension for the report format (e.g. "csv" or "xlsx").
	//
	// ${value} is only replaced when the report is being split by a
	// column value, e.g. gateway name. If the report is not split,
	// then ${value} is unset.
	//
	// The ${extension} placeholder is always replaced.
	FilePath string `mapstructure:"file-path,omitempty"`

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

type reporterCommon struct {
	Report
	format          string
	scrambleNames   bool
	scrambleFunc    func(string) string
	scrambleOptCols []string
}

// newReporterCommon returns the common reporter fields for format using
// the evaluated reporter options
func newReporterCommon(format string, ropts *reporterOptions) reporterCommon {
	return reporterCommon{
		format:          format,
		scrambleNames:   ropts.scrambleNames,
		scrambleFunc:    ropts.scrambleFunc,
		scrambleOptCols: ropts.scrambleColumns,
	}
}

// scrambleTable returns rows with the values in the columns to be
// scrambled replaced, if scrambling is enabled, otherwise rows. The
// columns to scramble are those given with the ScrambleColumns option
// and those in the Report passed to Prepare. The caller's rows are
// never modified.
func (c *reporterCommon) scrambleTable(columns []string, rows [][]string) [][]string {
	if !c.scrambleNames {
		return rows
	}
	return scrambleColumns(columns, slices.Concat(c.scrambleOptCols, c.ScrambleColumns), rows, c.scrambleFunc)
}

// headlineSet holds headlines in the order they were first added
type headlineSet struct {
	order  []string
	values map[string]string
}

// add sets a headline. A new name is added to the end of the order,
// and an existing one keeps its place and has its value replaced.
func (h *headlineSet) add(name, value string) {
	if h.values == nil {
		h.values = map[string]string{}
	}
	if _, ok := h.values[name]; !ok {
		h.order = append(h.order, name)
	}
	h.values[name] = value
}

// addAll adds the headlines in sorted name order, so the result does
// not depend on map iteration order
func (h *headlineSet) addAll(headlines map[string]string) {
	for _, name := range slices.Sorted(maps.Keys(headlines)) {
		h.add(name, headlines[name])
	}
}

func (h *headlineSet) len() int {
	return len(h.order)
}

var log = logger.Logger

var reporterFactories = make(map[string]func(string, io.Writer, ...any) (Reporter, error))

func registerReporter(format string, factory func(string, io.Writer, ...any) (Reporter, error)) {
	reporterFactories[format] = factory
}

// NewReporter returns a reporter for type format, which must be one of
// "toolkit", "csv", "tsv", "api", "dataview", "xlsx", "table" or
// "html". If a destination writer is required for the reporter type,
// then w should be the io.Writer to use. options are a list of options
// of either ReporterOptions or the options for the selected reporter
// type.
//
// TODO: add "column" output format, based on tabwriter.
func NewReporter(format string, w io.Writer, options ...any) (r Reporter, err error) {
	fn, ok := reporterFactories[format]
	if !ok {
		err = fmt.Errorf("unknown report type %q", format)
		return
	}
	r, err = fn(format, w, options...)
	return
}

// CollectOptions TODO: Document its behavior, inputs, and effects.
func CollectOptions[T any](options ...any) (opts []T) {
	for _, o := range options {
		if opt, ok := o.(T); ok {
			opts = append(opts, opt)
		}
	}
	return
}
