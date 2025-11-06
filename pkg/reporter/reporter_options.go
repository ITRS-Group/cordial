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

type ReporterOptions func(*reporterOptions)

type reporterOptions struct {
	scrambleNames   bool
	scrambleFunc    func(string) string
	scrambleColumns []string
}

func evalReporterOptions(options ...ReporterOptions) (ro *reporterOptions) {
	ro = &reporterOptions{
		scrambleFunc: scrambleWords,
	}
	for _, opt := range options {
		opt(ro)
	}
	return
}

func Scramble(scramble bool) ReporterOptions {
	return func(ro *reporterOptions) {
		ro.scrambleNames = scramble
	}
}

func ScrambleFunc(fn func(in string) string) ReporterOptions {
	return func(ro *reporterOptions) {
		ro.scrambleFunc = fn
	}
}

func ScrambleColumns(columns []string) ReporterOptions {
	return func(ro *reporterOptions) {
		ro.scrambleColumns = columns
	}
}
