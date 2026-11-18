//go:build !noxlsx

package reporter

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

func renderXLSX(t *testing.T, r Reporter, buf *bytes.Buffer) *excelize.File {
	t.Helper()
	if err := r.Render(); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestStringToCell(t *testing.T) {
	big := strings.Repeat("9", 25)
	tests := []struct {
		in   string
		want any
	}{
		{"42", int64(42)},
		{"0", int64(0)},
		{"007", "007"},
		{"00", "00"},
		{big, big},
		{"9223372036854775807", int64(9223372036854775807)},
		{"9223372036854775808", "9223372036854775808"},
		{"50%", 0.5},
		{"50 %", 0.5},
		{"text", "text"},
		{"", ""},
		{"2025-01-02T03:04:05Z", time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := stringToCell(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stringToCell(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestXLSXReporterRobustToOddTables(t *testing.T) {
	var buf bytes.Buffer
	r, err := NewReporter("xlsx", &buf, XLSXScramble(true))
	if err != nil {
		t.Fatal(err)
	}
	// no sheet yet, and no sheet prepared: must be ignored
	r.UpdateTable([]string{"a"}, [][]string{{"x"}})
	r.AddHeadline("ignored", "x")

	long := strings.Repeat("é", 40)
	if err := r.Prepare(Report{Title: long, ScrambleColumns: []string{"b"}}); err != nil {
		t.Fatal(err)
	}
	if n := utf8.RuneCountInString(long); n != 40 {
		t.Fatal(n)
	}

	rows := [][]string{
		{"dup", "left.example", "1"},
		{"dup", "right.example", "2"}, // same first column
		{},                            // empty row
		{"wide", "w.example", "3", "extra", "cells"}, // more cells than columns
		{"short"}, // fewer cells than columns
	}
	orig := [][]string{
		{"dup", "left.example", "1"},
		{"dup", "right.example", "2"},
		{},
		{"wide", "w.example", "3", "extra", "cells"},
		{"short"},
	}
	r.UpdateTable([]string{"a", "b", "c"}, rows)
	if !reflect.DeepEqual(rows, orig) {
		t.Errorf("caller rows modified: %v", rows)
	}

	f := renderXLSX(t, r, &buf)
	var sheet string
	for _, s := range f.GetSheetList() {
		if s != "Summary" {
			sheet = s
		}
	}
	if utf8.RuneCountInString(sheet) != 31 || !utf8.ValidString(sheet) {
		t.Errorf("sheet name %q has %d runes, want 31 valid characters", sheet, utf8.RuneCountInString(sheet))
	}
	got, err := f.GetRows(sheet)
	if err != nil {
		t.Fatal(err)
	}
	// header + 5 rows, the empty row being blank
	if len(got) < 6 {
		t.Fatalf("rows = %v, want a header and 5 rows (including duplicates)", got)
	}
	if got[1][0] != "dup" || got[2][0] != "dup" || got[1][1] == got[2][1] {
		t.Errorf("duplicate first-column rows not both kept: %v", got[1:3])
	}
	if strings.Contains(got[1][1], "example") || strings.Contains(got[2][1], "example") {
		t.Errorf("column b not scrambled by XLSXScramble: %v", got[1:3])
	}
	if got[4][0] != "wide" {
		t.Errorf("rows = %v", got)
	}
}

func TestXLSXReporterUpdateTableReplacesData(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewReporter("xlsx", &buf)
	_ = r.Prepare(Report{Title: "Data"})
	r.UpdateTable([]string{"a", "b"}, [][]string{{"1", "2"}, {"3", "4"}})
	r.UpdateTable([]string{"a", "b"}, [][]string{{"5", "6"}})
	f := renderXLSX(t, r, &buf)
	got, _ := f.GetRows("Data")
	want := [][]string{{"a", "b"}, {"5", "6"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestXLSXReporterNoSheetsAreSkippedByBadColumns(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewReporter("xlsx", &buf)

	good := ConditionalFormat{
		Test: ConditionalFormatTest{Columns: []string{"v"}, Condition: "=", Value: "x"},
	}
	bad := ConditionalFormat{
		Test: ConditionalFormatTest{Columns: []string{"nosuchcolumn"}, Condition: "=", Value: "x"},
	}
	badSet := ConditionalFormat{
		Test: ConditionalFormatTest{Columns: []string{"v"}, Condition: "=", Value: "x"},
		Set:  []ConditionalFormatSet{{Columns: []string{"nosuchcolumn"}, Format: "ok"}},
	}

	const n = 8
	for i := range n {
		report := Report{Title: "bad" + strconv.Itoa(i)}
		report.XLSX.FreezeColumn = "nosuchcolumn"
		report.XLSX.ConditionalFormat = []ConditionalFormat{bad, badSet}
		_ = r.Prepare(report)
		r.UpdateTable([]string{"k", "v"}, [][]string{{"a", "x"}})

		report = Report{Title: "good" + strconv.Itoa(i)}
		report.XLSX.FreezeColumn = "k"
		report.XLSX.ConditionalFormat = []ConditionalFormat{bad, good}
		_ = r.Prepare(report)
		r.UpdateTable([]string{"k", "v"}, [][]string{{"a", "x"}})
	}

	// a sheet with a conditional format using Rows, but no table at all
	report := Report{Title: "empty"}
	report.XLSX.ConditionalFormat = []ConditionalFormat{{
		Test: ConditionalFormatTest{Columns: []string{"v"}, Condition: "=", Value: "x"},
		Set:  []ConditionalFormatSet{{Rows: "a*"}, {NotRows: "b*"}},
	}}
	_ = r.Prepare(report)

	f := renderXLSX(t, r, &buf)
	for i := range n {
		name := "good" + strconv.Itoa(i)
		cf, err := f.GetConditionalFormats(name)
		if err != nil || len(cf) == 0 {
			t.Errorf("sheet %s: conditional formats = %v, err = %v; want one", name, cf, err)
		}
		panes, err := f.GetPanes(name)
		if err != nil || !panes.Freeze {
			t.Errorf("sheet %s: panes = %+v, err = %v; want frozen", name, panes, err)
		}
	}
}

func TestMinMaxColumnWidthZeroUsesDefault(t *testing.T) {
	o := evalXLSXReportOptions(MinColumnWidth(0), MaxColumnWidth(0))
	if o.minColWidth != 10 || o.maxColWidth != 60 {
		t.Errorf("widths = %v, %v; want the defaults 10, 60", o.minColWidth, o.maxColWidth)
	}
	o = evalXLSXReportOptions(MinColumnWidth(5), MaxColumnWidth(20))
	if o.minColWidth != 5 || o.maxColWidth != 20 {
		t.Errorf("widths = %v, %v; want 5, 20", o.minColWidth, o.maxColWidth)
	}
}
