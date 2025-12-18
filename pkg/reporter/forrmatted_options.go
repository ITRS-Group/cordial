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
	renderas         string
	dvcssclass       string
	headlinecssclass string
	htmlpreamble     string
	htmlpostscript   string
	orderbycolumns   []int
}

func evalFormattedOptions(options ...FormattedReporterOptions) (fro *formattedReporterOptions) {
	fro = &formattedReporterOptions{
		renderas:         "table",
		dvcssclass:       "table",
		headlinecssclass: "headlines",
		orderbycolumns:   []int{0},
	}
	for _, opt := range options {
		opt(fro)
	}
	return
}

type FormattedReporterOptions func(*formattedReporterOptions)

func Writer(w io.Writer) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.writer = w
	}
}

func ZipWriter(z *zip.Writer) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.zipWriter = z
	}
}

func RenderAs(renderas string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.renderas = renderas
	}
}

func DataviewCSSClass(cssclass string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.dvcssclass = cssclass
	}
}

func HeadlineCSSClass(cssclass string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.headlinecssclass = cssclass
	}
}

func HTMLPreamble(preamble string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.htmlpreamble = preamble
	}
}

func HTMLPostscript(postscript string) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.htmlpostscript = postscript
	}
}

func OrderByColumns(cols ...int) FormattedReporterOptions {
	return func(fro *formattedReporterOptions) {
		fro.orderbycolumns = cols
	}
}
