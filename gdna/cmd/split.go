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

package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
)

// splitTitle returns the title of the report for the split value, which
// is the name of its dataview and is not SQL
func splitTitle(cf *config.Config, report Report, value string, lookup config.ExpandOption) string {
	return config.Expand[string](cf, report.Title, config.LookupTable(map[string]string{
		"split-column": report.SplitColumn,
		"value":        value,
	}), lookup, config.ExpandNonStringToCSV())
}

// resetSplitViews removes any old dataviews of the split report for the
// values
func resetSplitViews(cf *config.Config, r reporter.Reporter, report Report, values []string, lookup config.ExpandOption) {
	for _, v := range values {
		rep := reporter.Report{}
		rep.Title = splitTitle(cf, report, v, lookup)
		rep.Dataview.Group = report.Dataview.Group
		r.Reset(rep)
	}
}

// create a report per gateway (or other column) and populate with given queries
func publishReportSplit(ctx context.Context, cf *config.Config, tx *sql.Tx, r reporter.Reporter, report Report) (err error) {
	split := []string{}
	lookup := config.LookupTable(report.lookupTable())

	if report.Subreport == "" {
		if report.SplitValues == "" {
			return errors.New("no split-values-query defined")
		}

		// get list of split values (typically gateways)
		splitquery := config.Expand[string](cf, report.SplitValues, lookup, config.ExpandNonStringToCSV())
		traceQuery(ctx, splitquery)
		rows, err := tx.QueryContext(ctx, splitquery)
		if err != nil {
			log.Error("failed to execute query", slog.Any("error", err), slog.String("query", splitquery))
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var value string
			if err = rows.Scan(&value); err != nil {
				return err
			}
			split = append(split, value)
		}
		if err = rows.Err(); err != nil {
			return err
		}
		rows.Close()
		slices.Sort(split)

		if _, ok := r.(*reporter.XLSXReporter); ok && report.XLSX.Enable != nil && !*report.XLSX.Enable {
			log.Debug("report disabled for XLSX output, removing any old dataviews", slog.String("report", report.Name))
			resetSplitViews(cf, r, report, split, lookup)
			return nil
		}

		if _, ok := r.(*reporter.APIReporter); ok && report.Dataview.Enable != nil && !*report.Dataview.Enable {
			log.Debug("report disabled for dataview output, removing any old dataviews", slog.String("report", report.Name))
			resetSplitViews(cf, r, report, split, lookup)
			return nil
		}

		// get possible list of all previous views and remove any not in the
		// new list
		previouslist := config.Expand[string](cf, report.SplitValuesAll, lookup, config.ExpandNonStringToCSV())
		if previouslist != "" {
			traceQuery(ctx, previouslist)
			rows, err := tx.QueryContext(ctx, previouslist)
			if err != nil {
				log.Error("failed to execute query", slog.Any("error", err), slog.String("query", previouslist))
				return err
			}
			defer rows.Close()

			previous := []string{}
			for rows.Next() {
				var value string
				if err = rows.Scan(&value); err != nil {
					return err
				}
				previous = append(previous, value)
			}
			if err = rows.Err(); err == nil {
				// no error - process list

				previous = slices.DeleteFunc(previous, func(v string) bool {
					return slices.Contains(split, v)
				})
				resetSplitViews(cf, r, report, previous, lookup)
			}
			rows.Close()
		}
	} else {
		split = []string{report.Subreport}
	}

	var errs []error
	for _, v := range split {
		rep := report // copy, as we update some fields per split

		// the value as is, for the title and file path, and as a SQL
		// string, for the queries
		splitValues := func(value string) map[string]string {
			return map[string]string{
				"split-column": rep.SplitColumn,
				"name":         rep.Name,
				"value":        value,
				"extension":    r.Extension(),
			}
		}
		valueLookup := splitValues(v)
		sqlLookup := splitValues(strings.ReplaceAll(v, "'", "''"))

		origname := rep.Title
		rep.Title = config.Expand[string](cf, rep.Title, config.LookupTable(valueLookup), lookup, config.ExpandNonStringToCSV())
		if rep.FilePath != "" {
			rep.FilePath = config.Expand[string](cf, rep.FilePath, config.LookupTable(valueLookup), lookup, config.ExpandNonStringToCSV())
		} else {
			// generate filepath from rep name
			rep.FilePath = strings.ReplaceAll(rep.Title, " ", "_") + "." + r.Extension()
		}

		if err = r.Prepare(rep.Report); err != nil {
			// do not add anything to the report that was prepared before
			log.Error("failed to prepare report", slog.String("report", rep.Name), slog.String("value", v), slog.Any("error", err))
			errs = append(errs, fmt.Errorf("value %q: cannot prepare report: %w", v, err))
			continue
		}
		if !outputZip {
			r.AddHeadline("reportName", rep.Name)
			if scrambleNames {
				r.AddHeadline("scrambledColumns", strings.Join(report.ScrambleColumns, ","))
			}
		}
		rep.Title = origname

		if err = addHeadlines(ctx, cf, tx, r, rep, config.LookupTable(sqlLookup), lookup); err != nil {
			errs = append(errs, fmt.Errorf("value %q: %w", v, err))
			return errors.Join(errs...)
		}

		query := config.Expand[string](cf, rep.Query, config.LookupTable(sqlLookup), lookup, config.ExpandNonStringToCSV())
		traceQuery(ctx, query)
		t, err := queryToTable(ctx, tx, rep.Columns, query)
		if err != nil {
			errs = append(errs, fmt.Errorf("value %q: %w", v, err))
			return errors.Join(errs...)
		}
		if len(t) > 0 {
			r.UpdateTable(t[0], t[1:])
		}
	}
	return errors.Join(errs...)
}
