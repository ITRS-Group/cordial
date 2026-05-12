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
	"encoding/csv"
	"fmt"
	"io"
	"maps"
	"strings"
)

type ToolkitReporter struct {
	reporterCommon
	w               io.Writer
	c               *csv.Writer
	table           [][]string
	headlines       map[string]string
	scrambleColumns []string
}

// ensure that *ToolkitReporter conforms to the Reporter interface
var _ Reporter = (*ToolkitReporter)(nil)

func newToolkitReporter(w io.Writer, opts *reporterOptions) *ToolkitReporter {
	_ = opts
	return &ToolkitReporter{
		reporterCommon: reporterCommon{
			format:        "toolkit",
			scrambleNames: opts.scrambleNames,
		},
		w:         w,
		c:         csv.NewWriter(w),
		headlines: make(map[string]string),
	}
}

func (t *ToolkitReporter) Prepare(report Report) error {
	t.scrambleColumns = report.ScrambleColumns
	return nil
}

// AddHeadline adds a headline to the reporter.
func (t *ToolkitReporter) AddHeadline(name, value string) {
	t.headlines[name] = value
}

// AddHeadlines adds multiple headlines to the reporter.
func (t *ToolkitReporter) AddHeadlines(headlines map[string]string) {
	maps.Copy(t.headlines, headlines)
}

// UpdateTable sets the main data table to the rows given. The first row
// must be the column names. It overwrites any existing table data.
func (t *ToolkitReporter) UpdateTable(columns []string, rows [][]string) {
	if t.scrambleNames {
		scrambleColumns(columns, t.scrambleColumns, rows)
	}
	// escape commas and newlines in the data, as Toolkit CSV format does not support quoting
	for i, row := range rows {
		for j, cell := range row {
			// replace commas and newlines with spaces or escaped newlines
			cell = strings.ReplaceAll(cell, ",", " ")
			cell = strings.ReplaceAll(cell, "\n", "\\n")
			rows[i][j] = cell
		}
	}
	t.table = append([][]string{columns}, rows...)
}

func (t *ToolkitReporter) Reset(report Report) (err error) {
	// do nothing
	return
}

func (t *ToolkitReporter) Render() {
	t.c.WriteAll(t.table)
	t.c.Flush()
	for name, value := range t.headlines {
		fmt.Fprintf(t.w, "<!>%s,%s\n", name, value)
	}
	t.table = [][]string{}
	t.headlines = map[string]string{}

}

// Close will call Close on the writer if it has a Close method
func (t *ToolkitReporter) Close() {

	if c, ok := t.w.(io.Closer); ok {
		c.Close()
	}
}

func (t *ToolkitReporter) Extension() string {
	return "txt"
}
