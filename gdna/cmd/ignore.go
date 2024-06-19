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
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

// ignores are loaded from files/urls defined in the config (or the raw
// content), and overwrite the contents of the tables given
func loadIgnores(ctx context.Context, cf *config.Config, tx *sql.Tx) error {
OUTER:
	for ignore := range cf.GetStringMap("ignore") {
		var r io.Reader
		source := cf.GetString(config.Join("ignore", ignore, "source"))
		content := cf.GetString(config.Join("ignore", ignore, "content"))

		if source == "" && content == "" {
			log.Debug().Msgf("no ignore for %s, skipping", ignore)
			continue
		}

		log.Debug().Msgf("trying to load ignores from %s", source)
		s, err := openSource(ctx, source)
		if err != nil {
			if content == "" {
				continue
			}
			r = bytes.NewBufferString(content)
		} else {
			defer s.Close()
			r = s
		}

		table, err := cf.ExpandRawString(config.Join("ignore", ignore, "table"))
		if err != nil {
			log.Error().Err(err).Msg(config.Join("ignore", ignore, "table"))
			continue
		}

		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			log.Info().Err(err).Msgf("delete from %s failed", table)
			// NOT an error in itself
			err = nil
		}

		insertStmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s VALUES (?);", table))
		if err != nil {
			log.Error().Err(err).Msgf("prepare for %s failed", table)
			continue
		}
		defer insertStmt.Close()

		// use csv even for a single column for consistency (and comment line ignores)
		c := csv.NewReader(r)
		c.ReuseRecord = true
		c.Comment = '#'

		lines := 0
		for {
			fields, err := c.Read()
			if err == io.EOF {
				// all good, end of CSV input, return
				break
			}
			if err != nil {
				return err
			}
			if _, err = insertStmt.ExecContext(ctx, &fields[0]); err != nil {
				log.Error().Err(err).Msgf("insert for %s failed", table)
				continue OUTER
			}
			lines++
		}
		log.Debug().Msgf("read %d lines from %s and added to %s", lines, source, table)
	}
	return nil
}

func ignoreColumns(cf *config.Config) (columns []string) {
	for e := range cf.GetStringMap("ignore") {
		columns = append(columns, cf.GetString(config.Join("ignore", e, "column")))
	}
	return
}

// ignores returns a config.ExpandOptions LookupTable ready for
// GetString()
func ignores(cf *config.Config) config.ExpandOptions {
	// prepare standard ignore clauses for config.ExpandString() etc.
	return config.LookupTable(map[string]string{
		"ignore-all":            ignoreClause(cf),
		"ignore-all-no-plugin":  ignoreClause(cf, slices.DeleteFunc(ignoreColumns(cf), func(e string) bool { return e == "plugin" })...),
		"ignore-gateway-source": ignoreClause(cf, "gateway", "source"),
	})
}

// build an ignore clause ready to use directly after and AND in a
// SELECT. Limit to given column or all available.
//
// The general form is
//
//	strings.Join("[category] NOT IN ${ignores.[category].table}", "AND")
func ignoreClause(cf *config.Config, columns ...string) (ignore string) {
	var e []string
	ignores := ignoreColumns(cf)
	if len(columns) == 0 {
		columns = ignores
	}
	for _, column := range columns {
		if !slices.Contains(ignores, column) {
			log.Error().Msgf("trying to ignore invalid column %s, skipping", column)
			continue
		}
		if cf.IsSet(config.Join("ignore", column+"s", "table")) {
			table := cf.GetString(config.Join("ignore", column+"s", "table"))
			e = append(e, fmt.Sprintf("%s NOT IN %s", column, table))
		} else {
			log.Debug().Msgf("cannot locate %s as an ignore in config, skipping", column+"s")
		}
	}
	if len(e) > 0 {
		return strings.Join(e, " AND ")
	}
	return "1"
}
