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
	"archive/zip"
	"io"
)

type formattedReporterOptions struct {
	writer           io.Writer
	zipWriter        *zip.Writer
	dvcssclass       string
	headlinecssclass string
	htmlpreamble     string
	htmlpostscript   string
	orderbycolumns   []int
}

func evalFormattedOptions(options ...FormattedReporterOption) (fro *formattedReporterOptions) {
	fro = &formattedReporterOptions{
		dvcssclass:       "table",
		headlinecssclass: "headlines",
		// orderbycolumns:   []int{0},
	}
	for _, opt := range options {
		opt(fro)
	}
	return
}

// FormattedReporterOption TODO: Describe its purpose and intended use.
type FormattedReporterOption func(*formattedReporterOptions)

// Writer TODO: Document its behavior, inputs, and effects.
func Writer(w io.Writer) FormattedReporterOption {
	return func(fro *formattedReporterOptions) {
		fro.writer = w
	}
}

// ZipWriter TODO: Document its behavior, inputs, and effects.
func ZipWriter(z *zip.Writer) FormattedReporterOption {
	return func(fro *formattedReporterOptions) {
		fro.zipWriter = z
	}
}

// DataviewCSSClass TODO: Document its behavior, inputs, and effects.
func DataviewCSSClass(cssclass string) FormattedReporterOption {
	return func(fro *formattedReporterOptions) {
		fro.dvcssclass = cssclass
	}
}

// HeadlineCSSClass TODO: Document its behavior, inputs, and effects.
func HeadlineCSSClass(cssclass string) FormattedReporterOption {
	return func(fro *formattedReporterOptions) {
		fro.headlinecssclass = cssclass
	}
}

// HTMLPreamble TODO: Document its behavior, inputs, and effects.
func HTMLPreamble(preamble string) FormattedReporterOption {
	return func(fro *formattedReporterOptions) {
		fro.htmlpreamble = preamble
	}
}

// HTMLPostscript TODO: Document its behavior, inputs, and effects.
func HTMLPostscript(postscript string) FormattedReporterOption {
	return func(fro *formattedReporterOptions) {
		fro.htmlpostscript = postscript
	}
}

// OrderByColumns sets the column indices to order the table by, in
// order of precedence. The default is to order by the first column
// only.
func OrderByColumns(cols ...int) FormattedReporterOption {
	return func(fro *formattedReporterOptions) {
		fro.orderbycolumns = cols
	}
}
