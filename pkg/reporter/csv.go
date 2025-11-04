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
	"io"
)

type CSVReporter struct {
	ReporterCommon
	w     io.Writer
	c     *csv.Writer
	table [][]string
	// scrambleNames   bool
	scrambleColumns []string
}

// ensure that *CSVReporter conforms to the Reporter interface
var _ Reporter = (*CSVReporter)(nil)

func newCSVReporter(w io.Writer, opts *reporterOptions) *CSVReporter {
	return &CSVReporter{
		ReporterCommon: ReporterCommon{scrambleNames: opts.scrambleNames},
		w:              w,
		c:              csv.NewWriter(w),
	}
}

func (t *CSVReporter) Prepare(report Report) error {
	t.scrambleColumns = report.ScrambleColumns
	return nil
}

func (t *CSVReporter) AddHeadline(name, value string) {
	// No headlines in CSV format
}

func (t *CSVReporter) UpdateTable(columns []string, data [][]string) {
	if t.scrambleNames {
		scrambleColumns(columns, t.scrambleColumns, data)
	}
	t.table = append([][]string{columns}, data...)
}

func (t *CSVReporter) Remove(report Report) (err error) {
	// do nothing
	return
}

func (t *CSVReporter) Render() {
	t.c.WriteAll(t.table)
	t.c.Flush()
	t.table = [][]string{}
}

// Close will call Close on the writer if it has a Close method
func (t *CSVReporter) Close() {
	if c, ok := t.w.(io.Closer); ok {
		c.Close()
	}
}
