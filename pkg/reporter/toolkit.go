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
)

type ToolkitReporter struct {
	w               io.Writer
	c               *csv.Writer
	table           [][]string
	headlines       map[string]string
	scrambleColumns []string
}

// ensure that *ToolkitReporter conforms to the Reporter interface
var _ Reporter = (*ToolkitReporter)(nil)

func NewToolkitReporter(w io.Writer) *ToolkitReporter {
	return &ToolkitReporter{
		w:         w,
		c:         csv.NewWriter(w),
		headlines: make(map[string]string),
	}
}

func (t *ToolkitReporter) SetReport(report Report) error {
	t.scrambleColumns = report.ScrambleColumns
	return nil
}

func (t *ToolkitReporter) WriteTable(data ...[]string) {
	if len(data) == 0 {
		return
	}
	scrambleColumns(t.scrambleColumns, data)
	// set heading first time we see any data
	if len(t.table) == 0 {
		t.table = [][]string{
			data[0],
		}
	}
	t.table = append(t.table, data[1:]...)
}

// WriteHeadline writes a Geneos Toolkit formatted headline to the
// reporter.
func (t *ToolkitReporter) WriteHeadline(name, value string) {
	t.headlines[name] = value
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

func (c *ToolkitReporter) Close() {
	//
}
