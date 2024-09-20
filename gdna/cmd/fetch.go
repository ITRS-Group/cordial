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
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
)

//go:embed _docs/fetch.md
var fetchCmdDescription string

var overrideFiletime, postProcess bool

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch usage data",
	Long:  fetchCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// Handle SIGINT (CTRL+C) gracefully.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		db, err := openDB(ctx, cf, "db.dsn", false)
		if err != nil {
			return
		}
		defer db.Close()

		log.Debug().Msg("fetching data")
		if _, err = fetch(ctx, cf, db); err != nil {
			return
		}

		log.Debug().Msg("closing database")
		_, err = db.ExecContext(ctx, "VACUUM")
		return
	},
}

func init() {
	GDNACmd.AddCommand(fetchCmd)

	fetchCmd.Flags().BoolVarP(&postProcess, "post-process", "p", false, "post process data for reporting database")
	fetchCmd.Flags().BoolVarP(&overrideFiletime, "time", "T", false, "Override file times with the current time (for testing only)")
	// fetchCmd.Flags().StringVarP(&logFile, "logfile", "l", execname+"-fetch.log", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	fetchCmd.Flags().SortFlags = false
}

func fetch(ctx context.Context, cf *config.Config, db *sql.DB) (sources []string, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Error().Msgf("cannot BEGIN transaction: %s", err)
		return
	}
	defer tx.Rollback()

	if err = createTables(ctx, cf, tx, "db.main-tables", "create"); err != nil {
		return
	}

	log.Debug().Msgf("licd-sources: %v", cf.GetStringSlice("gdna.licd-sources"))
	for _, source := range cf.GetStringSlice("gdna.licd-sources") {
		var s []string
		log.Debug().Msgf("reading from %s", source)
		if s, err = readLicdReports(ctx, cf, tx, source); err != nil {
			log.Error().Err(err).Msgf("readLicenseReports for %s failed", source)
		}
		sources = append(sources, s...)
	}

	// for all file sources in the sources table also check if they still exist

	sourceFilesQuery := cf.GetString("db.sources.invalid-files")
	log.Trace().Msgf("query:\n%s", sourceFilesQuery)
	rows, err := tx.QueryContext(ctx, sourceFilesQuery)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	defer rows.Close()

	var sourcePath string
	for rows.Next() {
		if err = rows.Scan(&sourcePath); err != nil {
			return
		}

		// try to open path
		log.Debug().Msgf("checking: %q", sourcePath)
		f, err := os.Open(sourcePath)
		if err == nil {
			f.Close()
			if err = execSQL(ctx, cf, tx, "db.sources", "update-file-status", nil,
				sql.Named("status", "STALE"),
				sql.Named("path", sourcePath),
			); err != nil {
				return sources, err
			}
			continue
		} else {
			if err = execSQL(ctx, cf, tx, "db.sources", "update-file-status", nil,
				sql.Named("status", fmt.Sprintf("ERROR: %s", errors.Unwrap(err))),
				sql.Named("path", sourcePath),
			); err != nil {
				return sources, err
			}

		}

	}
	rows.Close()

	for _, source := range cf.GetStringSlice("gdna.licd-reports") {
		var s []string
		log.Debug().Msgf("reading licd report file(s): %s", source)
		if s, err = readLicdReportFile(ctx, cf, tx, source); err != nil {
			return
		}
		sources = append(sources, s...)
	}

	slices.Sort(sources)

	if err = runPostInsertHooks(ctx, cf, tx); err != nil {
		return
	}

	if postProcess {
		if err = updateReportingDatabase(ctx, cf, tx, sources); err != nil {
			return
		}
	}

	err = tx.Commit()
	return
}
