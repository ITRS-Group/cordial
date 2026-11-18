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
	"log/slog"
	"maps"
	"slices"

	"github.com/itrs-group/cordial/pkg/config"
)

func loadPluginTables(ctx context.Context, cf *config.Config, tx *sql.Tx) (err error) {
	for _, pluginTable := range slices.Sorted(maps.Keys(config.Get[map[string]any](cf, "plugins"))) {
		if table, ok := config.Lookup[string](cf, cf.Join("plugins", pluginTable, "table")); ok {
			if err = loadPluginTable(ctx, tx, table, config.Get[[]string](cf, config.Join("plugins", pluginTable, "plugins"))); err != nil {
				return
			}
		}
	}

	return
}

// loadPluginTable replaces the contents of table with the plugins
func loadPluginTable(ctx context.Context, tx *sql.Tx, table string, plugins []string) error {
	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %q VALUES (?);", table))
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err = tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %q", table)); err != nil {
		return err
	}
	for _, p := range plugins {
		if _, err = stmt.ExecContext(ctx, p); err != nil {
			log.Error("insert failed", slog.Any("error", err), slog.String("plugin", p), slog.String("table", table))
			return err
		}
	}
	return nil
}
