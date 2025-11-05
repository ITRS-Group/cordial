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
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// FormattedReporter implements a Reporter that outputs in various formatted
// text formats, including plain text tables, Markdown and HTML.
type FormattedReporter struct {
	ReporterCommon
	title       string
	name        string
	writer      io.Writer
	tableWriter table.Writer
	zipWriter   *zip.Writer
	format      string
	render      func() string

	// set if any data has been rendered, to allow a spacer line to be
	// added for those formats that make sense
	rendered bool

	headlineOrder   []string
	headlines       map[string]string
	headlinestyle   table.Style
	columns         []string
	tableOrder      []string
	table           map[string][]string
	tablestyle      table.Style
	htmlpreamble    string
	htmlpostscript  string
	options         []FormattedReporterOptions
	scrambleColumns []string
}

// ensure that *FormattedReporter is a Reporter
var _ Reporter = (*FormattedReporter)(nil)

// newFormattedReporter returns a new FormattedReporter reporter
func newFormattedReporter(ropts *reporterOptions, options ...FormattedReporterOptions) (t *FormattedReporter) {
	opts := evalFormattedOptions(options...)
	t = &FormattedReporter{
		ReporterCommon: ReporterCommon{scrambleNames: ropts.scrambleNames},
		writer:         opts.writer,
		zipWriter:      opts.zipWriter,
		tableWriter:    table.NewWriter(),
		columns:        []string{},
		options:        options,
	}
	// t.tableWriter.SetOutputMirror(t.writer)

	t.updateReporter(options...)
	if t.format == "html" {
		t.writer.Write([]byte(t.htmlpreamble))
	}
	return
}

func (fr *FormattedReporter) Prepare(report Report) (err error) {
	// flush any existing data
	fr.Render()

	// reset
	*fr = FormattedReporter{
		ReporterCommon:  ReporterCommon{scrambleNames: fr.scrambleNames},
		title:           report.Title,
		name:            report.Name,
		writer:          fr.writer,
		tableWriter:     table.NewWriter(),
		zipWriter:       fr.zipWriter,
		columns:         []string{},
		options:         fr.options,
		scrambleColumns: report.ScrambleColumns,
		rendered:        fr.rendered,
		format:          fr.format,
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
		fr.tableOrder = append(fr.tableOrder, row[0])
		fr.table[row[0]] = row
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

	if fr.zipWriter != nil || fr.format != "csv" {
		if len(fr.headlines) > 0 {
			// render headers
			headlines := []table.Row{}
			for _, h := range fr.headlineOrder {
				headlines = append(headlines, table.Row{h, fr.headlines[h]})
			}
			fr.tableWriter.SetColumnConfigs([]table.ColumnConfig{
				{Number: 2, WidthMax: 0},
			})
			fr.tableWriter.Style().Format.Header = text.FormatDefault
			fr.tableWriter.SetTitle(fr.title + " Headlines")
			fr.tableWriter.AppendRows(headlines)
			fr.tableWriter.SetStyle(fr.headlinestyle)
			if fr.tableWriter.Length() > 0 {
				w := fr.writer
				if fr.zipWriter != nil {
					// need to create a new writer for headlines
					w, err = fr.zipWriter.CreateHeader(&zip.FileHeader{
						Name:     fr.name + ".headlines." + fr.format,
						Method:   zip.Deflate,
						Modified: time.Now(),
					})
					if err != nil {
						panic(err)
					}
				} else if fr.rendered {
					fmt.Fprintln(fr.writer)
				}

				fr.tableWriter.SetOutputMirror(w)
				fr.render()
				if fr.zipWriter != nil {
					fr.zipWriter.Flush()
				}
				fr.rendered = true
			}
		}

		// render table
		fr.tableWriter.ResetHeaders()
		fr.tableWriter.ResetRows()
		fr.tableWriter.SetTitle(fr.title)
	}

	fr.tableWriter.SetAllowedRowLength(0)
	fr.tableWriter.SetColumnConfigs([]table.ColumnConfig{
		{Number: len(fr.columns), WidthMax: 0},
	})
	fr.tableWriter.Style().Format.Header = text.FormatDefault

	headings := table.Row{}
	for _, h := range fr.columns {
		headings = append(headings, h)
	}
	fr.tableWriter.AppendHeader(headings)
	for _, rn := range fr.tableOrder {
		row := table.Row{}
		for cn := range fr.columns {
			row = append(row, fr.table[rn][cn])
		}
		fr.tableWriter.AppendRow(row)
	}
	fr.tableWriter.SetStyle(fr.tablestyle)
	if fr.tableWriter.Length() > 0 {
		w := fr.writer
		if fr.zipWriter != nil {
			// need to create a new writer for table
			w, err = fr.zipWriter.CreateHeader(&zip.FileHeader{
				Name:     fr.name + "." + fr.format,
				Method:   zip.Deflate,
				Modified: time.Now(),
			})
			if err != nil {
				panic(err)
			}
		} else if fr.rendered {
			fmt.Fprintln(fr.writer)
		}
		fr.tableWriter.SetOutputMirror(w)
		fr.render()
		if fr.zipWriter != nil {
			fr.zipWriter.Flush()
		}
		fr.rendered = true
	}
}

func (fr *FormattedReporter) Close() {
	if fr.format == "html" {
		fr.writer.Write([]byte(fr.htmlpostscript))
	}

	if fr.zipWriter != nil {
		// close the ZIP, create the directory entries
		err := fr.zipWriter.Close()
		if err != nil {
			panic(err)
		}
	}

	if c, ok := fr.writer.(io.Closer); ok {
		err := c.Close()
		if err != nil {
			panic(err)
		}
	}
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
		fr.render = fr.tableWriter.RenderHTML
		fr.htmlpreamble = opts.htmlpreamble
		fr.htmlpostscript = opts.htmlpostscript
	case "csv":
		fr.render = fr.tableWriter.RenderCSV
	case "markdown", "md":
		fr.render = fr.tableWriter.RenderMarkdown
	case "tsv":
		fr.headlinestyle = table.StyleLight
		fr.tablestyle = table.StyleLight
		fr.render = fr.tableWriter.RenderTSV
	case "table":
		fallthrough
	default:
		s := table.StyleLight
		s.Format.Header = text.FormatDefault
		fr.headlinestyle = s
		fr.tablestyle = s

		fr.render = fr.tableWriter.Render
	}
}
