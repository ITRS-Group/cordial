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
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xuri/excelize/v2"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
)

// Hardwired cell defaults
var (
	CriticalCellStyle = excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"DC143C"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Color: "FFFFFF",
		},
	}
	WarningCellStyle = excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"FFD700"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Color: "000000",
		},
	}
	OKCellStyle = excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"32cd32"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Color: "FFFFFF",
		},
	}
	HeadingCellStyle = excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
	}
)

const (
	infoRow    = 1
	infoColumn = 1

	headlineRow    = infoRow + 3
	headlineColumn = 1

	dataviewRow    = headlineRow + 3
	dataviewColumn = 1
)

var (
	infoNames = []string{
		"Source",
		"Gateway",
		"Probe",
		"Entity",
		"Sampler",
		"Type",
		"Dataview",
		"Sample Time",
		"XPath",
	}

	infoTable, _     = excelize.CoordinatesToCellName(infoColumn, infoRow)
	infoTableData, _ = excelize.CoordinatesToCellName(infoColumn, infoRow+1)
	headlineTable, _ = excelize.CoordinatesToCellName(headlineColumn, headlineRow)
	dataviewTable, _ = excelize.CoordinatesToCellName(dataviewColumn, dataviewRow)
	rownamesStart, _ = excelize.CoordinatesToCellName(dataviewColumn, dataviewRow+1)
)

func createXLSX(cf *config.Config, data DV2EMailData) (buf *bytes.Buffer, err error) {
	rowStripes := cf.GetBool("xlsx.row-stripes")

	// if zero then auto-size based on widest value, else fixed
	columnWidth := cf.GetFloat64("xlsx.column-width")

	x := excelize.NewFile()

	headingStyle, _ := x.NewStyle(&HeadingCellStyle)
	criticalStyle, _ := x.NewStyle(&CriticalCellStyle)
	warningStyle, _ := x.NewStyle(&WarningCellStyle)
	okStyle, _ := x.NewStyle(&OKCellStyle)

	t := time.Now()

	// number of digits needed for "serial" - the length of string of the number of the length of dataviews
	digits := len(strconv.Itoa(len(data.Dataviews)))

	// sort by entity and dataview
	sort.Slice(data.Dataviews, func(i, j int) bool {
		return data.Dataviews[i].XPath.String() < data.Dataviews[j].XPath.String()
	})

	for di, dv := range data.Dataviews {
		var sheetname string
		lookup := dv.XPath.LookupValues()
		lookup["date"] = t.Local().Format("20060102")
		lookup["time"] = t.Local().Format("150405")
		lookup["datetime"] = t.Local().Format(time.RFC3339)
		lookup["serial"] = fmt.Sprintf("%0*d", digits, di)

		// if sheetname is "auto" then apply some heuristics

		sheetname = cf.GetString("xlsx.sheetname", config.LookupTable(lookup))
		sheetname = buildName(sheetname, lookup)

		if len(sheetname) > 31 {
			log.Debug().Msgf("truncating %s to %s", sheetname, sheetname[:31])
			sheetname = sheetname[:31]
		}

		if _, err = x.NewSheet(sheetname); err != nil {
			log.Error().Err(err).Msgf(`new sheet "%s"`, sheetname)
			continue
		}

		// build initial meta info table
		//
		// we do not set the column widths, as the xpath will definitely
		// be too wide. instead rely on later data to drive this, or the
		// default column width setting

		if err = x.SetSheetRow(sheetname, infoTable, &infoNames); err != nil {
			log.Error().Err(err).Msg("")
		}
		info := []string{
			"Geneos dv2email",
			lookup["gateway"],
			lookup["probe"],
			lookup["entity"],
			lookup["sampler"],
			lookup["dataview"],
			dv.SampleTime.Format(time.RFC3339),
			dv.XPath.String(),
		}

		infoTableHeadingsEnd, _ := excelize.CoordinatesToCellName(infoColumn+len(info), infoRow)
		x.SetCellStyle(sheetname, infoTable, infoTableHeadingsEnd, headingStyle)

		if err = x.SetSheetRow(sheetname, infoTableData, &info); err != nil {
			log.Error().Err(err).Msg("")
		}

		colwidths := make([]float64, max(len(dv.Headlines), len(dv.ColumnOrder), len(info)))

		// build two rows of headlines, which always have at least a samplingStatus

		// sorted slice of headlines
		headlines := []string{}
		for h := range dv.Headlines {
			headlines = append(headlines, h)
		}
		sort.Strings(headlines)
		if err = x.SetSheetRow(sheetname, headlineTable, &headlines); err != nil {
			log.Error().Err(err).Msg("")
		}

		for hi, hn := range headlines {
			cell, _ := excelize.CoordinatesToCellName(headlineColumn+hi, headlineRow+1)
			value := dv.Headlines[hn].Value
			x.SetCellStr(sheetname, cell, value)
			switch dv.Headlines[hn].Severity {
			case "CRITICAL":
				x.SetCellStyle(sheetname, cell, cell, criticalStyle)
			case "WARNING":
				x.SetCellStyle(sheetname, cell, cell, warningStyle)
			case "OK":
				x.SetCellStyle(sheetname, cell, cell, okStyle)
			}
			colwidths[hi] = max(colWidth(len(value), minColWidth), colwidths[hi])
		}
		headlinesEnd, _ := excelize.CoordinatesToCellName(headlineColumn+len(dv.Headlines)-1, headlineRow)
		x.SetCellStyle(sheetname, headlineTable, headlinesEnd, headingStyle)

		// build main dataview table

		// set columns names as the first row
		if err = x.SetSheetRow(sheetname, dataviewTable, &dv.ColumnOrder); err != nil {
			log.Error().Err(err).Msg("")
		}

		for i, c := range dv.ColumnOrder {
			colwidths[i] = max(colWidth(len(c), minColWidth), colwidths[i])
		}

		// set rownames in first column, starting on second row (below headings)
		if err = x.SetSheetCol(sheetname, rownamesStart, &dv.RowOrder); err != nil {
			log.Error().Err(err).Msg("")
		}

		for ri, r := range dv.RowOrder {
			// update the width of the first column based on each rowname
			colwidths[0] = max(colWidth(len(r), minColWidth), colwidths[0])

			for ci, c := range dv.ColumnOrder[1:] { // skip rowname column
				cell, _ := excelize.CoordinatesToCellName(dataviewColumn+ci+1, dataviewRow+ri+1)
				value := dv.Table[r][c].Value
				x.SetCellStr(sheetname, cell, value)
				colwidths[ci+1] = max(colWidth(len(value), minColWidth), colwidths[ci+1])
			}
		}

		end, _ := excelize.CoordinatesToCellName(dataviewColumn+len(dv.ColumnOrder)-1, dataviewRow+len(dv.RowOrder))
		tablename := validTablename(sheetname, "Dataview")
		err = x.AddTable(sheetname, &excelize.Table{
			Range:             dataviewTable + ":" + end,
			Name:              tablename,
			StyleName:         cf.GetString("xlsx.style", config.Default("TableStyleMedium2")),
			ShowFirstColumn:   true,
			ShowLastColumn:    false,
			ShowRowStripes:    &rowStripes,
			ShowColumnStripes: cf.GetBool("xlsx.column-stripes"),
		})
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		for ri, r := range dv.RowOrder {
			for ci, c := range dv.ColumnOrder[1:] { // skip rowname column
				cell, _ := excelize.CoordinatesToCellName(dataviewColumn+ci+1, dataviewRow+ri+1)
				switch dv.Table[r][c].Severity {
				case "CRITICAL":
					x.SetCellStyle(sheetname, cell, cell, criticalStyle)
				case "WARNING":
					x.SetCellStyle(sheetname, cell, cell, warningStyle)
				case "OK":
					x.SetCellStyle(sheetname, cell, cell, okStyle)
				}
			}
		}

		// set column widths
		if columnWidth == 0 {
			for i, c := range colwidths {
				col, _ := excelize.ColumnNumberToName(i + 1)
				if err = x.SetColWidth(sheetname, col, col, c); err != nil {
					return
				}
			}
		} else {
			endcol, _ := excelize.ColumnNumberToName(len(colwidths))
			if err = x.SetColWidth(sheetname, "A", endcol, columnWidth); err != nil {
				return
			}

		}
	}

	// now remove the default sheet
	x.DeleteSheet("Sheet1")

	buf = &bytes.Buffer{}
	if err = x.Write(buf, excelize.Options{
		Password: cf.GetString("xlsx.password"),
	}); err != nil {
		return
	}
	if err = x.Close(); err != nil {
		return
	}
	return
}

// a scale factor for the column width versus string len
const colScale = 1.2

// minimum column width
var minColWidth = 10.0

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

// validTablename returns a tablename that conforms to Excel
// requirements. sheetname is prefixed with prefix and an underscore,
// then characters are checked and all invalid ones are replaced with an
// underscore. If the tablename does not start with a letter, underscore
// or backslash then an underscore is prefixed.
func validTablename(sheetname, prefix string) (tablename string) {
	tablename = prefix + "_" + sheetname
	tablename = strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z':
			return r
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '_' || r == '.' || r == '\\':
			return r
		default:
			return '_'
		}
	}, tablename)
	// prefix a table with underscore is first letter not valid
	r := tablename[0]
	switch {
	case r >= 'A' && r <= 'Z':
		break
	case r >= 'a' && r <= 'z':
		break
	case r == '_' || r == '\\':
		break
	default:
		tablename = "_" + tablename
	}
	return
}

func buildXLSXFiles(cf *config.Config, data DV2EMailData, timestamp time.Time) (files []dataFile, err error) {
	lookupDateTime := map[string]string{
		"date":     timestamp.Local().Format("20060102"),
		"time":     timestamp.Local().Format("150405"),
		"datetime": timestamp.Local().Format(time.RFC3339),
	}
	switch cf.GetString("xlsx.split") {
	case "entity":
		entities := map[string][]*commands.Dataview{}
		for _, d := range data.Dataviews {
			if len(entities[d.XPath.Entity.Name]) == 0 {
				entities[d.XPath.Entity.Name] = []*commands.Dataview{}
			}
			entities[d.XPath.Entity.Name] = append(entities[d.XPath.Entity.Name], d)
		}
		for entity, e := range entities {
			many := DV2EMailData{
				Dataviews: e,
				Env:       data.Env,
			}
			lookup := map[string]string{
				"default":   "dataviews",
				"entity":    entity,
				"sampler":   "",
				"dataview":  "",
				"timestamp": timestamp.Local().Format("20060102150405"),
			}
			buf, err := createXLSX(cf, many)
			if err != nil {
				return files, err
			}
			filename := buildName(cf.GetString("xlsx.filename", config.LookupTable(lookupDateTime)), lookup) + ".xlsx"
			files = append(files, dataFile{
				name:    filename,
				content: buf,
			})
		}
	case "dataview":
		for _, d := range data.Dataviews {
			one := DV2EMailData{
				Dataviews: []*commands.Dataview{d},
				Env:       data.Env,
			}
			lookup := map[string]string{
				"default":   "dataviews",
				"entity":    d.XPath.Entity.Name,
				"sampler":   d.XPath.Sampler.Name,
				"dataview":  d.XPath.Dataview.Name,
				"timestamp": timestamp.Local().Format("20060102150405"),
			}
			buf, err := createXLSX(cf, one)
			if err != nil {
				return files, err
			}
			filename := buildName(cf.GetString("xlsx.filename", config.LookupTable(lookupDateTime)), lookup) + ".xlsx"
			files = append(files, dataFile{
				name:    filename,
				content: buf,
			})
		}
	default:
		lookup := map[string]string{
			"default":   "dataviews",
			"entity":    "",
			"sampler":   "",
			"dataview":  "",
			"timestamp": timestamp.Local().Format("20060102150405"),
		}

		buf, err := createXLSX(cf, data)
		if err != nil {
			return files, err
		}
		filename := buildName(cf.GetString("xlsx.filename", config.LookupTable(lookupDateTime)), lookup) + ".xlsx"
		files = append(files, dataFile{
			name:    filename,
			content: buf,
		})
	}

	return
}
