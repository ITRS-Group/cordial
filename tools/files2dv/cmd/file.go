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
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func processFiles(dv *config.Config) (dataview Dataview, err error) {
	dataview.Name = dv.GetString("name")
	columns := []Column{}

	max := dv.GetInt("row-limit")
	var n int

	ignores := []*regexp.Regexp{}
	var matches int

	for _, i := range dv.GetStringSlice("ignore-lines") {
		if r, err := regexp.Compile(i); err != nil {
			log.Error().Err(err).Msgf("compile of '%s' failed", i)
		} else {
			ignores = append(ignores, r)
		}
	}

	// try direct unmarshal, fall back to slice of strings
	err = dv.UnmarshalKey("columns", &columns, viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc()))
	if err != nil {
		// reset columns
		columns = []Column{}
		cols := dv.GetStringSlice("columns", config.NoExpand())
		if len(cols) == 0 {
			err = errors.New("columns is not either an array or strings or maps of the right type")
			return
		}
		values := dv.GetStringSlice("values", config.NoExpand())
		if len(values) != len(cols) {
			err = errors.New("number of columns does not match number of values")
			return
		}
		for i, c := range cols {
			columns = append(columns, Column{
				Name:  c,
				Value: values[i],
			})
		}
	}

	colNames := []string{}
	for _, c := range columns {
		colNames = append(colNames, c.Name)
		if c.Match != nil {
			matches++
		}
	}
	dataview.Table = append(dataview.Table, colNames)

	fileCount := 0
	// fileErrors := 0
	// fileFails := 0

	for _, pattern := range dv.GetStringSlice("paths") {
		var path string
		path, err = geneos.ExpandFileDates(pattern, time.Now())
		if err != nil {
			continue
		}
		var files []string
		if strings.ContainsAny(path, "*?[\\") {
			files, err = filepath.Glob(path)
			if err != nil {
				log.Error().Err(err).Msgf("match of pattern %q failed", path)
				continue
			}
		} else {
			files = append(files, path)
		}

		fileCount += len(files)

		if len(files) == 0 {
			if slices.Contains(dv.GetStringSlice("ignore-file-errors"), "match") {
				continue
			}
			fullpath, err := filepath.Abs(path)
			if err != nil {
				fullpath = path
			}
			lookup := map[string]string{
				"fullpath": fullpath,
				"path":     path,
				"pattern":  pattern,
				"filename": "",
				"status":   "NO_MATCH",
			}
			cols := make([]string, len(columns))

			for i, c := range columns {
				if c.Match == nil {
					cols[i] = dv.ExpandString(c.Value, config.LookupTable(lookup))
				}
			}

			dataview.Table = append(dataview.Table, cols)
			continue
		}

		// dv.Set("types", []string{"file", "symlink"})

		for _, file := range files {
			if max > 0 && n >= max {
				return
			}
			n++

			lookup, skip := buildFileLookupTable(dv, file, pattern)
			if skip {
				cols := []string{}
				for _, c := range columns {
					cols = append(cols, dv.ExpandString(c.Value, config.LookupTable(lookup)))
				}

				dataview.Table = append(dataview.Table, cols)
				continue
			}

			if matches == 0 {
				cols := []string{}
				for _, c := range columns {
					cols = append(cols, dv.ExpandString(c.Value, config.LookupTable(lookup)))
				}

				dataview.Table = append(dataview.Table, cols)
				continue
			}

			// open file and scan for matches

			values := make([]string, len(colNames))
			for i, c := range columns {
				if c.Match == nil {
					values[i] = cf.ExpandString(c.Value, config.LookupTable(lookup))
				}
			}

			inp, err := os.Open(file)
			if err != nil {
				log.Error().Err(err).Msgf("cannot open %s", file)
				continue
			}
			maxlines := dv.GetInt("max-lines")

			s := bufio.NewScanner(inp)
		LINE:
			// loop until end of file or maxlines scanned - if maxlines
			// is zero or less, then no limit
			for i := 0; s.Scan() && (maxlines < 1 || i < maxlines) && matches > 0; i++ {
				line := s.Text()
				// skip ignored matches
				for _, ignore := range ignores {
					if ignore.MatchString(line) {
						continue LINE
					}
				}

				// check for matches, skip columns with existing values (first match wins)
				for i, c := range columns {
					if values[i] != "" {
						continue
					}

					if c.Match != nil {
						num := c.Match.NumSubexp()
						names := c.Match.SubexpNames()
						submatchLookup := make(map[string]string, num+len(names))
						if numMatches := c.Match.FindStringSubmatchIndex(line); len(numMatches) > 0 {
							// add indexes to colLookup (inc ${0} for whole match)
							for j := 0; j <= num; j++ {
								start, end := numMatches[j*2], numMatches[j*2+1]
								submatchLookup[strconv.Itoa(j)] = line[start:end]
							}

							// add named matches to colLookup
							for _, n := range names {
								if n == "" {
									continue
								}
								i := c.Match.SubexpIndex(n)
								submatchLookup[n] = line[numMatches[i*2]:numMatches[i*2+1]]
							}

							values[i] = dv.ExpandString(c.Value, config.LookupTable(submatchLookup, lookup))
							matches--
						}
					}
				}
			}
			inp.Close()

			finalStatus := ""
			onFail := dv.GetString("on-fail.status", config.NoExpand())

			for i, col := range values {
				if col == "" && columns[i].Fail != "" {
					// set status to "on-fail.status"
					if onFail != "" && finalStatus == "" {
						finalStatus = dv.ExpandString(onFail, config.LookupTable(map[string]string{"status": onFail}, lookup))
					}
					values[i] = dv.ExpandString(columns[i].Fail, config.LookupTable(lookup))
				}
			}

			if finalStatus != "" {
				for i, col := range columns {
					if strings.Contains(col.Value, "${status}") {
						values[i] = dv.ExpandString(col.Value, config.LookupTable(map[string]string{"status": finalStatus}, lookup))
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
