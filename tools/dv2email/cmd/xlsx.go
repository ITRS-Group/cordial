/*
Copyright © 2023 ITRS Group

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

package cmd

import (
	"bytes"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos/commands"
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
		"Type",
		"Sampler",
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

func createXLSX(cf *config.Config, data DV2EMailData) (out *bytes.Reader, err error) {
	rowStripes := config.Get[bool](cf, cf.Join("xlsx", "row-stripes"))

	// if zero then auto-size based on widest value, else fixed
	columnWidth := config.Get[float64](cf, "xlsx.column-width")

	x := excelize.NewFile()
	defer x.Close()

	headingStyle, _ := x.NewStyle(&HeadingCellStyle)
	criticalStyle, _ := x.NewStyle(&CriticalCellStyle)
	warningStyle, _ := x.NewStyle(&WarningCellStyle)
	okStyle, _ := x.NewStyle(&OKCellStyle)

	t := time.Now()

	// number of digits needed for "serial" - the length of string of the number of the length of dataviews
	digits := len(strconv.Itoa(len(data.Dataviews)))

	// sort by entity and dataview, without changing the caller's slice
	// and calculating the sort keys only once
	dataviews := slices.Clone(data.Dataviews)
	keys := make(map[*commands.Dataview]string, len(dataviews))
	for _, dv := range dataviews {
		keys[dv] = dv.XPath.String()
	}
	slices.SortStableFunc(dataviews, func(a, b *commands.Dataview) int {
		return strings.Compare(keys[a], keys[b])
	})

	// the default sheet is deleted below so must not be reused
	usedSheets := map[string]bool{"sheet1": true}
	usedTables := map[string]bool{}

	for di, dv := range dataviews {
		var sheetname string
		lookup := dv.XPath.LookupValues()
		lookup["date"] = t.Local().Format("20060102")
		lookup["time"] = t.Local().Format("150405")
		lookup["datetime"] = t.Local().Format(time.RFC3339)
		lookup["serial"] = fmt.Sprintf("%0*d", digits, di+1)

		// if sheetname is "auto" then apply some heuristics

		sheetname = config.Get[string](cf, "xlsx.sheetname", config.LookupTable(lookup))
		sheetname = uniqueSheetName(buildName(sheetname, lookup), usedSheets)

		if _, err = x.NewSheet(sheetname); err != nil {
			log.Error("failed to create new sheet", slog.Any("error", err), slog.String("sheetname", sheetname))
			continue
		}

		// build initial meta info table
		//
		// we do not set the column widths, as the xpath will definitely
		// be too wide. instead rely on later data to drive this, or the
		// default column width setting

		if err = x.SetSheetRow(sheetname, infoTable, &infoNames); err != nil {
			log.Error("failed to set sheet row", slog.Any("error", err), slog.String("sheetname", sheetname))
		}
		info := dataviewInfo(dv, lookup)

		infoTableHeadingsEnd, _ := excelize.CoordinatesToCellName(infoColumn+len(info)-1, infoRow)
		x.SetCellStyle(sheetname, infoTable, infoTableHeadingsEnd, headingStyle)

		if err = x.SetSheetRow(sheetname, infoTableData, &info); err != nil {
			log.Error("failed to set sheet row", slog.Any("error", err), slog.String("sheetname", sheetname))
		}

		colwidths := make([]float64, max(len(dv.Headlines), len(dv.ColumnOrder), len(info)))

		// build two rows of headlines, which always have at least a samplingStatus

		// sorted slice of headlines
		headlines := slices.Sorted(maps.Keys(dv.Headlines))
		if err = x.SetSheetRow(sheetname, headlineTable, &headlines); err != nil {
			log.Error("failed to set sheet row", slog.Any("error", err), slog.String("sheetname", sheetname))
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
		if len(headlines) > 0 {
			headlinesEnd, _ := excelize.CoordinatesToCellName(headlineColumn+len(headlines)-1, headlineRow)
			x.SetCellStyle(sheetname, headlineTable, headlinesEnd, headingStyle)
		}

		// build main dataview table

		// set columns names as the first row
		if err = x.SetSheetRow(sheetname, dataviewTable, &dv.ColumnOrder); err != nil {
			log.Error("failed to set sheet row", slog.Any("error", err), slog.String("sheetname", sheetname))
		}

		for i, c := range dv.ColumnOrder {
			colwidths[i] = max(colWidth(len(c), minColWidth), colwidths[i])
		}

		// set rownames in first column, starting on second row (below headings)
		if err = x.SetSheetCol(sheetname, rownamesStart, &dv.RowOrder); err != nil {
			log.Error("failed to set sheet column", slog.Any("error", err), slog.String("sheetname", sheetname))
		}

		for ri, r := range dv.RowOrder {
			// update the width of the first column based on each rowname
			colwidths[0] = max(colWidth(len(r), minColWidth), colwidths[0])

			for ci, c := range dataColumns(dv) { // skip rowname column
				cell, _ := excelize.CoordinatesToCellName(dataviewColumn+ci+1, dataviewRow+ri+1)
				value := dv.Table[r][c].Value
				x.SetCellStr(sheetname, cell, value)
				colwidths[ci+1] = max(colWidth(len(value), minColWidth), colwidths[ci+1])
			}
		}

		end, _ := excelize.CoordinatesToCellName(dataviewColumn+len(dv.ColumnOrder)-1, dataviewRow+len(dv.RowOrder))
		tablename := uniqueName(validTablename(sheetname, "Dataview"), usedTables)
		err = x.AddTable(sheetname, &excelize.Table{
			Range:             dataviewTable + ":" + end,
			Name:              tablename,
			StyleName:         config.Get[string](cf, "xlsx.style", config.DefaultValue("TableStyleMedium2")),
			ShowFirstColumn:   true,
			ShowLastColumn:    false,
			ShowRowStripes:    &rowStripes,
			ShowColumnStripes: config.Get[bool](cf, cf.Join("xlsx", "column-stripes")),
		})
		if err != nil {
			log.Error("failed to add table", slog.Any("error", err), slog.String("sheetname", sheetname))
		}

		for ri, r := range dv.RowOrder {
			for ci, c := range dataColumns(dv) { // skip rowname column
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

	buf := &bytes.Buffer{}
	if err = x.Write(buf, excelize.Options{
		Password: config.Get[string](cf, "xlsx.password"),
	}); err != nil {
		return
	}
	out = bytes.NewReader(buf.Bytes())
	return
}

// dataviewInfo returns the values for the "Information" table, in the
// order of infoNames
func dataviewInfo(dv *commands.Dataview, lookup map[string]string) []string {
	return []string{
		"Geneos dv2email",
		lookup["gateway"],
		lookup["probe"],
		lookup["entity"],
		lookup["type"],
		lookup["sampler"],
		lookup["dataview"],
		dv.SampleTime.Format(time.RFC3339),
		dv.XPath.String(),
	}
}

// dataColumns returns the column names of a dataview, excluding the
// first (row name) column
func dataColumns(dv *commands.Dataview) []string {
	if len(dv.ColumnOrder) == 0 {
		return nil
	}
	return dv.ColumnOrder[1:]
}

// maxSheetName is the maximum length, in characters, of an Excel sheet name
const maxSheetName = 31

// uniqueName returns name, or name with a numeric suffix if it has
// already been used (ignoring case, as Excel does), and records it in
// used. The suffix only uses characters valid in a table name.
func uniqueName(name string, used map[string]bool) string {
	candidate := name
	for i := 2; used[strings.ToLower(candidate)]; i++ {
		candidate = name + "_" + strconv.Itoa(i)
	}
	used[strings.ToLower(candidate)] = true
	return candidate
}

// uniqueSheetName returns a valid Excel sheet name, based on name, that
// has not already been used. Invalid characters are replaced with an
// underscore, and the name is truncated to 31 characters (not bytes),
// before making it unique.
func uniqueSheetName(name string, used map[string]bool) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case ':', '\\', '/', '?', '*', '[', ']':
			return '_'
		}
		return r
	}, name)
	name = strings.Trim(name, "'")
	if name == "" {
		name = "dataviews"
	}

	runes := []rune(name)
	if len(runes) > maxSheetName {
		log.Debug("truncating", slog.String("sheetname", name), slog.String("truncated", string(runes[:maxSheetName])))
		runes = runes[:maxSheetName]
	}
	name = string(runes)

	candidate := name
	for i := 2; used[strings.ToLower(candidate)]; i++ {
		suffix := "~" + strconv.Itoa(i)
		base := []rune(name)
		if len(base)+len([]rune(suffix)) > maxSheetName {
			base = base[:maxSheetName-len([]rune(suffix))]
		}
		candidate = string(base) + suffix
	}
	used[strings.ToLower(candidate)] = true
	return candidate
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

func buildXLSXFiles(cf *config.Config, d any, timestamp time.Time) (files []dataFile, err error) {
	data, ok := d.(DV2EMailData)
	if !ok {
		err = os.ErrInvalid
		return
	}

	lookupDateTime := map[string]string{
		"date":     timestamp.Local().Format("20060102"),
		"time":     timestamp.Local().Format("150405"),
		"datetime": timestamp.Local().Format(time.RFC3339),
	}
	switch config.Get[string](cf, "xlsx.split") {
	case "entity":
		for _, group := range groupByEntity(data.Dataviews) {
			many := DV2EMailData{
				Dataviews: group.dataviews,
				Env:       data.Env,
			}
			lookup := map[string]string{
				"default":   "dataviews",
				"entity":    group.entity,
				"sampler":   "",
				"dataview":  "",
				"timestamp": timestamp.Local().Format("20060102150405"),
			}
			buf, err := createXLSX(cf, many)
			if err != nil {
				return files, err
			}
			filename := buildName(config.Get[string](cf, "xlsx.filename", config.LookupTable(lookupDateTime)), lookup) + ".xlsx"
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
			filename := buildName(config.Get[string](cf, "xlsx.filename", config.LookupTable(lookupDateTime)), lookup) + ".xlsx"
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
		filename := buildName(config.Get[string](cf, "xlsx.filename", config.LookupTable(lookupDateTime)), lookup) + ".xlsx"
		files = append(files, dataFile{
			name:    filename,
			content: buf,
		})
	}

	return
}
