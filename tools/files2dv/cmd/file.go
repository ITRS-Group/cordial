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
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
	"github.com/rs/zerolog/log"
)

func processFiles(dv *config.Config) (dataview Dataview, err error) {
	dataview.Name = dv.GetString("name")
	ignores := []*regexp.Regexp{}

	for _, i := range dv.GetStringSlice("file.ignore-lines") {
		if r, err := regexp.Compile(i); err != nil {
			log.Error().Err(err).Msgf("compile of '%s' failed", i)
		} else {
			ignores = append(ignores, r)
		}
	}

	columns := []Column{}

	colNames := []string{}
	colSpec := dv.GetSliceStringMapString("columns", config.NoExpand())
	for i, c := range colSpec {
		name, ok := c["name"]
		if !ok {
			log.Error().Msgf("no column name found for entry %d", i)
		}
		colNames = append(colNames, name)
		value, ok := c["value"]
		if !ok {
			log.Error().Msgf("no value found for entry %d", i)
		}
		col := Column{
			Name:  name,
			Value: value,
			Check: c["check"],
		}
		match, ok := c["match"]
		if ok {
			col.Regexp, err = regexp.Compile(match)
			if err != nil {
				log.Error().Err(err).Msg("")
			}
		}
		columns = append(columns, col)
	}
	dataview.Table = append(dataview.Table, colNames)
	split := dv.GetString("split")

	fileCount := 0
	// fileErrors := 0
	// fileFails := 0

	for _, pattern := range dv.GetStringSlice("paths") {
		path, err := geneos.ExpandFileDates(pattern, time.Now())
		if err != nil {
			continue
		}
		files, err := filepath.Glob(path)
		if err != nil {
			log.Error().Err(err).Msgf("match of pattern %q failed", path)
			continue
		}

		fileCount += len(files)

		if len(files) == 0 {
			if slices.Contains(dv.GetStringSlice("ignore-file-errors"), "match") {
				continue
			}
			lookup := map[string]string{
				"path":     path,
				"filename": "",
				"status":   "NO_MATCH",
			}
			columns := make([]string, len(colSpec))
			for i, c := range colSpec {
				if c["match"] == "" {
					v := dv.ExpandString(c["value"], config.LookupTable(lookup))
					columns[i] = v
				}
			}

			dataview.Table = append(dataview.Table, columns)
			continue
		}

		dv.Set("types", []string{"file", "symlink"})

		for _, file := range files {
			lookup, skip := buildFileLookupTable(dv, file, pattern)
			if skip {
				columns := []string{}
				for _, c := range colSpec {
					v := dv.ExpandString(c["value"], config.LookupTable(lookup))
					columns = append(columns, v)
				}

				dataview.Table = append(dataview.Table, columns)
				continue
			}

			values := make([]string, len(colNames))

			// fill in non-match columns once per file, unless "split" is used
			for i, c := range columns {
				if c.Regexp == nil {
					// if we are using the split option, don't
					// update values with expand strings until later
					if split == "" || (split != "" && strings.Contains(c.Value, "${")) {
						values[i] = cf.ExpandString(c.Value, config.LookupTable(lookup))
					}
				}
			}

			inp, err := os.Open(file)
			if err != nil {
				log.Error().Err(err).Msgf("cannot open %s", file)
				continue
			}
			maxlines := dv.GetInt("file.max-lines")

			s := bufio.NewScanner(inp)
		LINE:
			// loop until end of file or maxlines scanned - if maxlines
			// is zero or less, then no limit
			for i := 0; s.Scan() && (maxlines < 1 || i < maxlines); i++ {
				line := s.Text()
				// skip ignored matches
				for _, ignore := range ignores {
					if ignore.MatchString(line) {
						continue LINE
					}
				}

				// check for matches, skip non-empty (first match wins, initially)
				for i, c := range columns {
					if values[i] != "" {
						continue
					}

					if c.Regexp != nil {
						num := c.Regexp.NumSubexp()
						names := c.Regexp.SubexpNames()
						submatchLookup := make(map[string]string, num+len(names))
						if matches := c.Regexp.FindStringSubmatchIndex(line); len(matches) > 0 {
							// add indexes to colLookup (inc ${0} for whole match)
							for j := 0; j <= num; j++ {
								start, end := matches[j*2], matches[j*2+1]
								submatchLookup[strconv.Itoa(j)] = line[start:end]
							}

							// add named matches to colLookup
							for _, n := range names {
								if n == "" {
									continue
								}
								i := c.Regexp.SubexpIndex(n)
								submatchLookup[n] = line[matches[i*2]:matches[i*2+1]]
							}

							values[i] = dv.ExpandString(c.Value, config.LookupTable(submatchLookup, lookup))
						}
					}
				}
			}

			finalStatus := ""
			onFail := dv.GetString("on-fail.status", config.NoExpand())

			for i, col := range values {
				if col == "" && colSpec[i]["fail"] != "" {
					// set status to "on-fail.status"
					if onFail != "" && finalStatus == "" {
						finalStatus = dv.ExpandString(onFail, config.LookupTable(map[string]string{"status": onFail}, lookup))
					}
					values[i] = dv.ExpandString(colSpec[i]["fail"], config.LookupTable(lookup))
				}
			}

			if finalStatus != "" {
				for i, col := range colSpec {
					if strings.Contains(col["value"], "${status}") {
						values[i] = dv.ExpandString(col["value"], config.LookupTable(map[string]string{"status": finalStatus}, lookup))
					}
				}
			}

			dataview.Table = append(dataview.Table, values)
		}
	}

	headlinesLookup := map[string]string{}

	for _, h := range dv.GetSliceStringMapString("headlines", config.NoExpand()) {
		dataview.Headlines = append(dataview.Headlines, Headline{
			Name:  h["name"],
			Value: dv.ExpandString(h["value"], config.LookupTable(headlinesLookup)),
		})
	}

	return
}
