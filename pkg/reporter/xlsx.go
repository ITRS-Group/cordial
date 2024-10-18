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
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xuri/excelize/v2"

	"github.com/itrs-group/cordial/pkg/config"
)

type XLSXReporter struct {
	ReporterCommon
	x *excelize.File
	w io.Writer

	summarySheet string

	// current prepared currentSheet name
	currentSheet string
	sheets       map[string]*sheet

	// common styles
	topHeading   int
	leftHeading  int
	rightAlign   int
	dateStyle    int
	intStyle     int
	percentStyle int
	plainStyle   int

	// top level scramble toggle
	scambleNames bool
	password     *config.Plaintext

	// conditional format global styles
	cond        map[string]*int
	minColWidth float64
	maxColWidth float64

	headlines int
}

type sheet struct {
	// row name mapping
	rowOrder []string

	// rows of data where the key to the map is the rowname and the string slice is the row data
	rows map[string][]string

	// offset for first row of data table (column names etc.) 0 means excel row 1
	rowOffset int

	// the names of the columns for the dataview data
	columns []string

	// the width to apply to the columns based on min/max and the length of the data in the column
	columnWidths []float64

	scrambleColumns   []string
	conditionalFormat []ConditionalFormat
	freezeColumn      string

	headlineOrder []string
	headlines     map[string]string
}

// ensure that *Table is a Reporter
var _ Reporter = (*XLSXReporter)(nil)

// NewTableReporter returns a new Table reporter
func newXLSXReporter(w io.Writer, ropts *reporterOptions, options ...XLSXReporterOptions) (x *XLSXReporter) {
	opts := evalXLSXReportOptions(options...)

	x = &XLSXReporter{
		x:            excelize.NewFile(),
		w:            w,
		scambleNames: opts.scramble,
		password:     opts.password,
		sheets:       map[string]*sheet{},
		headlines:    opts.headlines,
	}

	x.topHeading, _ = x.x.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"cccccc"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Bold: true,
		},
	})

	x.leftHeading, _ = x.x.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"cccccc"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Bold: true,
		},
	})

	x.rightAlign, _ = x.x.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
	})

	x.dateStyle, _ = x.x.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		CustomNumFmt: &opts.dateFormat,
	})

	x.intStyle, _ = x.x.NewStyle(&excelize.Style{
		NumFmt: opts.intFormat,
	})

	x.percentStyle, _ = x.x.NewStyle(&excelize.Style{
		NumFmt: opts.percentFormat,
	})

	x.plainStyle, _ = x.x.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			WrapText: true,
		},
		NumFmt: 49, // `@` which is `Text`
	})

	// set conditional formats
	ok, _ := x.x.NewConditionalStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{opts.okColour},
			Pattern: 1,
		},
	})
	warning, _ := x.x.NewConditionalStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{opts.warningColour},
			Pattern: 1,
		},
	})
	critical, _ := x.x.NewConditionalStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{opts.criticalColour},
			Pattern: 1,
		},
	})
	undefined, _ := x.x.NewConditionalStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{opts.undefinedColour},
			Pattern: 1,
		},
	})

	x.cond = map[string]*int{
		"ok":        &ok,
		"warning":   &warning,
		"critical":  &critical,
		"undefined": &undefined,
	}

	x.summarySheet = opts.summarySheetName
	x.sheets[opts.summarySheetName] = &sheet{}
	x.x.SetSheetName("Sheet1", opts.summarySheetName)

	x.minColWidth = opts.minColWidth
	x.maxColWidth = opts.maxColWidth

	return
}

type XLSXReporterOptions func(*xlsxReportOptions)

type xlsxReportOptions struct {
	scramble         bool
	password         *config.Plaintext
	summarySheetName string
	dateFormat       string
	intFormat        int
	percentFormat    int
	undefinedColour  string
	okColour         string
	warningColour    string
	criticalColour   string
	minColWidth      float64
	maxColWidth      float64
	headlines        int
}

func evalXLSXReportOptions(options ...XLSXReporterOptions) (xo *xlsxReportOptions) {
	xo = &xlsxReportOptions{
		summarySheetName: "Summary",
		dateFormat:       "yyyy-mm-ddThh:MM:ss",
		intFormat:        1,
		percentFormat:    9,
		undefinedColour:  "BFBFBF",
		okColour:         "5BB25C",
		warningColour:    "F9B057",
		criticalColour:   "FF5668",
		minColWidth:      10.0,
		maxColWidth:      60.0,
	}
	for _, opt := range options {
		opt(xo)
	}
	return
}

func XLSXScramble(scramble bool) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.scramble = scramble
	}
}

// XLSXPassword sets the workbook password
func XLSXPassword(password *config.Plaintext) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.password = password
	}
}

const (
	XLSXHeadlinesNone = iota
	XLSXHeadlinesVertical
	XLSXHeadlinesHorizontal
)

// XLSXHeadlines sets visibility and the direction of headlines in a
// sheet. The default is not to include headlines. If passed
// XLSXHeadlinesVertical then the headlines are added as a two column
// list of name/value pairs. If passed XLSXHeadlinesHorizontal then headlines
// are added as two rows with each headlines added as a single column,
// name above value.
func XLSXHeadlines(headlines int) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.headlines = headlines
	}
}

func SummarySheetName(name string) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.summarySheetName = name
	}
}

func DateFormat(dateFormat string) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.dateFormat = dateFormat
	}
}

func IntFormat(format int) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.intFormat = format
	}
}

func PercentFormat(format int) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.percentFormat = format
	}
}

func SeverityColours(undefined, ok, warning, critical string) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.undefinedColour = undefined
		xro.okColour = ok
		xro.warningColour = warning
		xro.criticalColour = critical
	}
}

func MinColumnWidth(n float64) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.minColWidth = n
	}
}

func MaxColumnWidth(n float64) XLSXReporterOptions {
	return func(xro *xlsxReportOptions) {
		xro.maxColWidth = n
	}
}

func (x *XLSXReporter) Prepare(report Report) (err error) {
	x.currentSheet = report.Title

	if len(x.currentSheet) > 31 {
		log.Error().Msgf("report title '%s' exceeds sheet name limit of 31 chars, truncating", x.currentSheet)
		x.currentSheet = x.currentSheet[:31]
	}
	idx, _ := x.x.GetSheetIndex(x.currentSheet)
	if idx != -1 && x.currentSheet != x.summarySheet {
		log.Error().Msgf("a sheet with the same name already exists, data will clash: '%s'", x.currentSheet)
	}

	x.sheets[x.currentSheet] = &sheet{
		scrambleColumns:   report.ScrambleColumns,
		conditionalFormat: report.XLSX.ConditionalFormat,
		freezeColumn:      report.XLSX.FreezeColumn,
		rows:              map[string][]string{},
		// rowOffset:         3,
	}

	_, err = x.x.NewSheet(x.currentSheet)
	return
}

func (x *XLSXReporter) UpdateTable(columns []string, data [][]string) {
	sheet := x.sheets[x.currentSheet]
	if sheet == nil {
		return
	}

	sheet.columns = columns
	for _, c := range sheet.columns {
		sheet.columnWidths = append(sheet.columnWidths, limitWidth(len(c), x.minColWidth, x.maxColWidth))
	}

	if len(data) == 0 {
		return
	}

	if x.scambleNames {
		scrambleColumns(columns, sheet.scrambleColumns, data)
	}

	for _, cellsString := range data {
		sheet.rows[cellsString[0]] = cellsString
		sheet.rowOrder = append(sheet.rowOrder, cellsString[0])

		for j, c := range cellsString {
			sheet.columnWidths[j] = limitWidth(len(fmt.Sprint(c)), sheet.columnWidths[j], x.maxColWidth)
		}
	}
}

func (x *XLSXReporter) setSheetData() {
	for sheetname, sheet := range x.sheets {
		// render headlines first, updating rowOffset as we go if "vertical"
		switch x.headlines {
		case XLSXHeadlinesVertical:
			sheet.rowOffset = 1
			if len(sheet.columnWidths) < 2 {
				sheet.columnWidths = []float64{0.0, 0.0}
			}
			for i, headline := range sheet.headlineOrder {
				namecell, _ := excelize.CoordinatesToCellName(1, 1+i)
				x.x.SetCellStr(sheetname, namecell, headline)
				x.x.SetCellStyle(sheetname, namecell, namecell, x.leftHeading)

				valuecell, _ := excelize.CoordinatesToCellName(2, i+1)
				value := stringToCell(sheet.headlines[headline])
				x.x.SetCellValue(sheetname, valuecell, value)
				switch value.(type) {
				case time.Time:
					x.x.SetCellStyle(sheetname, valuecell, valuecell, x.dateStyle)
				case int64:
					x.x.SetCellStyle(sheetname, valuecell, valuecell, x.intStyle)
				case float64:
					x.x.SetCellStyle(sheetname, valuecell, valuecell, x.percentStyle)
				default:
					x.x.SetCellStyle(sheetname, valuecell, valuecell, x.plainStyle)
				}
				sheet.columnWidths[0] = limitWidth(len(headline), sheet.columnWidths[0], x.maxColWidth)
				sheet.columnWidths[1] = limitWidth(len(fmt.Sprint(value)), sheet.columnWidths[1], x.maxColWidth)
				sheet.rowOffset++
			}
		case XLSXHeadlinesHorizontal:
			sheet.rowOffset = 3
			for i, headline := range sheet.headlineOrder {
				namecell, _ := excelize.CoordinatesToCellName(i+1, 1)
				x.x.SetCellStr(sheetname, namecell, headline)
				x.x.SetCellStyle(sheetname, namecell, namecell, x.leftHeading)

				valuecell, _ := excelize.CoordinatesToCellName(i+1, 2)
				value := stringToCell(sheet.headlines[headline])
				x.x.SetCellValue(sheetname, valuecell, value)
				switch value.(type) {
				case time.Time:
					x.x.SetCellStyle(sheetname, valuecell, valuecell, x.dateStyle)
				case int64:
					x.x.SetCellStyle(sheetname, valuecell, valuecell, x.intStyle)
				case float64:
					x.x.SetCellStyle(sheetname, valuecell, valuecell, x.percentStyle)
				default:
					x.x.SetCellStyle(sheetname, valuecell, valuecell, x.plainStyle)
				}
				// x.x.SetCellStr(sheetname, valuecell, sheet.headlines[headline])

				if len(sheet.columnWidths) == i {
					sheet.columnWidths = append(sheet.columnWidths, limitWidth(max(len(headline), len(fmt.Sprint(value))), x.minColWidth, x.maxColWidth))
				} else {
					sheet.columnWidths[i] = limitWidth(max(len(headline), len(fmt.Sprint(value))), sheet.columnWidths[i], x.maxColWidth)
				}

			}
		default:
			// nothing
		}

		// render main table
		columnsCell, err := excelize.CoordinatesToCellName(1, 1+sheet.rowOffset, false)
		if err != nil {
			panic(err)
		}
		if err = x.x.SetSheetRow(sheetname, columnsCell, &sheet.columns); err != nil {
			return
		}

		for rownum, rowname := range sheet.rowOrder {
			cellsString := sheet.rows[rowname]
			cells := stringsToCells(cellsString)
			if err = x.x.SetSheetRow(sheetname, fmt.Sprintf("A%d", 2+rownum+sheet.rowOffset), &cells); err != nil {
				return
			}

			// update styles
			for i, cell := range cells {
				cellname, _ := excelize.CoordinatesToCellName(i+1, 2+rownum+sheet.rowOffset)
				switch cell.(type) {
				case time.Time:
					x.x.SetCellStyle(sheetname, cellname, cellname, x.dateStyle)
				case int64:
					x.x.SetCellStyle(sheetname, cellname, cellname, x.intStyle)
				case float64:
					x.x.SetCellStyle(sheetname, cellname, cellname, x.percentStyle)
				default:
					x.x.SetCellStyle(sheetname, cellname, cellname, x.plainStyle)
				}
			}
		}
		x.x.SetRowStyle(sheetname, 1+sheet.rowOffset, 1+sheet.rowOffset, x.topHeading)
	}
}

func (x *XLSXReporter) setColumnWidths() {
	for sheetname, sheet := range x.sheets {
		// set column widths
		for i, c := range sheet.columnWidths {
			col, err := excelize.ColumnNumberToName(i + 1)
			if err != nil {
				panic(err)
			}
			if err = x.x.SetColWidth(sheetname, col, col, c); err != nil {
				return
			}
		}
	}
}

func (x *XLSXReporter) freezePanes() {
	for sheetname, sheet := range x.sheets {
		var err error
		if sheet.freezeColumn == "" {
			// just row headings / headlines
			cellname, _ := excelize.CoordinatesToCellName(1, 2+sheet.rowOffset, false)

			if err = x.x.SetPanes(sheetname, &excelize.Panes{
				Freeze:      true,
				YSplit:      1 + sheet.rowOffset,
				TopLeftCell: cellname,
				ActivePane:  "bottomLeft",
				Selection: []excelize.Selection{
					{SQRef: cellname, ActiveCell: cellname, Pane: "bottomLeft"},
				},
			}); err != nil {
				log.Error().Err(err).Msg("freeze top row")
			}
		} else {
			i := slices.Index(sheet.columns, sheet.freezeColumn)
			if i == -1 {
				log.Warn().Msgf("unknown column name %q, skipping freeze left pane for sheet %s", sheet.freezeColumn, sheetname)
				return
			}
			// cellname is the first unlocked cell (so +2)
			cellname, _ := excelize.CoordinatesToCellName(i+2, 2+sheet.rowOffset, false)
			if err = x.x.SetPanes(sheetname, &excelize.Panes{
				Freeze:      true,
				XSplit:      i + 1,
				YSplit:      1 + sheet.rowOffset,
				TopLeftCell: cellname,
				ActivePane:  "bottomRight",
				Selection: []excelize.Selection{
					{SQRef: cellname, ActiveCell: cellname, Pane: "bottomRight"},
				},
			}); err != nil {
				log.Error().Err(err).Msg("freeze table")
			}
		}
	}
}

func logicalWrapper(logic string) string {
	switch strings.ToLower(logic) {
	case "or", "any":
		return "OR"
	default:
		return "AND"
	}
}

func (x *XLSXReporter) Remove(report Report) (err error) {
	// do nothing, yet
	return
}

func (x *XLSXReporter) AddHeadline(name, value string) {
	sheet := x.sheets[x.currentSheet]

	if len(sheet.headlineOrder) == 0 {
		sheet.headlines = make(map[string]string)
	}

	sheet.headlineOrder = append(sheet.headlineOrder, name)
	sheet.headlines[name] = value
}

func (x *XLSXReporter) Render() {
	x.setSheetData()
	x.applyConditionalFormat()
	x.setColumnWidths()
	x.freezePanes()
	x.x.Write(x.w, excelize.Options{
		Password: x.password.String(),
	})
}

func (x *XLSXReporter) Close() {
	x.x.Close()
}

var percentRE = regexp.MustCompile(`^\d+\s*%$`)
var numRE = regexp.MustCompile(`^\d+$`)
var validcond = []string{
	"=",
	">",
	"<",
	">=",
	"<=",
	"<>",
}

func stringToCell(s string) (cell any) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	} else if t, err := time.Parse(time.Layout, s); err == nil {
		return t
	} else if percentRE.MatchString(s) {
		var f float64
		fmt.Sscan(s, &f)
		return f / 100.0
	} else if numRE.MatchString(s) {
		var n int64
		fmt.Sscan(s, &n)
		return n
	} else {
		return s
	}
}

func stringsToCells(rowStrings []string) (row []any) {
	for _, cell := range rowStrings {
		row = append(row, stringToCell(cell))
	}
	return
}

// apply conditional formatting to all sheets as part of the render
// process
func (x *XLSXReporter) applyConditionalFormat() {
	for sheetname, sheet := range x.sheets {
		// conditional formats apply to columns of table data, so create
		// the format from the config then apply to range

		for _, c := range sheet.conditionalFormat {
			// validate conditions allowed
			if !slices.Contains(validcond, c.Test.Condition) {
				log.Error().Msgf("sheet %s: invalid condition %s, skipping test", sheetname, c.Test.Condition)
				continue
			}

			// match defaults to true unless all "Rows/NotRows" fail. if
			// no tests, then succeed regardless
			match := true
			format := "undefined"

			// columns this conditional format will apply to
			cols := []string{}

			for _, s := range c.Set {
				if s.NotRows != "" {
					match = false
					if ok, _ := path.Match(s.NotRows, sheet.columns[0]); !ok {
						format = s.Format
						cols = s.Columns
						match = true
						break
					}
				} else if s.Rows != "" {
					match = false
					if ok, _ := path.Match(s.Rows, sheet.columns[0]); ok {
						format = s.Format
						cols = s.Columns
						match = true
						break
					}
				} else {
					format = s.Format
					cols = s.Columns
				}
			}

			if !match {
				continue
			}

			rows := len(sheet.rowOrder)

			tc := []string{}

			// if no set columns set, then use test columns
			if len(cols) == 0 {
				cols = c.Test.Columns
			}

			// build formulas
			for _, t := range c.Test.Columns {
				i := slices.Index(sheet.columns, t)
				if i == -1 {
					log.Warn().Msgf("unknown column name %q, skipping conditional formatting for sheet %s", t, sheetname)
					return
				}
				firstcell, err := excelize.CoordinatesToCellName(i+1, 2+sheet.rowOffset, false)
				if err != nil {
					panic(err)
				}
				// make the column *only* absolute
				firstcell = "$" + firstcell

				switch c.Test.Type {
				case "number":
					// convert black cells to zero and then text with
					tc = append(tc, fmt.Sprintf("IF(OR(ISBLANK(%[1]s),%[1]s=\"\"),0,%[1]s)%s%s", firstcell, c.Test.Condition, c.Test.Value))
				default:
					tc = append(tc, fmt.Sprintf("%s%s%q", firstcell, c.Test.Condition, c.Test.Value))
				}
			}

			logic := logicalWrapper(c.Test.Logical)
			formula := logic + "(" + strings.Join(tc, ",") + ")"

			refs := []string{}
			for _, col := range cols {
				i := slices.Index(sheet.columns, col)
				if i == -1 {
					log.Warn().Msgf("unknown column name %q, skipping conditional formatting for sheet %s", col, sheetname)
					return
				}
				firstcell, err := excelize.CoordinatesToCellName(i+1, 2+sheet.rowOffset, false)
				if err != nil {
					panic(err)
				}
				lastcell, err := excelize.CoordinatesToCellName(i+1, 1+sheet.rowOffset+rows, false)
				if err != nil {
					panic(err)
				}
				refs = append(refs, firstcell+":"+lastcell)
			}

			x.setConditionalFormat(sheetname, formula, format, refs...)
		}
	}
}

func (x *XLSXReporter) setConditionalFormat(sheetname, formula, format string, refs ...string) {
	if err := x.x.SetConditionalFormat(sheetname, strings.Join(refs, ","), []excelize.ConditionalFormatOptions{
		{
			Type:     "formula",
			Criteria: formula,
			Format:   x.cond[format],
		},
	}); err != nil {
		log.Fatal().Err(err).Msgf("formula %s on %s", formula, strings.Join(refs, ","))
	}
}

// a scale factor for the column width versus string len
const colScale = 1.25

func limitWidth(chars int, minWidth, maxWidth float64) float64 {
	return max(min(colScale*float64(chars), maxWidth), minWidth)
}
