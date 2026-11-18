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
	"archive/zip"
	"encoding/csv"
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
)

// outputCSVZip writes the slice of Entity structs to a zip files
//
// the default files are:
//
// - entities.csv - all entities, their probes and a variable number of
// attributes where the column names are the attributes in alphabetical
// order. Also, one column per plugin type (not sampler) and total of instances
// - file plugins (fkm, ftm, stateTracker)
// - processes.csv
func outputCSVZip(cf *config.Config, gateway string, Entities []Entity, probes map[string]geneos.Probe) (err error) {
	dir := config.Get[string](cf, "output.directory")
	_ = os.MkdirAll(dir, 0775)

	conftable := config.LookupTable(map[string]string{
		"gateway":  gateway,
		"datetime": startTimestamp,
	})

	filename := config.Get[string](cf, "output.formats.csv", conftable)
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(dir, filename)
	}

	zipfile, err := os.Create(filename)
	if err != nil {
		return
	}
	z := zip.NewWriter(zipfile)
	defer func() {
		// report close errors, e.g. for a truncated or failed write
		if cerr := z.Close(); err == nil {
			err = cerr
		}
		if cerr := zipfile.Close(); err == nil {
			err = cerr
		}
	}()

	// output a summary file
	w, err := createCSVZip(cf, z, config.Get[string](cf, "output.reports.summary.filename", conftable)+".csv")
	if err != nil {
		log.Error("failed to create CSV zip", slog.Any("error", err))
		return
	}
	if err = outputCSVSummary(w, cf, gateway, Entities, probes); err != nil {
		return
	}

	// entities.csv
	w, err = createCSVZip(cf, z, config.Get[string](cf, "output.reports.entities.filename", conftable)+".csv")
	if err != nil {
		log.Error("failed to create CSV zip", slog.Any("error", err))
		return
	}
	if err = outputCVSEntities(w, Entities, cf, conftable); err != nil {
		return
	}

	for _, plugin := range config.Get[[]string](cf, "output.plugins.single-column") {
		rows := 0
		for _, e := range Entities {
			for _, s := range e.Samplers {
				if s.Plugin == plugin {
					rows += len(s.Column1)
				}
			}
		}
		if rows == 0 && config.Get[bool](cf, "output.skip-empty-reports") {
			continue
		}

		w, err = createCSVZip(cf, z, config.Get[string](cf, config.Join("output", "reports", plugin, "filename"), conftable)+".csv")
		if err != nil {
			return
		}
		if err = outputCSVSinglePlugin(w, Entities, cf, conftable, plugin); err != nil {
			return
		}
	}

	for _, plugin := range config.Get[[]string](cf, "output.plugins.two-column") {
		rows := 0
		for _, e := range Entities {
			for _, s := range e.Samplers {
				if s.Plugin == plugin {
					rows += len(s.Column1) + len(s.Column2)
				}
			}
		}
		if rows == 0 && config.Get[bool](cf, "output.skip-empty-reports") {
			continue
		}

		w, err = createCSVZip(cf, z, config.Get[string](cf, config.Join("output", "reports", plugin, "filename"), conftable)+".csv")
		if err != nil {
			return
		}
		if err = outputCSVTwoColumnPlugin(w, Entities, cf, conftable, plugin); err != nil {
			return
		}
	}

	return
}

func createCSVZip(cf *config.Config, z *zip.Writer, filename string) (w io.Writer, err error) {
	return z.Create(filename)
}

type csvFiles struct {
	path      string
	filename  string
	sheetname string
}

// writeCSVFile creates filename in dir, calls write and returns the
// first error from either the write or closing the file
func writeCSVFile(dir, filename string, write func(w io.Writer) error) (err error) {
	f, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return
	}
	err = write(f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return
}

func outputCSVDir(cf *config.Config, gateway string, Entities []Entity, probes map[string]geneos.Probe) (csvfiles []csvFiles, subdir string, err error) {
	conftable := config.LookupTable(map[string]string{
		"gateway":  gateway,
		"datetime": startTimestamp,
	})

	dir := config.Get[string](cf, "output.directory")
	subdir = config.Get[string](cf, "output.formats.csvdir", conftable)
	if !filepath.IsAbs(subdir) {
		subdir = filepath.Join(dir, subdir)
	}
	if err = os.MkdirAll(subdir, 0775); err != nil {
		return
	}

	// output a summary file
	summaryFile := config.Get[string](cf, "output.reports.summary.filename", conftable) + ".csv"
	err = writeCSVFile(subdir, summaryFile, func(w io.Writer) error {
		return outputCSVSummary(w, cf, gateway, Entities, probes)
	})
	if err != nil {
		log.Error("failed to create CSV file", slog.Any("error", err))
		return
	}

	csvfiles = append(csvfiles, csvFiles{
		path:      filepath.Join(subdir, summaryFile),
		filename:  config.Get[string](cf, "output.reports.summary.filename", config.NoExpand()),
		sheetname: config.Get[string](cf, "output.reports.summary.sheetname", config.NoExpand()),
	})

	// entities.csv
	entitiesFile := config.Get[string](cf, "output.reports.entities.filename", conftable) + ".csv"
	err = writeCSVFile(subdir, entitiesFile, func(w io.Writer) error {
		return outputCVSEntities(w, Entities, cf, conftable)
	})
	if err != nil {
		log.Error("failed to create CSV file", slog.Any("error", err))
		return
	}

	csvfiles = append(csvfiles, csvFiles{
		path:      filepath.Join(subdir, entitiesFile),
		filename:  config.Get[string](cf, "output.reports.entities.filename", config.NoExpand()),
		sheetname: config.Get[string](cf, "output.reports.entities.sheetname", config.NoExpand()),
	})

	skipEmpty := config.Get[bool](cf, "output.skip-empty-reports")
	showEmpty := config.Get[bool](cf, "output.show-empty-samplers")

	for _, plugin := range config.Get[[]string](cf, "output.plugins.single-column") {
		rows := 0
		for _, e := range Entities {
			for _, s := range e.Samplers {
				if s.Plugin == plugin {
					if len(s.Column1) > 0 {
						rows += len(s.Column1)
					} else if showEmpty {
						rows++
					}
				}
			}
		}
		if rows == 0 && skipEmpty {
			continue
		}

		pluginFile := config.Get[string](cf, config.Join("output", "reports", plugin, "filename"), conftable) + ".csv"
		err = writeCSVFile(subdir, pluginFile, func(w io.Writer) error {
			return outputCSVSinglePluginWithRowname(w, Entities, cf, conftable, plugin)
		})
		if err != nil {
			log.Error("failed to create CSV file", slog.Any("error", err))
			return
		}

		csvfiles = append(csvfiles, csvFiles{
			path:      filepath.Join(subdir, pluginFile),
			filename:  config.Get[string](cf, config.Join("output", "reports", plugin, "filename"), config.NoExpand()),
			sheetname: config.Get[string](cf, config.Join("output", "reports", plugin, "sheetname"), config.NoExpand()),
		})
	}

	for _, plugin := range config.Get[[]string](cf, "output.plugins.two-column") {
		rows := 0
		for _, e := range Entities {
			for _, s := range e.Samplers {
				if s.Plugin == plugin {
					if len(s.Column1)+len(s.Column2) > 0 {
						rows += len(s.Column1) + len(s.Column2)
					} else if showEmpty {
						rows++
					}
				}
			}
		}
		if rows == 0 && skipEmpty {
			continue
		}

		pluginFile := config.Get[string](cf, config.Join("output", "reports", plugin, "filename"), conftable) + ".csv"
		err = writeCSVFile(subdir, pluginFile, func(w io.Writer) error {
			return outputCSVTwoColumnPluginWithRowname(w, Entities, cf, conftable, plugin)
		})
		if err != nil {
			log.Error("failed to create CSV file", slog.Any("error", err))
			return
		}

		csvfiles = append(csvfiles, csvFiles{
			path:      filepath.Join(subdir, pluginFile),
			filename:  config.Get[string](cf, config.Join("output", "reports", plugin, "filename"), config.NoExpand()),
			sheetname: config.Get[string](cf, config.Join("output", "reports", plugin, "sheetname"), config.NoExpand()),
		})
	}

	return
}

func outputCSVSummary(w io.Writer, cf *config.Config, gateway string, entities []Entity, probes map[string]geneos.Probe) (err error) {
	scsv := csv.NewWriter(w)
	hostname, _ := os.Hostname()
	return scsv.WriteAll([][]string{
		{"Name", "Value"},
		{"ITRS Gateway Reporter", "Version: " + cordial.VERSION},
		{"Site", config.Get[string](cf, "site", config.DefaultValue("ITRS"))},
		{"Report Date/Time", startTime.Format(time.RFC3339)},
		{"Hostname", hostname},
		{"Gateway Name", gateway},
		{"Probes", strconv.Itoa(len(probes))},
		{"Managed Entities", strconv.Itoa(len(entities))},
	})
}

// fitRow returns row padded with empty strings or truncated so that it
// has exactly n columns, to keep rows consistent with the (configurable)
// header
func fitRow[T any](row []T, n int) []T {
	if len(row) >= n {
		return row[:n]
	}
	return append(row, make([]T, n-len(row))...)
}

// pluginColumns returns the configured column headings for the report
func pluginColumns(cf *config.Config, report string, defaults []string) []string {
	return config.Get[[]string](cf,
		config.Join("output", "reports", report, "columns"),
		config.DefaultValue(defaults),
	)
}

// entityColumns returns the configured leading columns for the entities
// report
func entityColumns(cf *config.Config) []string {
	return config.Get[[]string](cf, "output.reports.entities.columns",
		config.DefaultValue([]string{"managedEntity", "probe", "hostname", "port"}),
	)
}

// entityRow returns the leading columns for the entities report, fitted
// to the number of configured columns
func entityRow(e Entity, n int) []string {
	port := "7036"
	hostname := e.Probe.Hostname
	if hostname == "" {
		hostname = "[Virtual Probe]"
		port = ""
	} else if e.Probe.Port != 0 {
		port = strconv.Itoa(e.Probe.Port)
	}
	return fitRow([]string{e.Name, e.Probe.Name, hostname, port}, n)
}

func outputEntityColumns(Entities []Entity, cf *config.Config, conftable config.ExpandOption) (cols, attrs, plugins []string, err error) {
	cols = slices.Clone(entityColumns(cf))

	attrs = config.Get[[]string](cf, "output.reports.entities.attributes")
	if len(attrs) == 0 {
		attrs = getAttributes(Entities)
	}

	cols = append(cols, attrs...)

	plugins = config.Get[[]string](cf, "output.reports.entities.plugins")
	if len(plugins) == 0 {
		plugins = getPlugins(Entities)
	}
	cols = append(cols, plugins...)

	return
}

func outputCVSEntities(w io.Writer, Entities []Entity, cf *config.Config, conftable config.ExpandOption) (err error) {
	cols, attrs, plugins, err := outputEntityColumns(Entities, cf, conftable)
	if err != nil {
		return
	}
	nbase := len(entityColumns(cf))

	ecsv := csv.NewWriter(w)

	if err = ecsv.Write(cols); err != nil {
		return
	}
	for _, e := range Entities {
		cols := entityRow(e, nbase)

		for _, a := range attrs {
			cols = append(cols, e.Attributes[a])
		}
		ptotals := make(map[string]int)
		for _, s := range e.Samplers {
			if s.Plugin == "" {
				ptotals["unsupported"]++
			} else {
				ptotals[s.Plugin]++
			}
		}
		for _, p := range plugins {
			if ptotals[p] > 0 {
				cols = append(cols, strconv.Itoa(ptotals[p]))
			} else {
				cols = append(cols, "")
			}
		}
		if err = ecsv.Write(cols); err != nil {
			return
		}
	}
	ecsv.Flush()

	return ecsv.Error()
}

func outputCSVSinglePlugin(w io.Writer, Entities []Entity, cf *config.Config, conftable config.ExpandOption, plugin string) (err error) {
	fcsv := csv.NewWriter(w)

	columns := pluginColumns(cf, plugin, []string{"managedEntity", "samplerType", "samplerName", "data"})
	if err = fcsv.Write(columns); err != nil {
		return
	}

	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				for _, data := range s.Column1 {
					if err = fcsv.Write(fitRow([]string{e.Name, s.Type, s.Name, data}, len(columns))); err != nil {
						return
					}
				}
			}
		}
	}
	fcsv.Flush()
	return fcsv.Error()
}

// twoColumnRows returns the rows for a two column sampler, padding the
// shorter list with empty strings
func twoColumnRows(e Entity, s Sampler, n int) (rows [][]string) {
	for i := range max(len(s.Column2), len(s.Column1)) {
		row := []string{e.Name, s.Type, s.Name, "", ""}
		if len(s.Column1) > i {
			row[3] = s.Column1[i]
		}
		if len(s.Column2) > i {
			row[4] = s.Column2[i]
		}
		rows = append(rows, fitRow(row, n))
	}
	return
}

func outputCSVTwoColumnPlugin(w io.Writer, Entities []Entity, cf *config.Config, conftable config.ExpandOption, plugin string) (err error) {
	fcsv := csv.NewWriter(w)

	columns := pluginColumns(cf, plugin, []string{"managedEntity", "samplerType", "samplerName", "data1", "data2"})
	if err = fcsv.Write(columns); err != nil {
		return
	}

	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				for _, row := range twoColumnRows(e, s, len(columns)) {
					if err = fcsv.Write(row); err != nil {
						return
					}
				}
			}
		}
	}
	fcsv.Flush()
	return fcsv.Error()
}

// samplerRowname returns the first column for the CSV files for
// toolkit consumption
func samplerRowname(e Entity, s Sampler) (rowname string) {
	rowname = e.Name + "-" + s.Name
	if s.Type != "" {
		rowname += " (" + s.Type + ")"
	}
	return
}

func outputCSVSinglePluginWithRowname(w io.Writer, Entities []Entity, cf *config.Config, conftable config.ExpandOption, plugin string) (err error) {
	fcsv := csv.NewWriter(w)

	columns := pluginColumns(cf, plugin, []string{"managedEntity", "samplerType", "samplerName", "data"})
	if err = fcsv.Write(append([]string{"rowname"}, columns...)); err != nil {
		return
	}

	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				for _, data := range s.Column1 {
					rowname := samplerRowname(e, s) + "-" + data
					row := append([]string{rowname}, fitRow([]string{e.Name, s.Type, s.Name, data}, len(columns))...)
					if err = fcsv.Write(row); err != nil {
						return
					}
				}
			}
		}
	}
	fcsv.Flush()
	return fcsv.Error()
}

func outputCSVTwoColumnPluginWithRowname(w io.Writer, Entities []Entity, cf *config.Config, conftable config.ExpandOption, plugin string) (err error) {
	fcsv := csv.NewWriter(w)

	columns := pluginColumns(cf, plugin, []string{"managedEntity", "samplerType", "samplerName", "data1", "data2"})
	if err = fcsv.Write(append([]string{"rowname"}, columns...)); err != nil {
		return
	}

	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				for _, row := range twoColumnRows(e, s, len(columns)) {
					if err = fcsv.Write(append([]string{samplerRowname(e, s)}, row...)); err != nil {
						return
					}
				}
			}
		}
	}
	fcsv.Flush()
	return fcsv.Error()
}

func getAttributes(Entities []Entity) (attrnames []string) {
	attrs := make(map[string]bool)

	for _, e := range Entities {
		for a := range e.Attributes {
			attrs[a] = true
		}
	}

	return slices.Sorted(maps.Keys(attrs))
}

func getPlugins(Entities []Entity) (plugins []string) {
	plugmap := make(map[string]bool)

	for _, e := range Entities {
		for _, a := range e.Samplers {
			if a.Plugin == "" {
				plugmap["unsupported"] = true
			} else {
				plugmap[a.Plugin] = true
			}
		}
	}

	return slices.Sorted(maps.Keys(plugmap))
}
