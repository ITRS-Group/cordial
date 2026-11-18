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

package reporter

import (
	"io"
	"strings"
)

// ToolkitReporter writes reports in the Geneos Toolkit plugin format.
// Toolkit CSV does not support quoting, so no quoting is applied and
// commas and newlines in values are replaced.
type ToolkitReporter struct {
	reporterCommon
	w         io.Writer
	table     [][]string
	headlines headlineSet
}

// ensure that *ToolkitReporter conforms to the Reporter interface
var _ Reporter = (*ToolkitReporter)(nil)

func init() {
	registerReporter("toolkit", newToolkitReporter)
}

func newToolkitReporter(format string, w io.Writer, options ...any) (Reporter, error) {
	opts := evalReporterOptions(CollectOptions[ReporterOption](options...)...)
	return &ToolkitReporter{
		reporterCommon: newReporterCommon("toolkit", opts),
		w:              w,
	}, nil
}

// Prepare TODO: Document its behavior, inputs, and effects.
func (t *ToolkitReporter) Prepare(report Report) error {
	t.Report = report
	return nil
}

// AddHeadline adds a headline to the reporter. Headlines are output in
// the order they were first added.
func (t *ToolkitReporter) AddHeadline(name, value string) {
	t.headlines.add(name, value)
}

// AddHeadlines adds multiple headlines to the reporter, in sorted name
// order.
func (t *ToolkitReporter) AddHeadlines(headlines map[string]string) {
	t.headlines.addAll(headlines)
}

// toolkitCell makes a value safe for Toolkit CSV, which does not
// support quoting: commas are replaced with spaces and line breaks with
// escaped newlines.
func toolkitCell(cell string) string {
	return toolkitText(strings.ReplaceAll(cell, ",", " "))
}

// toolkitText escapes line breaks only, for the value of a headline,
// which may contain commas as it follows the first comma on the line
func toolkitText(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	return strings.ReplaceAll(s, "\n", "\\n")
}

func toolkitRow(row []string) string {
	cells := make([]string, len(row))
	for i, cell := range row {
		cells[i] = toolkitCell(cell)
	}
	return strings.Join(cells, ",") + "\n"
}

// UpdateTable sets the main data table to the rows given. The first row
// must be the column names. It overwrites any existing table data. The
// rows passed are not modified.
func (t *ToolkitReporter) UpdateTable(columns []string, rows [][]string) {
	rows = t.scrambleTable(columns, rows)

	t.table = append([][]string{columns}, rows...)
}

// Reset TODO: Document its behavior, inputs, and effects.
func (t *ToolkitReporter) Reset(report Report) (err error) {
	// do nothing
	return
}

// Render writes the table and then the headlines, in the order they
// were added, and clears them.
func (t *ToolkitReporter) Render() error {
	for _, row := range t.table {
		if _, err := io.WriteString(t.w, toolkitRow(row)); err != nil {
			return err
		}
	}
	for _, name := range t.headlines.order {
		if _, err := io.WriteString(t.w, "<!>"+toolkitCell(name)+","+toolkitText(t.headlines.values[name])+"\n"); err != nil {
			return err
		}
	}
	t.table = nil
	t.headlines = headlineSet{}
	return nil
}

// Close will call Close on the writer if it has a Close method
func (t *ToolkitReporter) Close() error {

	if c, ok := t.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// Extension TODO: Document its behavior, inputs, and effects.
func (t *ToolkitReporter) Extension() string {
	return "txt"
}
