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
	"archive/zip"
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
)

// outputCSV writes the slice of Entity structs to a zip files in
//
// the default files are:
//
// - entities.csv - all entities, their probes and a variable number of
// attributes where the column names are the attributes in alphabetical
// order. Also, one column per plugin type (not sampler) and total of instances
// - file plugins (fkm, ftm, stateTracker)
// - processes.csv
func outputCSV(cf *config.Config, gateway string, Entities []Entity, probes map[string]geneos.Probe) (err error) {
	dir := cf.GetString("output.directory")
	_ = os.MkdirAll(dir, 0775)

	conftable := config.LookupTable(map[string]string{
		"gateway":  gateway,
		"datetime": startTimestamp,
	})

	filename := cf.GetString("output.formats.csv", conftable)
	if !filepath.IsAbs(filename) {
		filename = path.Join(dir, filename)
	}

	zipfile, err := os.Create(filename)
	defer zipfile.Close()

	z := zip.NewWriter(zipfile)

	// output a summary file
	outputCSVSummary(z, cf, gateway, Entities, probes)

	// entities.csv
	outputCVSEntities(z, Entities, cf, conftable)

	for _, f := range cf.GetStringSlice("output.plugins.single-column") {
		outputCSVSinglePlugin(z, Entities, cf, conftable, f)
	}

	for _, f := range cf.GetStringSlice("output.plugins.two-column") {
		outputCSVTwoColumnPlugin(z, Entities, cf, conftable, f)
	}

	z.Close()

	return
}

func outputCSVSummary(z *zip.Writer, cf *config.Config, gateway string, entities []Entity, probes map[string]geneos.Probe) (err error) {
	conftable := config.LookupTable(map[string]string{
		"gateway":  gateway,
		"datetime": startTimestamp,
	})

	filename := cf.GetString("output.reports.summary.filename", conftable) + ".csv"
	w, err := z.Create(filename)
	if err != nil {
		return
	}
	scsv := csv.NewWriter(w)
	hostname, _ := os.Hostname()
	scsv.WriteAll([][]string{
		{"Name", "Value"},
		{"ITRS Gateway Reporter", "Version: " + cordial.VERSION},
		{"Site", cf.GetString("site", config.Default("ITRS"))},
		{"Report Date/Time", startTime.Format(time.RFC3339)},
		{"Hostname", hostname},
		{"Gateway Name", gateway},
		{"Probes", strconv.Itoa(len(probes))},
		{"Managed Entities", strconv.Itoa(len(entities))},
	})
	return
}

func outputEntityColumns(Entities []Entity, cf *config.Config, conftable config.ExpandOptions) (cols, attrs, plugins []string, err error) {
	cols = cf.GetStringSlice("output.reports.entities.columns",
		config.Default([]string{"managedEntity", "probe", "hostname", "port"}),
	)

	attrs = cf.GetStringSlice("output.reports.entities.attributes")
	if len(attrs) == 0 {
		attrs = getAttributes(Entities)
	}

	cols = append(cols, attrs...)

	plugins = cf.GetStringSlice("output.reports.entities.plugins")
	if len(plugins) == 0 {
		plugins = getPlugins(Entities)
	}
	for _, p := range plugins {
		cols = append(cols, p)
	}

	return
}

func outputCVSEntities(z *zip.Writer, Entities []Entity, cf *config.Config, conftable config.ExpandOptions) (err error) {
	cols, attrs, plugins, err := outputEntityColumns(Entities, cf, conftable)

	filename := cf.GetString("output.reports.entities.filename", conftable) + ".csv"
	w, err := z.Create(filename)
	if err != nil {
		return
	}
	ecsv := csv.NewWriter(w)

	if err = ecsv.Write(cols); err != nil {
		z.Close()
		return
	}
	for _, e := range Entities {
		port := "7036"
		hostname := e.Probe.Hostname
		if hostname == "" {
			hostname = "[Virtual Probe]"
			port = ""
		} else if e.Probe.Port != 0 {
			port = fmt.Sprint(e.Probe.Port)
		}
		cols := []string{
			e.Name,
			e.Probe.Name,
			hostname,
			port,
		}

		for _, a := range attrs {
			if attr, ok := e.Attributes[a]; ok {
				cols = append(cols, attr)
			} else {
				cols = append(cols, "")
			}
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
				cols = append(cols, fmt.Sprint(ptotals[p]))
			} else {
				cols = append(cols, "")
			}
		}
		ecsv.Write(cols)
	}
	ecsv.Flush()

	return
}

func outputCSVSinglePlugin(z *zip.Writer, Entities []Entity, cf *config.Config, conftable config.ExpandOptions, plugin string) (err error) {
	rows := 0
	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				rows += len(s.Column1)
			}
		}
	}
	if rows == 0 && cf.GetBool("output.skip-empty-reports") {
		return
	}

	filename := cf.GetString(config.Join("output", "reports", plugin, "filename"), conftable) + ".csv"

	w, err := z.Create(filename)
	if err != nil {
		return
	}

	fcsv := csv.NewWriter(w)

	fcsv.Write(cf.GetStringSlice(
		config.Join("output", "reports", plugin, "columns"),
		config.Default([]string{
			"managedEntity",
			"samplerType",
			"samplerName",
			"data",
		}),
	))

	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				for _, data := range s.Column1 {
					fcsv.Write([]string{
						e.Name,
						s.Type,
						s.Name,
						data,
					})
				}
			}
		}
	}
	fcsv.Flush()
	return
}

func outputCSVTwoColumnPlugin(z *zip.Writer, Entities []Entity, cf *config.Config, conftable config.ExpandOptions, plugin string) (err error) {
	rows := 0
	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				rows += len(s.Column1) + len(s.Column2)
			}
		}
	}
	if rows == 0 && cf.GetBool("output.skip-empty-reports") {
		return
	}

	filename := cf.GetString(config.Join("output", "reports", plugin, "filename"), conftable) + ".csv"
	w, err := z.Create(filename)
	if err != nil {
		return
	}

	fcsv := csv.NewWriter(w)

	fcsv.Write(cf.GetStringSlice(
		config.Join("output", "reports", "files", plugin, "columns"),
		config.Default([]string{
			"managedEntity",
			"samplerType",
			"samplerName",
			"data1",
			"data2",
		}),
	))

	for _, e := range Entities {
		for _, s := range e.Samplers {
			if s.Plugin == plugin {
				items := len(s.Column1)
				if len(s.Column2) > items {
					items = len(s.Column2)
				}
				for i := 0; i < items; i++ {
					row := []string{
						e.Name,
						s.Type,
						s.Name,
					}
					if len(s.Column1) > i {
						row = append(row, s.Column1[i])
					} else {
						row = append(row, "")
					}
					if len(s.Column2) > i {
						row = append(row, s.Column2[i])
					} else {
						row = append(row, "")
					}
					fcsv.Write(row)
				}

			}
		}
	}
	fcsv.Flush()
	return
}

func getAttributes(Entities []Entity) (attrnames []string) {
	attrs := make(map[string]bool)

	for _, e := range Entities {
		for a := range e.Attributes {
			attrs[a] = true
		}
	}

	for a := range attrs {
		attrnames = append(attrnames, a)
	}

	sort.Strings(attrnames)
	return
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

	for p := range plugmap {
		plugins = append(plugins, p)
	}

	sort.Strings(plugins)
	return
}
