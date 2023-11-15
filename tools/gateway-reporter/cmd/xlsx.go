/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
)

// a scale factor for the column width versus string len
const colScale = 1.25

// minimum column width
const minColWidth = 10.0

// style index for top rows
var topHeading, leftHeading, rightAlign, dateStyle, dataColumnStyle int

func outputXLSX(cf *config.Config, gateway string, entities []Entity, probes map[string]geneos.Probe) (err error) {
	dir := cf.GetString("output.directory")
	_ = os.MkdirAll(dir, 0775) // ignore errors for now

	conftable := config.LookupTable(map[string]string{
		"gateway":  gateway,
		"datetime": startTimestamp,
	})

	xlsx := excelize.NewFile()
	defer xlsx.Close()

	topHeading, _ = xlsx.NewStyle(&excelize.Style{
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

	leftHeading, _ = xlsx.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Font: &excelize.Font{
			Bold: true,
		},
	})

	rightAlign, _ = xlsx.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
	})

	iso := "yyyy-mm-ddThh:MM:ss"
	dateStyle, _ = xlsx.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		CustomNumFmt: &iso,
	})

	dataColumnStyle, _ = xlsx.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			WrapText: true,
		},
	})

	summary := cf.GetString("output.reports.summary.sheetname", conftable)
	xlsx.SetSheetName("Sheet1", summary)
	xlsx.SetColStyle(summary, "A", leftHeading)
	xlsx.SetColStyle(summary, "B", rightAlign)
	site := cf.GetString("site", config.Default("ITRS"))
	xlsx.SetColWidth(summary, "A", "B", colWidth(len(site), 40))
	xlsx.SetSheetCol(summary, "A1", &[]interface{}{
		"ITRS Gateway Reporter",
		"",
		"Site",
		"Report Date/Time",
		"Hostname",
		"",
		"Gateway Name",
		"Probes",
		"Managed Entities",
	})

	hostname, _ := os.Hostname()
	xlsx.SetSheetCol(summary, "B1", &[]interface{}{
		"Version: " + cordial.VERSION,
		"",
		site,
		startTime,
		hostname,
		"",
		gateway,
		len(probes),
		len(entities),
	})

	xlsx.SetCellStyle(summary, "B4", "B4", dateStyle)

	// entities
	if err = outputXLSXEntities(xlsx, entities, cf, conftable); err != nil {
		return
	}

	// files
	for _, f := range cf.GetStringSlice("output.plugins.single-column") {
		if err = outputXLSXSingleColumn(xlsx, entities, cf, conftable, f); err != nil {
			return
		}
	}

	for _, f := range cf.GetStringSlice("output.plugins.two-column") {
		if err = outputXLSXTwoColumn(xlsx, entities, cf, conftable, f); err != nil {
			return
		}
	}

	filename := cf.GetString("output.formats.xlsx", conftable)
	if !filepath.IsAbs(filename) {
		filename = path.Join(dir, filename)
	}
	return xlsx.SaveAs(filename)
}

func colWidth(chars int, min float64) float64 {
	w := colScale * float64(chars)
	if w > 255 {
		return 255
	}
	if w < min {
		return min
	}
	return w
}

func outputXLSXEntities(x *excelize.File, Entities []Entity, cf *config.Config, conftable config.ExpandOptions) (err error) {
	sheet := cf.GetString("output.reports.entities.sheetname")
	if _, err = x.NewSheet(sheet); err != nil {
		return
	}
	cols, attrs, plugins, err := outputEntityColumns(Entities, cf, conftable)
	if err = x.SetSheetRow(sheet, "A1", &cols); err != nil {
		return
	}

	colwidths := []float64{}
	for _, c := range cols {
		colwidths = append(colwidths, colWidth(len(c), minColWidth))
	}

	i := 2
	for _, e := range Entities {
		port := "7036"
		hostname := e.Probe.Hostname
		if hostname == "" {
			hostname = "[Virtual Probe]"
			port = ""
		} else if e.Probe.Port != 0 {
			port = fmt.Sprint(e.Probe.Port)
		}
		row := []interface{}{
			e.Name,
			e.Probe.Name,
			hostname,
			port,
		}

		for _, a := range attrs {
			if attr, ok := e.Attributes[a]; ok {
				row = append(row, attr)
			} else {
				row = append(row, "")
			}
		}
		ptotals := make(map[string]int)
		for _, s := range e.Samplers {
			if s.Plugin == "" {
				ptotals["unsupported"]++
			} else {
				ptotals[s.Plugin]++
			}
		}
		for _, p := range plugins {
			if ptotals[p] > 0 {
				row = append(row, ptotals[p])
			} else {
				row = append(row, nil)
			}
		}
		if err = x.SetSheetRow(sheet, fmt.Sprintf("A%d", i), &row); err != nil {
			return
		}

		for j, c := range cols {
			colwidths[j] = colWidth(len(fmt.Sprint(c)), colwidths[j])
		}

		i++
	}

	// set column widths
	for i, c := range colwidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		x.SetColWidth(sheet, col, col, c)
	}

	// add header and merge cols
	if err = x.InsertRows(sheet, 1, 1); err != nil {
		return
	}

	mecols := len(cols) - len(attrs) - len(plugins)
	cell, _ := excelize.CoordinatesToCellName(1, 1)
	x.SetCellStr(sheet, cell, "Managed Entity")
	end, _ := excelize.CoordinatesToCellName(mecols, 1)
	x.MergeCell(sheet, cell, end)
	// x.SetCellStyle(sheet, cell, end, topHeading)

	cell, _ = excelize.CoordinatesToCellName(mecols+1, 1)
	x.SetCellStr(sheet, cell, "Attributes")
	end, _ = excelize.CoordinatesToCellName(mecols+len(attrs), 1)
	x.MergeCell(sheet, cell, end)
	// x.SetCellStyle(sheet, cell, end, topHeading)

	cell, err = excelize.CoordinatesToCellName(mecols+len(attrs)+1, 1)
	x.SetCellStr(sheet, cell, "Sampler Totals")
	end, _ = excelize.CoordinatesToCellName(mecols+len(attrs)+len(plugins), 1)
	x.MergeCell(sheet, cell, end)
	// x.SetCellStyle(sheet, cell, end, topHeading)

	x.SetRowStyle(sheet, 1, 2, topHeading)
	return x.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		XSplit:      1,
		YSplit:      2,
		TopLeftCell: "B3",
		ActivePane:  "topRight",
		Selection: []excelize.Selection{
			{SQRef: "B3", ActiveCell: "B3", Pane: "topRight"},
		},
	})

	// return
}

func outputXLSXSingleColumn(x *excelize.File, Entities []Entity, cf *config.Config, conftable config.ExpandOptions, plugin string) (err error) {
	rows := 0
	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				rows += len(s.Column1)
			}
		}
	}
	if rows == 0 && cf.GetBool("output.skip-empty-reports") {
		return
	}

	sheet := cf.GetString(
		config.Join("output", "reports", plugin, "sheetname"),
		config.Default(strings.ToTitle(plugin)),
	)

	if _, err = x.NewSheet(sheet); err != nil {
		return
	}

	columns := cf.GetStringSlice(
		config.Join("output", "reports", plugin, "columns"),
		config.Default([]string{
			"managedEntity",
			"samplerType",
			"samplerName",
			"data",
		}))

	if err = x.SetSheetRow(sheet, "A1", &columns); err != nil {
		return
	}

	colwidths := []float64{}
	for _, c := range columns {
		colwidths = append(colwidths, colWidth(len(c), minColWidth))
	}

	rownum := 1
	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				for _, data := range s.Column1 {
					rownum++
					row := []string{
						e.Name,
						s.Type,
						s.Name,
						data,
					}
					if err = x.SetSheetRow(sheet, fmt.Sprintf("A%d", rownum), &row); err != nil {
						return
					}
					for j, c := range row {
						colwidths[j] = colWidth(len(fmt.Sprint(c)), colwidths[j])
					}
				}
			}
		}
	}

	// mark up no data
	if rownum == 1 {
		message := fmt.Sprintf("[No %q samplers]", plugin)
		if err = x.SetSheetRow(sheet, "A2", &[]string{message}); err != nil {
			return
		}
		colwidths[1] = colWidth(len(message)*2, colwidths[1])
	}

	// set column widths
	for i, c := range colwidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		if err = x.SetColWidth(sheet, col, col, c); err != nil {
			return
		}
	}

	x.SetColStyle(sheet, "D", dataColumnStyle)
	x.SetRowStyle(sheet, 1, 1, topHeading)
	return x.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
		Selection: []excelize.Selection{
			{SQRef: "A2", ActiveCell: "A2", Pane: "bottomLeft"},
		},
	})
}

// output two columns of data, sort both lists, output blanks for shorter list
func outputXLSXTwoColumn(x *excelize.File, Entities []Entity, cf *config.Config, conftable config.ExpandOptions, plugin string) (err error) {
	sheet := cf.GetString(config.Join("output", "reports", plugin, "sheetname"), config.Default(strings.ToTitle(plugin)))

	if _, err = x.NewSheet(sheet); err != nil {
		return
	}

	columns := cf.GetStringSlice(
		config.Join("output", "reports", plugin, "columns"),
		config.Default([]string{
			"managedEntity",
			"samplerType",
			"samplerName",
			"data1",
			"data2",
		}))

	if err = x.SetSheetRow(sheet, "A1", &columns); err != nil {
		return
	}

	colwidths := []float64{}
	for _, c := range columns {
		colwidths = append(colwidths, colWidth(len(c), minColWidth))
	}

	rownum := 1
	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				items := len(s.Column1)
				if len(s.Column2) > items {
					items = len(s.Column2)
				}

				for i := 0; i < items; i++ {
					rownum++
					row := []string{
						e.Name,
						s.Type,
						s.Name,
					}
					if len(s.Column1) > i {
						row = append(row, s.Column1[i])
					} else {
						row = append(row, "")
					}
					if len(s.Column2) > i {
						row = append(row, s.Column2[i])
					} else {
						row = append(row, "")
					}

					if err = x.SetSheetRow(sheet, fmt.Sprintf("A%d", rownum), &row); err != nil {
						return
					}
					for j, c := range row {
						colwidths[j] = colWidth(len(fmt.Sprint(c)), colwidths[j])
					}
				}
			}
		}
	}

	// mark up no data
	if rownum == 1 {
		if cf.GetBool("output.skip-empty-reports") {
			x.DeleteSheet(sheet)
			return
		}
		message := fmt.Sprintf("[No %q samplers]", plugin)
		if err = x.SetSheetRow(sheet, "A2", &[]string{message}); err != nil {
			return
		}
		colwidths[1] = colWidth(len(message)*2, colwidths[1])
	}

	// set column widths
	for i, c := range colwidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		if err = x.SetColWidth(sheet, col, col, c); err != nil {
			return
		}
	}

	x.SetColStyle(sheet, "D:E", dataColumnStyle)
	x.SetRowStyle(sheet, 1, 1, topHeading)
	return x.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
		Selection: []excelize.Selection{
			{SQRef: "A2", ActiveCell: "A2", Pane: "bottomLeft"},
		},
	})
}
