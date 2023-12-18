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
	"path/filepath"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
	"github.com/rs/zerolog/log"
)

func processInfo(dv *config.Config) (dataview Dataview, err error) {
	dataview.Name = dv.GetString("name")

	max := dv.GetInt("info.limit")
	var n int

	dataview.Table = append(dataview.Table, dv.GetStringSlice("columns"))

	// get raw values, expand later, with lookups for each file
	values := dv.GetStringSlice("values", config.NoExpand())

	// list of wanted file types to a map
	ft := dv.GetStringSlice("info.types", config.Default([]string{"file", "directory", "symlink", "other"}))
	filetypes := make(map[string]bool, len(ft))
	for _, f := range ft {
		filetypes[f] = true
	}

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
				log.Error().Err(err).Msgf("match %s failed", path)
				continue
			}

			if len(files) == 0 {
				if dv.GetBool("ignore-file-errors.match") {
					continue
				}
				lookup := map[string]string{
					"path":     path,
					"filename": filepath.Base(path),
					"status":   "NO_MATCH",
				}
				columns := []string{}
				for _, c := range values {
					v := dv.ExpandString(c, config.LookupTable(lookup))
					columns = append(columns, v)
				}

				dataview.Table = append(dataview.Table, columns)
				continue
			}
		} else {
			files = append(files, path)
		}

		for _, f := range files {
			if n >= max {
				return
			}
			n++

			lookup, skip := buildFileLookupTable(dv, f, filetypes)
			if skip {
				continue
			}
			lookup["pattern"] = pattern

			columns := []string{}
			for _, c := range values {
				v := dv.ExpandString(c, config.LookupTable(lookup))
				columns = append(columns, v)
			}

			dataview.Table = append(dataview.Table, columns)
		}
	}
	return
}
