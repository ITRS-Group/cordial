/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/xpath"
)

// dataFile is a name and content as a buffer, returned from various builders
type dataFile struct {
	name    string
	content *bytes.Reader
}

func fetchDataviews(cmd *cobra.Command, gw *commands.Connection, firstcolumn, headlineList, rowList, columnList, rowOrder string) (data DV2EMailData, err error) {
	data = DV2EMailData{
		Dataviews: []*commands.Dataview{},
		Env:       make(map[string]string, len(os.Environ())),
	}

	// import all environment variables into both the template data
	// and also the config structure (config.WithEnvs doesn't work
	// for empty prefixes)
	for _, e := range os.Environ() {
		n := strings.SplitN(e, "=", 2)
		data.Env[n[0]] = n[1]
		cf.Set(n[0], n[1])
	}

	varpath := cf.GetString("_variablepath")
	if varpath == "" {
		varpath = "//managedEntity"
		if entityArg != "" {
			varpath += fmt.Sprintf("[(@name=%q)]", entityArg)
		}
		varpath += "/sampler"
		if samplerArg != "" {
			if cmd.Root().PersistentFlags().Changed("type") {
				varpath += fmt.Sprintf("[(@name=%q)][(@type=%q)]", samplerArg, typeArg)
			} else {
				varpath += fmt.Sprintf("[(@name=%q)]", samplerArg)

			}
		}
		varpath += "/dataview"
		if dataviewArg != "" {
			varpath += fmt.Sprintf("[(@name=%q)]", dataviewArg)
		}
	}
	dv, err := xpath.Parse(varpath)
	if err != nil {
		return
	}
	dv = dv.ResolveTo(&xpath.Dataview{})

	dataviews, err := gw.Match(dv, 0)
	if err != nil {
		return
	}

	if len(dataviews) == 0 {
		err = errors.New("no matching dataviews found")
		return
	}

	for _, d := range dataviews {
		dataview, err := getDataview(gw, d, firstcolumn, headlineList, rowList, columnList, rowOrder)
		if err != nil {
			log.Error().Err(err).Msg("")
			continue
		}

		data.Dataviews = append(data.Dataviews, dataview)
	}

	return
}

func getDataview(gw *commands.Connection, dv *xpath.XPath, firstcolumn, headlineList, rowList, columnList, rowOrder string) (dataview *commands.Dataview, err error) {
	dataview, err = gw.Snapshot(dv, "", commands.Scope{Value: true, Severity: true})
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	// filter here

	headlines := match(dataview.Name, "headline-filter", headlineList)
	if len(headlines) > 0 {
		nh := map[string]commands.DataItem{}
		for _, h := range headlines {
			h = strings.TrimSpace(h)
			for oh, headline := range dataview.Headlines {
				if ok, err := path.Match(h, oh); err == nil && ok {
					nh[oh] = headline
				}
			}
		}
		dataview.Headlines = nh
	}

	// the first column is either from `first-column` in config
	// (matched against the dataview name) or from the
	// environment variable _FIRSTCOLUMN or `rowname` and is
	// always the actual first column.
	var rowname string
	defaultRowName := match(dataview.Name, "first-column", firstcolumn)
	if len(defaultRowName) > 0 {
		rowname = defaultRowName[0]
	} else {
		rowname = "rowname"
	}

	// set the default, may be overridden below but then reset
	// to the same value
	if len(dataview.ColumnOrder) > 0 {
		dataview.ColumnOrder[0] = rowname
	}

	cols := match(dataview.Name, "column-filter", columnList)
	if len(cols) > 0 {
		nc := []string{rowname}
		for _, c := range cols {
			c = strings.TrimSpace(c)
			for _, oc := range dataview.ColumnOrder {
				if oc == rowname {
					continue
				}
				if ok, err := path.Match(c, oc); err == nil && ok {
					nc = append(nc, oc)
				}
			}
		}

		dataview.ColumnOrder = nc
	} else {
		matches := match(dataview.Name, "column-order", "")
		if len(matches) > 0 {
			m := matches[0]
			switch {
			case strings.HasPrefix(m, "desc"):
				slices.Sort(dataview.ColumnOrder[1:])
				slices.Reverse(dataview.ColumnOrder[1:])
			case strings.HasPrefix(m, "asc"):
				slices.Sort(dataview.ColumnOrder[1:])
			}
		}
	}

	rows := match(dataview.Name, "row-filter", rowList)
	if len(rows) > 0 {
		nr := map[string]map[string]commands.DataItem{}
		for _, r := range rows {
			r = strings.TrimSpace(r)
			for rowname, row := range dataview.Table {
				if ok, err := path.Match(r, rowname); err == nil && ok {
					nr[rowname] = row
				}
			}
		}
		dataview.Table = nr
	}

	asc := true
	matches := match(dataview.Name, "row-order", rowOrder)
	if len(matches) > 0 {
		colname := matches[0]
		switch {
		case strings.HasSuffix(colname, "-"):
			asc = false
			colname = colname[:len(colname)-1]
		case strings.HasSuffix(colname, "+"):
			colname = colname[:len(colname)-1]
			fallthrough
		default:
			asc = true
		}

		// if the row-order is for a column that is used as the
		// rowname (decided above in Column ordering) then sort
		// the data.Rows slice directly based on value and
		// not a cell in the row
		if colname == "rowname" || colname == dataview.ColumnOrder[0] {
			sort.Sort(NatsortStringSlice(dataview.RowOrder)) // natural sort
			if !asc {
				slices.Reverse(dataview.RowOrder)
			}
		} else {
			// indirect sort of rownames based on the values in a given column (that isn't rowname)
			r := dataview.RowOrder
			sort.Slice(r, func(i, j int) bool {
				a := dataview.Table[r[i]][colname].Value
				b := dataview.Table[r[j]][colname].Value
				if asc {
					return Less(a, b)
				} else {
					return Less(b, a)
				}
			})
		}
	}

	return
}

// buildName returns in as name unless in is "auto" in which case it
// applies some heuristics to the name. It builds a name based on
// `serial`, `entity`, `sampler` and `dataview` and `timestamp`from the
// lookup map, removes empty values and adjacent duplicates and then
// joins the remaining components with a `-`. If the resulting name (or
// the input string `in`) is empty then the function returns value of
// the "default" key in lookup or, if that is empty then "undefined"
func buildName(in string, lookup map[string]string) (name string) {
	name = in
	if in == "auto" {
		// build the slice of naming parts, remove empty strings and
		// then remove adjacent duplicates
		parts := []string{
			lookup["serial"],
			lookup["entity"],
			lookup["sampler"],
			lookup["dataview"],
			// lookup["timestamp"],
		}
		parts = slices.DeleteFunc(parts, func(s string) bool {
			return s == ""
		})
		parts = slices.Compact(parts)

		name = strings.Join(parts, "-")
	}
	if name == "" {
		name = lookup["default"]
		if name == "" {
			name = "undefined"
		}
	}
	if t, ok := lookup["timestamp"]; ok {
		name += "-" + t
	}
	return
}
