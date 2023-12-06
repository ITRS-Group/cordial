package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
)

func createTextTemplate(cf *config.Config, data DV2EMailData, textTemplate string) (text string, err error) {
	tt, err := template.New("dataview").Parse(textTemplate)
	if err != nil {
		return
	}

	var body strings.Builder
	err = tt.Execute(&body, data)
	if err != nil {
		return
	}
	text = body.String()
	return
}

func createTextTables(cf *config.Config, data DV2EMailData) (buf *bytes.Buffer, err error) {
	buf = &bytes.Buffer{}

	t := time.Now()
	digits := len(strconv.Itoa(len(data.Dataviews)))

	for di, dv := range data.Dataviews {
		lookup := dv.XPath.LookupValues()
		lookup["date"] = t.Local().Format("20060102")
		lookup["time"] = t.Local().Format("150405")
		lookup["datetime"] = t.Local().Format(time.RFC3339)
		lookup["serial"] = fmt.Sprintf("%0*d", digits, di)

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

		tab := table.NewWriter()
		tab.SetOutputMirror(buf)
		// tab.SetAllowedRowLength(132)
		tab.SetTitle("Information")

		tab.SetColumnConfigs([]table.ColumnConfig{
			{Number: 2, WidthMax: 60, WidthMaxEnforcer: text.WrapHard},
		})
		for i, v := range infoNames[1:] {
			tab.AppendRow([]any{v, info[i+1]})
		}

		tab.Render()
		fmt.Fprintln(buf)

		headlinesHeadings := []string{}
		for h := range dv.Headlines {
			headlinesHeadings = append(headlinesHeadings, h)
		}
		sort.Strings(headlinesHeadings)

		headlines := []table.Row{}
		for _, h := range headlinesHeadings {
			headlines = append(headlines, table.Row{h, dv.Headlines[h].Value})
		}

		tab.ResetHeaders()
		tab.ResetRows()
		tab.SetColumnConfigs([]table.ColumnConfig{
			{Number: 2, WidthMax: 0},
		})
		tab.Style().Format.Header = text.FormatDefault
		tab.SetTitle("Headlines")
		// tab2.AppendHeader(table.Row{"Headline", "Value"})
		tab.AppendRows(headlines)
		tab.Render()
		fmt.Fprintln(buf)

		tab.ResetHeaders()
		tab.ResetRows()
		tab.SetTitle("")
		tab.SetAllowedRowLength(0)
		tab.Style().Format.Header = text.FormatDefault

		headings := table.Row{}
		for _, h := range dv.ColumnOrder {
			headings = append(headings, h)
		}
		tab.AppendHeader(headings)
		for _, rn := range dv.RowOrder {
			row := table.Row{rn}
			for _, cn := range dv.ColumnOrder[1:] {
				row = append(row, dv.Table[rn][cn].Value)
			}
			tab.AppendRow(row)
		}
		tab.Render()
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, strings.Repeat("=", 130))
		fmt.Fprintln(buf)
	}

	fmt.Fprintln(buf)

	return
}

func buildTextTableFiles(cf *config.Config, data DV2EMailData, timestamp time.Time) (files []dataFile, err error) {
	lookupDateTime := map[string]string{
		"date":     timestamp.Local().Format("20060102"),
		"time":     timestamp.Local().Format("150405"),
		"datetime": timestamp.Local().Format(time.RFC3339),
	}
	switch cf.GetString("texttable.split") {
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
			text, err := createTextTables(cf, many)
			if err != nil {
				return files, err
			}
			lookup := map[string]string{
				"default":   "dataviews",
				"entity":    entity,
				"sampler":   "",
				"dataview":  "",
				"timestamp": timestamp.Local().Format("20060102150405"),
			}
			filename := buildName(cf.GetString("texttable.filename", config.LookupTable(lookupDateTime)), lookup) + ".txt"
			files = append(files, dataFile{
				name:    filename,
				content: text,
			})
		}
	case "dataview":
		for _, d := range data.Dataviews {
			one := DV2EMailData{
				Dataviews: []*commands.Dataview{d},
				Env:       data.Env,
			}
			text, err := createTextTables(cf, one)
			if err != nil {
				return files, err
			}
			lookup := map[string]string{
				"default":   "dataviews",
				"entity":    d.XPath.Entity.Name,
				"sampler":   d.XPath.Sampler.Name,
				"dataview":  d.XPath.Dataview.Name,
				"timestamp": timestamp.Local().Format("20060102150405"),
			}
			filename := buildName(cf.GetString("texttable.filename", config.LookupTable(lookupDateTime)), lookup) + ".txt"
			files = append(files, dataFile{
				name:    filename,
				content: text,
			})
		}
	default:
		text, err := createTextTables(cf, data)
		if err != nil {
			return files, err
		}
		lookup := map[string]string{
			"default":   "dataviews",
			"entity":    "",
			"sampler":   "",
			"dataview":  "",
			"timestamp": timestamp.Local().Format("20060102150405"),
		}
		filename := buildName(cf.GetString("texttable.filename", config.LookupTable(lookupDateTime)), lookup) + ".txt"
		files = append(files, dataFile{
			name:    filename,
			content: text,
		})
	}

	return
}
