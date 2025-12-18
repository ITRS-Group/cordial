/*
Copyright © 2025 ITRS Group

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

package reporter

// The "column" or "tabwriter" reporter outputs data in columns using
// the tabwriter package.
import (
	"io"
	"strings"
	"text/tabwriter"
)

// TabWriterReporter implements a Reporter using tabwriter to format
// output in columns.
type TabWriterReporter struct {
	reporterCommon
	writer               *tabwriter.Writer
	rows                 [][]string
	quoteFieldsWithSpace bool
}

func newTabWriterReporter(opts *reporterOptions, options ...TabWriterReporterOptions) *TabWriterReporter {
	_ = opts
	var opt TabWriterReporterOptions
	if len(options) > 0 {
		opt = evalTabWriterOptions(options...)
	}
	return &TabWriterReporter{
		writer:               tabwriter.NewWriter(opt.w, 3, 8, 2, ' ', 0),
		quoteFieldsWithSpace: opt.QuoteFieldsWithSpace,
	}
}

// Prepare initialises the TabWriterReporter.
func (t *TabWriterReporter) Prepare(report Report) error {
	t.Report = report
	return nil
}

// AddHeadline adds a headline (not used in tabwriter).
func (t *TabWriterReporter) AddHeadline(name, value string) {
	// No-op for tabwriter
}

// UpdateTable sets the headings and rows for the table.
func (t *TabWriterReporter) UpdateTable(headings []string, rows [][]string) {
	if len(headings) > 0 {
		t.Columns = headings
	}
	t.rows = rows
}

// Remove is a no-op for TabWriterReporter.
func (t *TabWriterReporter) Remove(report Report) error {
	return nil
}

// Render writes the table to the tabwriter.
func (t *TabWriterReporter) Render() {
	t.writer.Write([]byte(strings.Join(t.Columns, "\t") + "\n"))

	for _, row := range t.rows {
		t.writer.Write([]byte(strings.Join(row, "\t") + "\n"))
	}

	t.writer.Flush()
}

// Close releases resources for TabWriterReporter.
func (t *TabWriterReporter) Close() {
	if t.writer != nil {
		t.writer.Flush()
	}
}

// Extension returns the file extension for TabWriterReporter.
func (t *TabWriterReporter) Extension() string {
	return "txt"
}

type tabWriterReporterOptions struct {
	QuoteFieldsWithSpace bool
	w                    io.Writer
}

type TabWriterReporterOptions func(*tabWriterReporterOptions)

func evalTabWriterOptions(options ...TabWriterReporterOptions) (twro *tabWriterReporterOptions) {
	twro = &tabWriterReporterOptions{
		QuoteFieldsWithSpace: false,
		w:                    io.Discard,
	}
	for _, opt := range options {
		opt(twro)
	}
	return
}

// TabWriter is a TabWriterReporter option to set the output writer.

func TabWriter(w io.Writer) TabWriterReporterOptions {
	return func(twro *tabWriterReporterOptions) {
		twro.w = w
	}
}

// WithQuoteFieldsWithSpace is a TabWriterReporter option to quote fields
// containing spaces.
func WithQuoteFieldsWithSpace() TabWriterReporterOptions {
	return func(twro *tabWriterReporterOptions) {
		twro.QuoteFieldsWithSpace = true
	}
}
