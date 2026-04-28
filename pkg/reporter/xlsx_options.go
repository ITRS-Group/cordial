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
	"github.com/itrs-group/cordial/pkg/config"
)

type XLSXReporterOption func(*xlsxReportOptions)

type xlsxReportOptions struct {
	scramble         bool
	password         *config.Secret
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

func evalXLSXReportOptions(options ...XLSXReporterOption) (xo *xlsxReportOptions) {
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

func XLSXScramble(scramble bool) XLSXReporterOption {
	return func(xro *xlsxReportOptions) {
		xro.scramble = scramble
	}
}

// XLSXPassword sets the workbook password
func XLSXPassword(password *config.Secret) XLSXReporterOption {
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
func XLSXHeadlines(headlines int) XLSXReporterOption {
	return func(xro *xlsxReportOptions) {
		xro.headlines = headlines
	}
}

func SummarySheetName(name string) XLSXReporterOption {
	return func(xro *xlsxReportOptions) {
		xro.summarySheetName = name
	}
}

func DateFormat(dateFormat string) XLSXReporterOption {
	return func(xro *xlsxReportOptions) {
		xro.dateFormat = dateFormat
	}
}

func IntFormat(format int) XLSXReporterOption {
	return func(xro *xlsxReportOptions) {
		xro.intFormat = format
	}
}

func PercentFormat(format int) XLSXReporterOption {
	return func(xro *xlsxReportOptions) {
		xro.percentFormat = format
	}
}

func SeverityColours(undefined, ok, warning, critical string) XLSXReporterOption {
	return func(xro *xlsxReportOptions) {
		xro.undefinedColour = undefined
		xro.okColour = ok
		xro.warningColour = warning
		xro.criticalColour = critical
	}
}

func MinColumnWidth(n float64) XLSXReporterOption {
	return func(xro *xlsxReportOptions) {
		xro.minColWidth = n
	}
}

func MaxColumnWidth(n float64) XLSXReporterOption {
	return func(xro *xlsxReportOptions) {
		xro.maxColWidth = n
	}
}
