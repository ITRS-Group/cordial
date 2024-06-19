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

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

// loadGroupings
func loadGroupings(ctx context.Context, cf *config.Config, tx *sql.Tx) error {
OUTER:
	for grouping := range cf.GetStringMap("groupings") {
		var r io.Reader
		source := cf.GetString(config.Join("groupings", grouping, "source"))
		content := cf.GetString(config.Join("groupings", grouping, "content"))

		if source == "" && content == "" {
			log.Debug().Msgf("no groupings for %s, skipping", grouping)
			continue
		}

		log.Debug().Msgf("trying to load groupings from %s", source)
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

		table, err := cf.ExpandRawString(config.Join("groupings", grouping, "table"))
		if err != nil {
			log.Error().Err(err).Msg(config.Join("groupings", grouping, "table"))
			continue
		}

		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s;", table)); err != nil {
			log.Info().Err(err).Msgf("delete from %s failed", table)
			// NOT an error
			err = nil
		}

		insertStmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s VALUES (@grouping, @glob);", table))
		if err != nil {
			log.Error().Err(err).Msgf("prepare for %s failed", table)
			continue
		}
		defer insertStmt.Close()

		c := csv.NewReader(r)
		c.ReuseRecord = true
		c.FieldsPerRecord = -1
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
			if len(fields) < 2 {
				line, _ := c.FieldPos(0)
				log.Debug().Msgf("source: line %d has an incorrect format, it should be 'name,pattern'", line)
				continue
			}
			if _, err = insertStmt.ExecContext(ctx, sql.Named("grouping", fields[0]), sql.Named("glob", fields[1])); err != nil {
				log.Error().Err(err).Msgf("insert for %s failed", table)
				continue OUTER
			}
			lines++
		}

		log.Debug().Msgf("read %d lines from %s and added to %s", lines, source, table)
	}
	return nil
}
