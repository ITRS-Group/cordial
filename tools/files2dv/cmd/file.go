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
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
)

// maxLineSize is the longest line that will be scanned from a file
const maxLineSize = 16 * 1024 * 1024

func processFiles(dv *config.Config) (dataview Dataview, err error) {
	dataview.Name = config.Get[string](dv, "name")
	columns := []Column{}

	rowLimit := config.Get[int](dv, "row-limit")
	var n int

	ignores := []*regexp.Regexp{}

	for _, i := range config.Get[[]string](dv, "ignore-lines") {
		if r, err := regexp.Compile(i); err != nil {
			log.Error("failed to compile regex", slog.Any("error", err), slog.String("pattern", i))
		} else {
			ignores = append(ignores, r)
		}
	}

	// try direct unmarshal, fall back to slice of strings
	err = dv.UnmarshalKey("columns", &columns, config.NoExpand())
	if err != nil {
		// reset columns
		columns = []Column{}
		cols := config.Get[[]string](dv, "columns")
		if len(cols) == 0 {
			err = errors.New("columns is not either an array or strings or maps of the right type")
			return
		}
		// values are expanded later, per file, so must not be expanded here
		values := config.Get[[]string](dv, "values", config.NoExpand())
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
		// the fall-back worked, so the unmarshal error is not an error
		err = nil
	}

	colNames := []string{}
	var matches int
	for _, c := range columns {
		colNames = append(colNames, c.Name)
		if c.Match != nil {
			matches++
		}
	}
	dataview.Table = append(dataview.Table, colNames)

PATHS:
	for _, pattern := range config.Get[[]string](dv, "paths") {
		path, err := geneos.ExpandFileDates(pattern, time.Now())
		if err != nil {
			log.Error("failed to expand path", slog.Any("error", err), slog.String("pattern", pattern))
			continue
		}
		var files []string
		if strings.ContainsAny(path, "*?[\\") {
			files, err = filepath.Glob(path)
			if err != nil {
				log.Error("failed to match pattern", slog.Any("error", err), slog.String("pattern", path))
				continue
			}
		} else {
			files = append(files, path)
		}

		if len(files) == 0 {
			if slices.Contains(config.Get[[]string](dv, "ignore-file-errors"), "match") {
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
					cols[i] = config.Expand[string](dv, c.Value, config.LookupTable(lookup))
				}
			}

			dataview.Table = append(dataview.Table, cols)
			continue
		}

		for _, file := range files {
			// stop adding rows, but still output the headlines
			if rowLimit > 0 && n >= rowLimit {
				break PATHS
			}

			lookup, skip := buildFileLookupTable(dv, file, pattern)
			if skip {
				// drop this file from the output
				continue
			}

			values := make([]string, len(colNames))
			for i, c := range columns {
				if c.Match == nil {
					values[i] = config.Expand[string](dv, c.Value, config.LookupTable(lookup))
				}
			}

			// only open the file if there is something to look for
			if matches > 0 {
				inp, err := os.Open(file)
				if err != nil {
					log.Error("cannot open file", slog.Any("error", err), slog.String("file", file))
					continue
				}
				err = scanFile(dv, inp, columns, ignores, lookup, values)
				inp.Close()
				if err != nil {
					log.Error("error reading file", slog.Any("error", err), slog.String("file", file))
				}
				values = applyFailures(dv, columns, values, lookup)
			}

			n++
			dataview.Table = append(dataview.Table, values)
		}
	}

	headlinesLookup := map[string]string{}

	for _, h := range config.Get[[]map[string]string](dv, "headlines", config.NoExpand()) {
		dataview.Headlines = append(dataview.Headlines, Headline{
			Name:  h["name"],
			Value: config.Expand[string](dv, h["value"], config.LookupTable(headlinesLookup)),
		})
	}

	return
}

// scanFile reads inp, a line at a time, and fills in the values of
// the columns that have a `match` regular expression using the first
// line that matches (ignoring any lines that match any of ignores).
// The scan stops when every column with a `match` has a value, at the
// end of the file or after `max-lines` lines.
//
// The state of which columns have been matched is local to each call,
// so every file is scanned in full.
func scanFile(dv *config.Config, inp io.Reader, columns []Column, ignores []*regexp.Regexp, lookup map[string]string, values []string) error {
	remaining := 0
	for _, c := range columns {
		if c.Match != nil {
			remaining++
		}
	}
	done := make([]bool, len(columns))
	maxlines := config.Get[int](dv, "max-lines")

	s := bufio.NewScanner(inp)
	s.Buffer(make([]byte, 0, 64*1024), maxLineSize)

	lines := 0
LINE:
	// loop until end of file or maxlines scanned - if maxlines is zero
	// or less, then no limit - or every match column has a value
	for remaining > 0 && (maxlines < 1 || lines < maxlines) && s.Scan() {
		lines++
		line := s.Text()
		// skip ignored matches
		for _, ignore := range ignores {
			if ignore.MatchString(line) {
				continue LINE
			}
		}

		// check for matches, skip columns with existing values (first match wins)
		for i, c := range columns {
			if c.Match == nil || done[i] {
				continue
			}
			idx := c.Match.FindStringSubmatchIndex(line)
			if len(idx) == 0 {
				continue
			}
			values[i] = config.Expand[string](dv, c.Value, config.LookupTable(submatchLookup(c.Match, line, idx), lookup))
			done[i] = true
			remaining--
		}
	}
	return s.Err()
}

// submatchLookup returns a lookup table of the numbered (including
// ${0} for the whole match) and named sub-matches of re in line, given
// the indexes from FindStringSubmatchIndex. Groups that did not take
// part in the match are empty.
func submatchLookup(re *regexp.Regexp, line string, idx []int) map[string]string {
	lookup := make(map[string]string, re.NumSubexp()+1)
	group := func(j int) string {
		if start, end := idx[j*2], idx[j*2+1]; start >= 0 && end >= 0 {
			return line[start:end]
		}
		return ""
	}

	for j := 0; j <= re.NumSubexp(); j++ {
		lookup[strconv.Itoa(j)] = group(j)
	}
	for j, n := range re.SubexpNames() {
		if n != "" {
			lookup[n] = group(j)
		}
	}
	return lookup
}

// applyFailures replaces empty column values with their `fail` value
// and, if any were replaced, sets the `${status}` for the row from the
// `on-fail.status` setting
func applyFailures(dv *config.Config, columns []Column, values []string, lookup map[string]string) []string {
	finalStatus := ""
	onFail := config.Get[string](dv, "on-fail.status", config.NoExpand())

	for i, col := range values {
		if col == "" && columns[i].Fail != "" {
			// set status to "on-fail.status"
			if onFail != "" && finalStatus == "" {
				finalStatus = config.Expand[string](dv, onFail, config.LookupTable(map[string]string{"status": onFail}, lookup))
			}
			values[i] = config.Expand[string](dv, columns[i].Fail, config.LookupTable(lookup))
		}
	}

	if finalStatus != "" {
		for i, col := range columns {
			if strings.Contains(col.Value, "${status}") {
				values[i] = config.Expand[string](dv, col.Value, config.LookupTable(map[string]string{"status": finalStatus}, lookup))
			}
		}
	}
	return values
}
