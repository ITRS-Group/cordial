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
	"log/slog"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

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
	}

	// import all environment variables into both the template data
	// and also the config structure (config.WithEnvs doesn't work
	// for empty prefixes)
	data.Env = importEnvironment(globalCf, os.Environ())

	varpath := config.Get[string](globalCf, "_variablepath")
	if varpath == "" {
		varpath = buildVarpath(cmd.Root().PersistentFlags().Changed("type"))
	}
	dv, err := xpath.Parse(varpath)
	if err != nil {
		log.Debug("invalid dataview path", slog.Any("error", err), slog.String("path", varpath))
		return
	}
	dv = dv.ResolveTo(&xpath.Dataview{})

	dataviews, err := gw.Match(dv, 0)
	if err != nil {
		log.Debug("matching dataviews failed", slog.Any("error", err), slog.String("path", varpath))
		return
	}

	if len(dataviews) == 0 {
		err = errors.New("no matching dataviews found")
		return
	}

	for _, d := range dataviews {
		dataview, err := getDataview(gw, d, firstcolumn, headlineList, rowList, columnList, rowOrder)
		if err != nil {
			log.Error("failed to get dataview", slog.Any("error", err))
			continue
		}

		data.Dataviews = append(data.Dataviews, dataview)
	}

	return
}

// importEnvironment copies all the environment variables into the
// config structure, so that they can be used for settings and expansion,
// and returns them as a map for use in templates.
//
// Environment variables with the same name (ignoring case) as a
// top-level section or key already in the configuration, e.g. GATEWAY,
// EMAIL, FILES, TEXT or HTML, are not copied into the config as that
// would replace the whole section.
func importEnvironment(cf *config.Config, environ []string) (env map[string]string) {
	env = make(map[string]string, len(environ))

	reserved := map[string]bool{}
	for k := range cf.AllSettings() {
		reserved[strings.ToLower(k)] = true
	}

	for _, e := range environ {
		k, v, found := strings.Cut(e, "=")
		if !found {
			continue
		}
		env[k] = v
		if reserved[strings.ToLower(k)] {
			log.Debug("not importing environment variable over config section", slog.String("name", k))
			continue
		}
		config.Set(cf, k, v)
	}
	return
}

// buildVarpath builds the default XPath to select dataviews from the
// command line arguments. If typeSet is true then the (possibly empty)
// type argument is used to select samplers.
func buildVarpath(typeSet bool) (varpath string) {
	varpath = "//managedEntity"
	if entityArg != "" {
		varpath += fmt.Sprintf("[(@name=%q)]", entityArg)
	}
	varpath += "/sampler"
	if samplerArg != "" {
		if typeSet {
			varpath += fmt.Sprintf("[(@name=%q)][(@type=%q)]", samplerArg, typeArg)
		} else {
			varpath += fmt.Sprintf("[(@name=%q)]", samplerArg)
		}
	}
	varpath += "/dataview"
	if dataviewArg != "" {
		varpath += fmt.Sprintf("[(@name=%q)]", dataviewArg)
	}
	return
}

// entityGroup is the dataviews for a single entity
type entityGroup struct {
	entity    string
	dataviews []*commands.Dataview
}

// groupByEntity splits dataviews by entity name and returns the groups
// in entity name order, with the dataviews in each group in their
// original order
func groupByEntity(dataviews []*commands.Dataview) (groups []entityGroup) {
	byName := map[string][]*commands.Dataview{}
	for _, d := range dataviews {
		byName[d.XPath.Entity.Name] = append(byName[d.XPath.Entity.Name], d)
	}
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		groups = append(groups, entityGroup{entity: name, dataviews: byName[name]})
	}
	return
}

// matchesAny returns true if name matches any of the glob patterns. The
// patterns have surrounding white space removed. Invalid patterns are
// logged and never match.
func matchesAny(patterns []string, name, what string) bool {
	return slices.ContainsFunc(patterns, func(p string) bool {
		p = strings.TrimSpace(p)
		ok, err := path.Match(p, name)
		if err != nil {
			log.Error("invalid "+what, slog.Any("error", err), slog.String("filter", p))
			return false
		}
		return ok
	})
}

// validPatterns returns the patterns that are valid globs. If none are
// valid the result is empty and so no filtering should be done.
func validPatterns(patterns []string, what string) (valid []string) {
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if _, err := path.Match(p, ""); err != nil {
			log.Error("invalid "+what, slog.Any("error", err), slog.String("filter", p))
			continue
		}
		valid = append(valid, p)
	}
	return
}

// getDataview fetches a dataview and applies filters and ordering based
// on the configuration. It returns the resulting dataview or an error
// if the fetch fails.
func getDataview(gw *commands.Connection, dv *xpath.XPath, firstcolumn, headlineList, rowList, columnList, rowOrder string) (dataview *commands.Dataview, err error) {
	dataview, err = gw.Snapshot(dv, "", commands.Scope{Value: true, Severity: true, Snooze: true, UserAssignment: true})
	if err != nil {
		log.Error("failed to snapshot dataview", slog.Any("error", err))
		return
	}

	filterDataview(dataview, firstcolumn, headlineList, rowList, columnList, rowOrder)
	return
}

// filterDataview applies the configured filters and ordering to a
// dataview. Each filter keeps an item if it matches any of the
// patterns that apply to the dataview. The first column, the row name
// column, is never filtered or re-ordered.
func filterDataview(dataview *commands.Dataview, firstcolumn, headlineList, rowList, columnList, rowOrder string) {
	// first, filter headlineFilter. There is no ordering as they are always
	// displayed in alphabetical order in the Active Console
	headlineFilter := validPatterns(match(dataview.Name, "headline-filter", headlineList), "headline filter")
	if len(headlineFilter) > 0 && len(dataview.Headlines) > 0 {
		// remove headlines that do not match any of the filters
		maps.DeleteFunc(dataview.Headlines, func(k string, _ commands.DataItem) bool {
			return !matchesAny(headlineFilter, k, "headline filter")
		})
		dataview.HeadlineOrder = slices.DeleteFunc(dataview.HeadlineOrder, func(k string) bool {
			_, ok := dataview.Headlines[k]
			return !ok
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

	cols := validPatterns(match(dataview.Name, "column-filter", columnList), "column filter")
	if len(cols) > 0 && len(dataview.ColumnOrder) > 0 {
		// remove columns that do not match any of the filters, but
		// always keep the first column
		first := dataview.ColumnOrder[0]
		dataview.ColumnOrder = append([]string{first},
			slices.DeleteFunc(slices.Clone(dataview.ColumnOrder[1:]), func(c string) bool {
				return !matchesAny(cols, c, "column filter")
			})...)
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
	if len(matches) > 0 && len(dataview.ColumnOrder) > 1 {
		rest := dataview.ColumnOrder[1:]
		if len(matches) == 1 && (strings.EqualFold(matches[0], "asc") || strings.EqualFold(matches[0], "desc")) {
			slices.Sort(rest)
			if strings.EqualFold(matches[0], "desc") {
				slices.Reverse(rest)
			}
		} else {
			nc := []string{dataview.ColumnOrder[0]}
			// remaining columns are those not yet selected by an earlier pattern
			remaining := slices.Clone(rest)
			for _, m := range validPatterns(matches, "column order filter") {
				var next []string
				for _, c := range remaining {
					if ok, _ := path.Match(m, c); ok {
						nc = append(nc, c)
					} else {
						next = append(next, c)
					}
				}
				remaining = next
			}
			// now add back any leftovers
			dataview.ColumnOrder = append(nc, remaining...)
		}
	}

	rows := validPatterns(match(dataview.Name, "row-filter", rowList), "row filter")
	if len(rows) > 0 {
		keep := func(k string) bool { return matchesAny(rows, k, "row filter") }
		maps.DeleteFunc(dataview.Table, func(k string, _ map[string]commands.DataItem) bool {
			return !keep(k)
		})
		dataview.RowOrder = slices.DeleteFunc(dataview.RowOrder, func(k string) bool {
			return !keep(k)
		})
	}

	// order rows based on the value in a column, or based on the row
	// name if he column is the literal `rowname`
	matches = match(dataview.Name, "row-order", rowOrder)
	if len(matches) > 0 && len(dataview.RowOrder) > 0 && len(dataview.ColumnOrder) > 0 {
		colname := strings.TrimSpace(matches[0])
		asc := true
		switch {
		case strings.HasSuffix(colname, "-"):
			asc = false
			colname = strings.TrimSuffix(colname, "-")
		case strings.HasSuffix(colname, "+"):
			colname = strings.TrimSuffix(colname, "+")
		}

		// if the row-order is for a column that is used as the
		// rowname (decided above in Column ordering) then sort
		// the data.Rows slice directly based on value and
		// not a cell in the row
		if colname == "rowname" || colname == dataview.ColumnOrder[0] {
			slices.SortStableFunc(dataview.RowOrder, natCompare) // natural sort
			if !asc {
				slices.Reverse(dataview.RowOrder)
			}
		} else {
			// indirect sort of rownames based on the values in a given column (that isn't rowname)
			slices.SortStableFunc(dataview.RowOrder, func(x, y string) int {
				a := dataview.Table[x][colname].Value
				b := dataview.Table[y][colname].Value
				if !asc {
					a, b = b, a
				}
				return natCompare(a, b)
			})
		}
	}
}

// natCompare compares a and b in natural sort order for use with
// [slices.SortFunc]
func natCompare(a, b string) int {
	switch {
	case Less(a, b):
		return -1
	case Less(b, a):
		return 1
	}
	return 0
}

// buildName returns in as name unless in is "auto" in which case it
// applies some heuristics to the name. It builds a name based on
// `serial`, `entity`, `sampler` and `dataview` from the lookup map,
// removes empty values and adjacent duplicates, joins the remaining
// components with a `-` and then appends a `-` and the `timestamp`
// value, if set. If the resulting name (or the input string `in`) is
// empty then the function returns value of the "default" key in lookup
// or, if that is empty then "undefined". The timestamp is only added
// for "auto" names, any other name is used as given.
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
	if in == "auto" {
		if t := lookup["timestamp"]; t != "" {
			name += "-" + t
		}
	}
	return
}
