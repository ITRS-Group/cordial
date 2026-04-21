/*
Copyright © 2023 ITRS Group

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

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos/commands"
	"github.com/itrs-group/cordial/pkg/geneos/xpath"
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
		config.Set(cf, n[0], n[1])
	}

	varpath := config.Get[string](cf, "_variablepath")
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

// getDataview fetches a dataview and applies filters and ordering based
// on the configuration. It returns the resulting dataview or an error
// if the fetch fails.
func getDataview(gw *commands.Connection, dv *xpath.XPath, firstcolumn, headlineList, rowList, columnList, rowOrder string) (dataview *commands.Dataview, err error) {
	dataview, err = gw.Snapshot(dv, "", commands.Scope{Value: true, Severity: true})
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	// first, filter headlineFilter. There is no ordering as they are always
	// displayed in alphabetical order in the Active Console
	headlineFilter := match(dataview.Name, "headline-filter", headlineList)
	if len(headlineFilter) > 0 && len(dataview.Headlines) > 0 {
		// remove headlines that do not match the filters
		maps.DeleteFunc(dataview.Headlines, func(k string, _ commands.DataItem) bool {
			return slices.ContainsFunc(headlineFilter, func(h string) bool {
				h = strings.TrimSpace(h)
				ok, err := path.Match(h, k)
				if err != nil {
					log.Error().Err(err).Msgf("invalid headline filter %q", h)
					// if the filter is invalid then do not filter out
					// the headline based on that filter
					return false
				}
				return !ok
			})
		})
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
		// remove columns that do not match the filters
		dataview.ColumnOrder = slices.DeleteFunc(dataview.ColumnOrder, func(c string) bool {
			c = strings.TrimSpace(c)
			return slices.ContainsFunc(cols, func(col string) bool {
				ok, err := path.Match(col, c)
				if err != nil {
					log.Error().Err(err).Msgf("invalid column filter %q", col)
					// if the filter is invalid then do not filter out
					// the column based on that filter
					return false
				}
				return !ok
			})
		})
	}

	// order the resulting columns (after filters above) by their names.
	// this doesn't filter columns and so any not listed will be left in
	// their existing order but after those explicitly selected. the
	// first column, the row name, is always first and cannot be sorted
	// with the others.
	//
	// as a special case, if there is a single column-order value and it
	// is "asc" or "desc" then all column names are sorted in ascending
	// or descending order respectively.
	matches := match(dataview.Name, "column-order", "")
	if len(matches) == 1 && (strings.EqualFold(matches[0], "asc") || strings.EqualFold(matches[0], "desc")) {
		slices.Sort(dataview.ColumnOrder[1:])
		if strings.EqualFold(matches[0], "desc") {
			slices.Reverse(dataview.ColumnOrder[1:])
		}
	} else if len(matches) > 0 {
		// remove columns from oc as we match them
		oc := slices.Clone(dataview.ColumnOrder)

		nc := make([]string, 0, len(dataview.ColumnOrder))
		for _, m := range matches {
			m = strings.TrimSpace(m)
			for i, c := range oc {
				ok, err := path.Match(m, c)
				if err != nil {
					log.Error().Err(err).Msgf("invalid column order filter %q", m)
					continue
				}
				if ok {
					oc = slices.Delete(oc, i, i+1)
					nc = append(nc, c)
				}
			}
		}
		// now add back any leftovers
		nc = append(nc, oc...)
		dataview.ColumnOrder = nc
	}

	rows := match(dataview.Name, "row-filter", rowList)
	if len(rows) > 0 && len(dataview.Table) > 0 {
		maps.DeleteFunc(dataview.Table, func(k string, _ map[string]commands.DataItem) bool {
			return slices.ContainsFunc(rows, func(r string) bool {
				r = strings.TrimSpace(r)
				ok, err := path.Match(r, k)
				if err != nil {
					log.Error().Err(err).Msgf("invalid row filter %q", r)
					// if the filter is invalid then do not filter out
					// the row based on that filter
					return false
				}
				return !ok
			})
		})
	}

	// order rows based on the value in a column, or based on the row
	// name if he column is the literal `rowname`
	asc := true
	matches = match(dataview.Name, "row-order", rowOrder)
	if len(matches) > 0 && len(dataview.Table) > 0 {
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
