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

type Reporter interface {
	SetReport(report Report) error
	WriteHeadline(name, value string)
	WriteTable(rows ...[]string)
	Render()
	Close()
}

type Report struct {
	Name              string   `mapstructure:"name"`
	Group             string   `mapstructure:"group,omitempty"`
	Columns           []string `mapstructure:"columns,omitempty"`
	EnableForDataview *bool    `mapstructure:"enable-for-dataview,omitempty"`
	EnableForXLSX     *bool    `mapstructure:"enable-for-xlsx,omitempty"`
	FreezeColumn      string   `mapstructure:"freeze-to-column"`
	ScrambleColumns   []string `mapstructure:"scramble-columns,omitempty"`
	Type              string   `mapstructure:"type,omitempty"`
	Query             string   `mapstructure:"query,omitempty"`
	Headlines         string   `mapstructure:"headlines,omitempty"`

	Grouping      string   `mapstructure:"grouping,omitempty"`
	GroupingOrder []string `mapstructure:"grouping-order,omitempty"`

	ConditionalFormat []ConditionalFormat `mapstructure:"conditional-format,omitempty"`

	// when Type = "split" then
	SplitColumn string `mapstructure:"split-column,omitempty"`
	SplitValues string `mapstructure:"split-values-query,omitempty"`
}

type ReportOptions func(*reportOptions)

type reportOptions struct {
	scrambleColumns []string
}

func evalReportOptions(options ...ReportOptions) (ro *reportOptions) {
	ro = &reportOptions{}
	for _, opt := range options {
		opt(ro)
	}
	return
}

func ScrambleColumns(columns []string) ReportOptions {
	return func(ro *reportOptions) {
		ro.scrambleColumns = columns
	}
}
