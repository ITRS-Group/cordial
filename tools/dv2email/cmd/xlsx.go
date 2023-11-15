package cmd

import (
	"bytes"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xuri/excelize/v2"

	"github.com/itrs-group/cordial/pkg/config"
)

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
		"Sample Time",
		"Gateway",
		"Probe",
		"Entity",
		"Sampler",
		"Dataview",
		"XPath",
	}

	infoTable, _     = excelize.CoordinatesToCellName(infoColumn, infoRow)
	infoTableData, _ = excelize.CoordinatesToCellName(infoColumn, infoRow+1)
	headlineTable, _ = excelize.CoordinatesToCellName(headlineColumn, headlineRow)
	dataviewTable, _ = excelize.CoordinatesToCellName(dataviewColumn, dataviewRow)
	rownamesStart, _ = excelize.CoordinatesToCellName(dataviewColumn, dataviewRow+1)
)

func createXLSX(cf *config.Config, data dv2emailData) (buf *bytes.Buffer, err error) {
	rowStripes := cf.GetBool("attachments::xlsx::row-stripes")

	// if zero then auto-size based on widest value, else fixed
	columnWidth := cf.GetFloat64("attachments::xlsx::column-width")

	x := excelize.NewFile()

	headingStyle, _ := x.NewStyle(&HeadingCellStyle)
	criticalStyle, _ := x.NewStyle(&CriticalCellStyle)
	warningStyle, _ := x.NewStyle(&WarningCellStyle)
	okStyle, _ := x.NewStyle(&OKCellStyle)

	for _, dv := range data.Dataviews {
		lookup := dv.XPath.LookupValues()

		sheetname := cf.GetString("attachments::xlsx::sheetname", config.LookupTable(lookup))
		if _, err = x.NewSheet(sheetname); err != nil {
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
		info := []any{
			"Geneos dv2email",
			dv.SampleTime.Format(time.RFC3339),
			lookup["gateway"],
			lookup["probe"],
			lookup["entity"],
			lookup["sampler"],
			lookup["dataview"],
			dv.XPath.String(),
		}

		infoTableHeadingsEnd, _ := excelize.CoordinatesToCellName(infoColumn+len(info), infoRow)
		x.SetCellStyle(sheetname, infoTable, infoTableHeadingsEnd, headingStyle)

		if err = x.SetSheetRow(sheetname, infoTableData, &info); err != nil {
			log.Error().Err(err).Msg("")
		}

		colwidths := make([]float64, max(len(dv.Headlines), len(dv.Columns), len(info)))

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
		if err = x.SetSheetRow(sheetname, dataviewTable, &dv.Columns); err != nil {
			log.Error().Err(err).Msg("")
		}

		for i, c := range dv.Columns {
			colwidths[i] = max(colWidth(len(c), minColWidth), colwidths[i])
		}

		// set rownames in first column, starting on second row (below headings)
		if err = x.SetSheetCol(sheetname, rownamesStart, &dv.Rows); err != nil {
			log.Error().Err(err).Msg("")
		}

		for ri, r := range dv.Rows {
			// update the width of the first column based on each rowname
			colwidths[0] = max(colWidth(len(r), minColWidth), colwidths[0])

			for ci, c := range dv.Columns[1:] { // skip rowname column
				cell, _ := excelize.CoordinatesToCellName(dataviewColumn+ci+1, dataviewRow+ri+1)
				value := dv.Table[r][c].Value
				x.SetCellStr(sheetname, cell, value)
				colwidths[ci+1] = max(colWidth(len(value), minColWidth), colwidths[ci+1])
			}
		}

		end, _ := excelize.CoordinatesToCellName(dataviewColumn+len(dv.Columns)-1, dataviewRow+len(dv.Rows))
		tablename := validTablename(sheetname, "Dataview")
		err = x.AddTable(sheetname, &excelize.Table{
			Range:             dataviewTable + ":" + end,
			Name:              tablename,
			StyleName:         cf.GetString("attachments::xlsx::style", config.Default("TableStyleMedium2")),
			ShowFirstColumn:   true,
			ShowLastColumn:    false,
			ShowRowStripes:    &rowStripes,
			ShowColumnStripes: cf.GetBool("attachments::xlsx::column-stripes"),
		})
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		for ri, r := range dv.Rows {
			for ci, c := range dv.Columns[1:] { // skip rowname column
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
	if err = x.Write(buf, excelize.Options{Password: cf.GetString("attachments::xlsx::password")}); err != nil {
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
