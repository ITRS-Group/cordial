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
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type FormattedReporter struct {
	ReporterCommon
	name            string
	w               io.Writer
	t               table.Writer
	renderas        string
	render          func() string
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
	scrambleNames   bool
	scrambleColumns []string
}

// ensure that *Table is a Reporter
var _ Reporter = (*FormattedReporter)(nil)

// newFormattedReporter returns a new Table reporter
func newFormattedReporter(w io.Writer, ropts *reporterOptions, options ...FormattedReporterOptions) (tr *FormattedReporter) {
	tr = &FormattedReporter{
		w:       w,
		t:       table.NewWriter(),
		columns: []string{},
		options: options,
	}
	tr.t.SetOutputMirror(tr.w)

	tr.updateReporter(options...)
	if tr.renderas == "html" {
		tr.w.Write([]byte(tr.htmlpreamble))
	}
	return
}

func (t *FormattedReporter) Prepare(report Report) error {
	title := report.Title

	// write the last output
	t.Render()

	// reset
	*t = FormattedReporter{
		w:               t.w,
		name:            title,
		t:               table.NewWriter(),
		columns:         []string{},
		options:         t.options,
		scrambleColumns: report.ScrambleColumns,
	}
	t.t.SetOutputMirror(t.w)

	t.updateReporter(t.options...)
	return nil
}

func (tr *FormattedReporter) updateReporter(options ...FormattedReporterOptions) {
	tr.options = options
	opts := evalFormattedOptions(options...)
	if opts.writer != nil {
		tr.w = opts.writer
		tr.t.SetOutputMirror(opts.writer)
	}
	tr.scrambleNames = opts.scramble
	tr.renderas = opts.renderas

	switch tr.renderas {
	case "html":
		tr.tablestyle.HTML = table.HTMLOptions{
			CSSClass:    opts.dvcssclass,
			EmptyColumn: "&nbsp;",
			EscapeText:  true,
			Newline:     "<br/>",
		}
		tr.headlinestyle.HTML = table.HTMLOptions{
			CSSClass:    opts.headlinecssclass,
			EmptyColumn: "&nbsp;",
			EscapeText:  true,
			Newline:     "<br/>",
		}
		tr.render = tr.t.RenderHTML
		tr.htmlpreamble = opts.htmlpreamble
		tr.htmlpostscript = opts.htmlpostscript
	case "toolkit", "csv":
		tr.render = tr.t.RenderCSV
	case "markdown", "md":
		tr.render = tr.t.RenderMarkdown
	case "tsv":
		tr.headlinestyle = table.StyleLight
		tr.tablestyle = table.StyleLight
		tr.render = tr.t.RenderTSV
	case "table":
		fallthrough
	default:
		s := table.StyleLight
		s.Format.Header = text.FormatDefault
		tr.headlinestyle = s
		tr.tablestyle = s

		tr.render = tr.t.Render
	}

}

type formattedReporterOptions struct {
	writer           io.Writer
	renderas         string
	dvcssclass       string
	headlinecssclass string
	htmlpreamble     string
	htmlpostscript   string
	scramble         bool
}

func evalFormattedOptions(options ...FormattedReporterOptions) (fro *formattedReporterOptions) {
	fro = &formattedReporterOptions{
		renderas:         "table",
		dvcssclass:       "table",
		headlinecssclass: "headlines",
	}
	for _, opt := range options {
		opt(fro)
	}
	return
}

type FormattedReporterOptions func(*formattedReporterOptions)

func Writer(w io.Writer) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.writer = w
	}
}

func RenderAs(renderas string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.renderas = renderas
	}
}

func DataviewCSSClass(cssclass string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.dvcssclass = cssclass
	}
}

func HeadlineCSSClass(cssclass string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.headlinecssclass = cssclass
	}
}

func HTMLPreamble(preamble string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.htmlpreamble = preamble
	}
}

func HTMLPostscript(postscript string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.htmlpostscript = postscript
	}
}

func Scramble(scramble bool) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.scramble = scramble
	}
}

func (t *FormattedReporter) UpdateTable(columns []string, data [][]string) {
	if len(t.columns) == 0 {
		// init
		t.columns = columns
		t.table = map[string][]string{}
	}

	if len(data) == 0 {
		return
	}
	if t.scrambleNames {
		scrambleColumns(columns, t.scrambleColumns, data)
	}
	for _, row := range data {
		t.tableOrder = append(t.tableOrder, row[0])
		t.table[row[0]] = row
	}
}

func (t *FormattedReporter) Remove(report Report) (err error) {
	// do nothing
	return
}

func (t *FormattedReporter) AddHeadline(name, value string) {
	if len(t.headlineOrder) == 0 {
		// init map
		t.headlines = map[string]string{}
	}
	t.headlineOrder = append(t.headlineOrder, name)
	t.headlines[name] = value
}

// Render sends the collected report data to the underlying table.Writer
// as on table of headlines and another or table data
func (t *FormattedReporter) Render() {
	if len(t.headlines) > 0 {
		// render headers
		headlines := []table.Row{}
		for _, h := range t.headlineOrder {
			headlines = append(headlines, table.Row{h, t.headlines[h]})
		}
		t.t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 2, WidthMax: 0},
		})
		t.t.Style().Format.Header = text.FormatDefault
		t.t.SetTitle(t.name + " Headlines")
		t.t.AppendRows(headlines)
		t.t.SetStyle(t.headlinestyle)
		t.render()
		fmt.Fprintln(t.w)
	}

	// render dataview
	t.t.ResetHeaders()
	t.t.ResetRows()
	t.t.SetTitle(t.name)
	t.t.SetAllowedRowLength(0)
	t.t.SetColumnConfigs([]table.ColumnConfig{
		{Number: len(t.columns), WidthMax: 0},
	})
	t.t.Style().Format.Header = text.FormatDefault

	headings := table.Row{}
	for _, h := range t.columns {
		headings = append(headings, h)
	}
	t.t.AppendHeader(headings)
	for _, rn := range t.tableOrder {
		row := table.Row{}
		for cn := range t.columns {
			row = append(row, t.table[rn][cn])
		}
		t.t.AppendRow(row)
	}
	t.t.SetStyle(t.tablestyle)
	t.render()
	fmt.Fprintln(t.w)
}

func (t *FormattedReporter) Close() {
	if t.renderas == "html" {
		t.w.Write([]byte(t.htmlpostscript))
	}
	if c, ok := t.w.(io.Closer); ok {
		c.Close()
	}
}
