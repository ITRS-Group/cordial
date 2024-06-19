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
	"fmt"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
)

func loadPluginTables(ctx context.Context, cf *config.Config, tx *sql.Tx) (err error) {
	for pluginTable := range cf.GetStringMap("plugins") {
		if cf.IsSet(config.Join("plugins", pluginTable, "plugins")) {
			table := cf.GetString(config.Join("plugins", pluginTable, "table"))
			var stmt *sql.Stmt
			if stmt, err = tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %q VALUES (?);", table)); err != nil {
				return
			}
			plugins := cf.GetStringSlice(config.Join("plugins", pluginTable, "plugins"))
			for _, p := range plugins {
				if _, err = stmt.ExecContext(ctx, p); err != nil {
					log.Error().Err(err).Msg("insert failed")
					return
				}
			}
			stmt.Close()
		}
	}

	return
}
