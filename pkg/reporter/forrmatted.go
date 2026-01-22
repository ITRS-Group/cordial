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
	"archive/zip"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// FormattedReporter implements a Reporter that outputs in various formatted
// text formats, including plain text tables, Markdown and HTML.
type FormattedReporter struct {
	reporterCommon
	w      io.Writer
	t      table.Writer
	z      *zip.Writer
	format string

	// render is the function used to render the table in the selected format
	renderFunc func() string

	// set if any data has been rendered, to allow a spacer line to be
	// added for those formats that need it
	rendered bool

	headlineOrder []string
	headlines     map[string]string
	headlinestyle table.Style

	columns         []string
	tableOrder      []string
	table           map[string][]string
	tablestyle      table.Style
	htmlpreamble    string
	htmlpostscript  string
	orderbycolumns  []int
	options         []FormattedReporterOptions
	scrambleColumns []string
}

// ensure that *FormattedReporter is a Reporter
var _ Reporter = (*FormattedReporter)(nil)

// newFormattedReporter returns a new FormattedReporter reporter
func newFormattedReporter(ropts *reporterOptions, options ...FormattedReporterOptions) (t *FormattedReporter) {
	opts := evalFormattedOptions(options...)
	t = &FormattedReporter{
		reporterCommon: reporterCommon{
			scrambleNames: ropts.scrambleNames,
		},
		w:              opts.writer,
		z:              opts.zipWriter,
		t:              table.NewWriter(),
		columns:        []string{},
		options:        options,
		orderbycolumns: opts.orderbycolumns,
	}

	t.updateReporter(options...)
	if t.format == "html" {
		t.w.Write([]byte(t.htmlpreamble))
	}
	return
}

func (fr *FormattedReporter) Prepare(report Report) (err error) {
	// flush any existing data
	fr.Render()

	// reset
	*fr = FormattedReporter{
		reporterCommon: reporterCommon{
			Report:        report,
			format:        fr.format,
			scrambleNames: fr.scrambleNames,
		},
		// title:           report.Title,
		w:               fr.w,
		t:               table.NewWriter(),
		z:               fr.z,
		columns:         []string{},
		options:         fr.options,
		scrambleColumns: report.ScrambleColumns,
		rendered:        fr.rendered,
		orderbycolumns:  fr.orderbycolumns,
	}

	fr.updateReporter(fr.options...)
	return nil
}

func (fr *FormattedReporter) AddHeadline(name, value string) {
	if len(fr.headlineOrder) == 0 {
		// init map
		fr.headlines = map[string]string{}
	}
	fr.headlineOrder = append(fr.headlineOrder, name)
	fr.headlines[name] = value
}

func (fr *FormattedReporter) UpdateTable(columns []string, data [][]string) {
	if len(fr.columns) == 0 {
		// init
		fr.columns = columns
		fr.table = map[string][]string{}
	}

	if len(data) == 0 {
		return
	}
	if fr.scrambleNames {
		scrambleColumns(columns, fr.scrambleColumns, data)
	}
	for _, row := range data {
		order := ""
		for _, c := range fr.orderbycolumns {
			if c < len(row) {
				order += row[c] + "\000"
			}
		}
		fr.tableOrder = append(fr.tableOrder, order)
		fr.table[order] = row
	}
}

func (fr *FormattedReporter) Remove(report Report) (err error) {
	// do nothing
	return
}

// Render sends the collected report data to the underlying table.Writer
// as on table of headlines and another or table data
func (fr *FormattedReporter) Render() {
	var err error

	if fr.z != nil || fr.format != "csv" {
		if len(fr.headlines) > 0 {
			// render headers
			fr.t.AppendHeader(table.Row{"headline", "value"})
			headlines := []table.Row{}
			for _, h := range fr.headlineOrder {
				headlines = append(headlines, table.Row{h, fr.headlines[h]})
			}
			fr.t.SetColumnConfigs([]table.ColumnConfig{
				{Number: 2, WidthMax: 0},
			})
			fr.t.Style().Format.Header = text.FormatDefault
			if fr.format != "csv" {
				fr.t.SetTitle(fr.Title + " Headlines")
			}
			fr.t.AppendRows(headlines)
			fr.t.SetStyle(fr.headlinestyle)
			if fr.t.Length() > 0 {
				w := fr.w
				if fr.z != nil {
					// need to create a new writer for headlines
					p := fr.FilePath
					pe := path.Ext(p)
					pf := strings.TrimSuffix(p, pe)
					fp := pf + ".headlines" + pe
					w, err = fr.z.CreateHeader(&zip.FileHeader{
						Name:     fp,
						Method:   zip.Deflate,
						Modified: time.Now(),
					})
					if err != nil {
						panic(err)
					}
				} else if fr.rendered {
					fmt.Fprintln(fr.w)
				}

				fr.t.SetOutputMirror(w)
				fr.renderFunc()
				if fr.z != nil {
					fr.z.Flush()
				}
				fr.rendered = true
			}
		}

		// render table
		fr.t.ResetHeaders()
		fr.t.ResetRows()
		if fr.format != "csv" {
			fr.t.SetTitle(fr.Title)
		}
	}

	fr.t.SetAllowedRowLength(0)
	fr.t.SetColumnConfigs([]table.ColumnConfig{
		{Number: len(fr.columns), WidthMax: 0},
	})
	fr.t.Style().Format.Header = text.FormatDefault

	headings := table.Row{}
	for _, h := range fr.columns {
		headings = append(headings, h)
	}
	fr.t.AppendHeader(headings)
	for _, rn := range fr.tableOrder {
		row := table.Row{}
		for cn := range fr.columns {
			row = append(row, fr.table[rn][cn])
		}
		fr.t.AppendRow(row)
	}
	fr.t.SetStyle(fr.tablestyle)
	if fr.t.Length() > 0 || fr.z != nil {
		w := fr.w
		if fr.z != nil {
			// need to create a new writer for table
			w, err = fr.z.CreateHeader(&zip.FileHeader{
				Name:     fr.FilePath,
				Method:   zip.Deflate,
				Modified: time.Now(),
			})
			if err != nil {
				panic(err)
			}
		} else if fr.rendered {
			fmt.Fprintln(fr.w)
		}
		fr.t.SetOutputMirror(w)
		fr.renderFunc()
		if fr.z != nil {
			fr.z.Flush()
		}
		fr.rendered = true
	}
}

func (fr *FormattedReporter) Close() {
	if fr.format == "html" {
		fr.w.Write([]byte(fr.htmlpostscript))
	}

	if fr.z != nil {
		// close the ZIP, create the directory entries
		err := fr.z.Close()
		if err != nil {
			panic(err)
		}
	}

	if c, ok := fr.w.(io.Closer); ok {
		err := c.Close()
		if err != nil {
			panic(err)
		}
	}
}

func (fr *FormattedReporter) Extension() string {
	return fr.format
}

func (fr *FormattedReporter) updateReporter(options ...FormattedReporterOptions) {
	// update saved options
	fr.options = options

	opts := evalFormattedOptions(options...)
	// if opts.writer != nil {
	// 	fr.writer = opts.writer
	// 	fr.tableWriter.SetOutputMirror(opts.writer)
	// }
	fr.format = opts.renderas

	switch fr.format {
	case "html":
		fr.tablestyle.HTML = table.HTMLOptions{
			CSSClass:    opts.dvcssclass,
			EmptyColumn: "&nbsp;",
			EscapeText:  true,
			Newline:     "<br/>",
		}
		fr.headlinestyle.HTML = table.HTMLOptions{
			CSSClass:    opts.headlinecssclass,
			EmptyColumn: "&nbsp;",
			EscapeText:  true,
			Newline:     "<br/>",
		}
		fr.renderFunc = fr.t.RenderHTML
		fr.htmlpreamble = opts.htmlpreamble
		fr.htmlpostscript = opts.htmlpostscript
	case "csv":
		fr.renderFunc = fr.t.RenderCSV
	case "markdown", "md":
		fr.renderFunc = fr.t.RenderMarkdown
	case "tsv":
		fr.headlinestyle = table.StyleLight
		fr.tablestyle = table.StyleLight
		fr.renderFunc = fr.t.RenderTSV
	case "table":
		fallthrough
	default:
		s := table.StyleLight
		s.Format.Header = text.FormatDefault
		fr.headlinestyle = s
		fr.tablestyle = s

		fr.renderFunc = fr.t.Render
	}
}
