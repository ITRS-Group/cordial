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
	"slices"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
)

// create a report per gateway (or other column) and populate with given queries
func publishReportSplit(ctx context.Context, cf *config.Config, tx *sql.Tx, r reporter.Reporter, report Report) (err error) {
	split := []string{}
	lookup := config.LookupTable(reportLookupTable(report.Dataview.Group, report.Title, scrambleNames))

	if report.SplitValues == "" {
		log.Error().Msg("no split-values-query defined")
		return
	}

	if report.Subreport == "" {
		// get list of split values (typically gateways)
		splitquery := cf.ExpandString(report.SplitValues, lookup, config.ExpandNonStringToCSV())
		log.Trace().Msgf("query:\n%s", splitquery)
		rows, err := tx.QueryContext(ctx, splitquery)
		if err != nil {
			log.Error().Err(err).Msgf("query: %s", splitquery)
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
			log.Debug().Msgf("report %s disabled for XLSX output, removing any old dataviews", report.Name)
			for _, p := range split {
				group := report.Dataview.Group
				title := cf.ExpandString(report.Title, config.LookupTable(map[string]string{
					"split-column": report.SplitColumn,
					"value":        p,
				}), lookup, config.ExpandNonStringToCSV())
				rep := reporter.Report{}
				rep.Title = title
				rep.Dataview.Group = group
				r.Remove(rep)
			}
			return err
		}

		if _, ok := r.(*reporter.APIReporter); ok && report.Dataview.Enable != nil && !*report.Dataview.Enable {
			log.Debug().Msgf("report %s disabled for dataview output, removing any old dataviews", report.Name)
			for _, p := range split {
				group := report.Dataview.Group
				title := cf.ExpandString(report.Title, config.LookupTable(map[string]string{
					"split-column": report.SplitColumn,
					"value":        p,
				}), lookup, config.ExpandNonStringToCSV())
				rep := reporter.Report{}
				rep.Title = title
				rep.Dataview.Group = group
				r.Remove(rep)
			}
			return err
		}

		// get possible list of all previous views and remove any not in the
		// new list
		previouslist := cf.ExpandString(report.SplitValuesAll, lookup, config.ExpandNonStringToCSV())
		if previouslist != "" {
			log.Trace().Msgf("query:\n%s", previouslist)
			rows, err := tx.QueryContext(ctx, previouslist)
			if err != nil {
				log.Error().Err(err).Msgf("query: %s", previouslist)
				return err
			}

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
				for _, p := range previous {
					group := report.Dataview.Group
					title := cf.ExpandString(report.Title, config.LookupTable(map[string]string{
						"split-column": report.SplitColumn,
						"value":        p,
					}), lookup, config.ExpandNonStringToCSV())
					rep := reporter.Report{}
					rep.Title = title
					rep.Dataview.Group = group
					r.Remove(rep)
				}
			}
			rows.Close()
		}
	} else {
		split = []string{report.Subreport}
	}

	for _, v := range split {
		rep := report // copy, as we update some fields per split

		v = strings.ReplaceAll(v, "'", "''")
		split := map[string]string{
			"split-column": rep.SplitColumn,
			"name":         rep.Name,
			"value":        v,
			"extension":    r.Extension(),
		}
		origname := rep.Title
		rep.Title = cf.ExpandString(rep.Title, config.LookupTable(split), lookup, config.ExpandNonStringToCSV())
		if rep.FilePath != "" {
			rep.FilePath = cf.ExpandString(rep.FilePath, config.LookupTable(split), lookup, config.ExpandNonStringToCSV())
		} else {
			// generate filepath from rep name
			rep.FilePath = strings.ReplaceAll(rep.Title, " ", "_") + "." + r.Extension()
		}

		if err = r.Prepare(rep.Report); err != nil {
			log.Debug().Err(err).Msg("")
		}
		if !outputZip {
			r.AddHeadline("reportName", rep.Name)
			if scrambleNames {
				r.AddHeadline("scrambledColumns", strings.Join(report.ScrambleColumns, ","))
			}
		}
		rep.Title = origname

		if query := cf.ExpandString(rep.Headlines, config.LookupTable(split), lookup, config.ExpandNonStringToCSV()); query != "" {
			names, headlines, err := queryHeadlines(ctx, tx, query)
			if err != nil {
				log.Error().Msgf("failed to execute headline query: %s\n%s", err, query)
				return err
			}
			for _, h := range names {
				r.AddHeadline(h, headlines[h])
			}
		}

		query := cf.ExpandString(rep.Query, config.LookupTable(split), lookup, config.ExpandNonStringToCSV())
		log.Trace().Msgf("query:\n%s ->\n%s", rep.Query, query)
		t, err := queryToTable(ctx, tx, rep.Columns, query)
		if err != nil {
			return err
		}
		if len(t) > 0 {
			r.UpdateTable(t[0], t[1:])
		}
	}
	return
}
