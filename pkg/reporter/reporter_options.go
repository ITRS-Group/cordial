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

// The reporter package provides a simple interface to generating Geneos
// dataviews, with headlines and a data table, either though the XML-RPC
// API, as Toolkit compatible CSV, XLSX workbooks or a number of other
// formats.
package reporter

// ReporterOption TODO: Describe its purpose and intended use.
type ReporterOption func(*reporterOptions)

type reporterOptions struct {
	scrambleNames   bool
	scrambleFunc    func(string) string
	scrambleColumns []string
}

func evalReporterOptions(options ...ReporterOption) (ro *reporterOptions) {
	ro = &reporterOptions{
		scrambleFunc: scrambleWords,
	}
	for _, opt := range options {
		opt(ro)
	}
	return
}

// Scramble enables scrambling of the values in the columns named in
// the ScrambleColumns of each report, and any given to
// [ScrambleColumns]. The caller's data is not modified.
func Scramble(scramble bool) ReporterOption {
	return func(ro *reporterOptions) {
		ro.scrambleNames = scramble
	}
}

// ScrambleFunc sets the function used to scramble values. The default
// replaces each word with a truncated hash.
func ScrambleFunc(fn func(in string) string) ReporterOption {
	return func(ro *reporterOptions) {
		ro.scrambleFunc = fn
	}
}

// ScrambleColumns sets column names to scramble in every report, in
// addition to those in the ScrambleColumns of each report.
func ScrambleColumns(columns []string) ReporterOption {
	return func(ro *reporterOptions) {
		ro.scrambleColumns = columns
	}
}
