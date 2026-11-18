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
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// FormattedReporter implements a Reporter that outputs in various formatted
// text formats, including plain text tables, Markdown and HTML.
type FormattedReporter struct {
	reporterCommon
	w io.Writer
	t table.Writer
	z *zip.Writer

	// render is the function used to render the table in the selected format
	renderFunc func() string

	// set if any data has been rendered, to allow a spacer line to be
	// added for those formats that need it
	rendered bool

	// set once Prepare has been called, so that the first call does not
	// render an empty report
	prepared bool

	headlines     headlineSet
	headlinestyle table.Style

	columns          []string
	rowsBytableOrder map[string][][]string
	tablestyle       table.Style
	htmlpreamble     string
	htmlpostscript   string
	orderbycolumns   []int
	options          []FormattedReporterOption
}

// ensure that *FormattedReporter is a Reporter
var _ Reporter = (*FormattedReporter)(nil)

func init() {
	registerReporter("table", newFormattedReporter)
	registerReporter("html", newFormattedReporter)
	registerReporter("markdown", newFormattedReporter)
	registerReporter("md", newFormattedReporter)
	registerReporter("tsv", newFormattedReporter)
	registerReporter("csv", newFormattedReporter)
}

// newFormattedReporter returns a new FormattedReporter reporter
func newFormattedReporter(format string, w io.Writer, options ...any) (Reporter, error) {
	ropts := evalReporterOptions(CollectOptions[ReporterOption](options...)...)
	foptions := CollectOptions[FormattedReporterOption](options...)
	opts := evalFormattedOptions(foptions...)

	// an explicit Writer option takes precedence
	if opts.writer != nil {
		w = opts.writer
	}

	fr := &FormattedReporter{
		reporterCommon: newReporterCommon(format, ropts),
		w:              w,
		z:              opts.zipWriter,
		t:              table.NewWriter(),
		columns:        []string{},
		options:        foptions,
		orderbycolumns: opts.orderbycolumns,
	}

	fr.updateReporter(foptions...)
	if fr.format == "html" && fr.w != nil {
		if _, err := fr.w.Write([]byte(fr.htmlpreamble)); err != nil {
			return nil, fmt.Errorf("write report HTML preamble: %w", err)
		}
	}
	return fr, nil
}

// Prepare TODO: Document its behavior, inputs, and effects.
func (fr *FormattedReporter) Prepare(report Report) (err error) {
	// flush any existing data, but not before the first report
	if fr.prepared {
		if err = fr.Render(); err != nil {
			return
		}
	}

	// reset
	common := fr.reporterCommon
	common.Report = report
	*fr = FormattedReporter{
		reporterCommon: common,
		w:              fr.w,
		t:              table.NewWriter(),
		z:              fr.z,
		columns:        []string{},
		options:        fr.options,
		rendered:       fr.rendered,
		orderbycolumns: fr.orderbycolumns,
		prepared:       true,
	}

	fr.updateReporter(fr.options...)
	return nil
}

// AddHeadline adds a headline to the reporter. The order of headlines
// is preserved, and the headlines are rendered in the order they were
// added.
func (fr *FormattedReporter) AddHeadline(name, value string) {
	fr.headlines.add(name, value)
}

// AddHeadlines adds multiple headlines to the reporter. As the order
// of a map is undefined they are added in sorted name order, after any
// headlines already added.
func (fr *FormattedReporter) AddHeadlines(headlines map[string]string) {
	fr.headlines.addAll(headlines)
}

// UpdateTable TODO: Document its behavior, inputs, and effects.
func (fr *FormattedReporter) UpdateTable(columns []string, data [][]string) {
	if len(fr.columns) == 0 {
		// init
		fr.columns = columns
	}
	if fr.rowsBytableOrder == nil {
		fr.rowsBytableOrder = make(map[string][][]string, len(data))
	}

	if len(data) == 0 {
		return
	}
	data = fr.scrambleTable(columns, data)

	// order by the first column unless told otherwise
	orderbycolumns := fr.orderbycolumns
	if len(orderbycolumns) == 0 {
		orderbycolumns = []int{0}
	}
	orderColumns := []int{}
	for _, c := range orderbycolumns {
		if c >= 0 && c < len(columns) {
			orderColumns = append(orderColumns, c)
		}
	}

	for _, row := range data {
		var order strings.Builder
		for _, c := range orderColumns {
			if c < len(row) {
				order.WriteString(row[c])
				order.WriteString("\000")
			}
		}
		fr.rowsBytableOrder[order.String()] = append(fr.rowsBytableOrder[order.String()], row)
	}
}

// Reset TODO: Document its behavior, inputs, and effects.
func (fr *FormattedReporter) Reset(report Report) (err error) {
	// do nothing
	return
}

// Render sends the collected report data to the underlying table.Writer
// as on table of headlines and another or table data
func (fr *FormattedReporter) Render() error {
	var err error

	if fr.z != nil || fr.format != "csv" {
		if fr.headlines.len() > 0 {
			// render headers
			fr.t.AppendHeader(table.Row{"headline", "value"})
			headlines := []table.Row{}
			for _, h := range fr.headlines.order {
				headlines = append(headlines, table.Row{h, fr.headlines.values[h]})
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
						return fmt.Errorf("create report headlines archive entry: %w", err)
					}
				} else if fr.rendered {
					fmt.Fprintln(fr.w)
				}

				fr.t.SetOutputMirror(w)
				fr.renderFunc()
				if fr.z != nil {
					if err = fr.z.Flush(); err != nil {
						return fmt.Errorf("flush report headlines archive entry: %w", err)
					}
				}
				fr.rendered = true
			}
		}

		// reset for render table
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
	// sort by order of rows (which can be grouped) and then by order of columns
	for _, rn := range slices.Sorted(maps.Keys(fr.rowsBytableOrder)) {
		for _, r := range fr.rowsBytableOrder[rn] {
			row := table.Row{}
			for cn := range fr.columns {
				// rows may be shorter than the columns
				if cn < len(r) {
					row = append(row, r[cn])
				} else {
					row = append(row, "")
				}
			}
			fr.t.AppendRow(row)
		}
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
				return fmt.Errorf("create report archive entry: %w", err)
			}
		} else if fr.rendered {
			fmt.Fprintln(fr.w)
		}
		fr.t.SetOutputMirror(w)
		fr.renderFunc()
		if fr.z != nil {
			if err = fr.z.Flush(); err != nil {
				return fmt.Errorf("flush report archive entry: %w", err)
			}
		}
		fr.rendered = true
	}
	return nil
}

// Close TODO: Document its behavior, inputs, and effects.
func (fr *FormattedReporter) Close() error {
	if fr.format == "html" && fr.w != nil {
		if _, err := fr.w.Write([]byte(fr.htmlpostscript)); err != nil {
			return fmt.Errorf("write report HTML postscript: %w", err)
		}
	}

	if fr.z != nil {
		// close the ZIP, create the directory entries
		err := fr.z.Close()
		if err != nil {
			return fmt.Errorf("close report archive: %w", err)
		}
	}

	if c, ok := fr.w.(io.Closer); ok {
		err := c.Close()
		if err != nil {
			return fmt.Errorf("close report output: %w", err)
		}
	}
	return nil
}

// Extension TODO: Document its behavior, inputs, and effects.
func (fr *FormattedReporter) Extension() string {
	return fr.format
}

func (fr *FormattedReporter) updateReporter(options ...FormattedReporterOption) {
	// update saved options
	fr.options = options

	opts := evalFormattedOptions(options...)

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
