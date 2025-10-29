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
	"os"
	"os/signal"
	"slices"
	"strings"

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

var fetchCmdSources Sources

func init() {
	GDNACmd.AddCommand(fetchCmd)

	fetchCmd.Flags().BoolVarP(&postProcess, "post-process", "p", false, "post process data for reporting database")
	fetchCmd.Flags().BoolVarP(&overrideFiletime, "time", "T", false, "Override file times with the current time (for testing only)")
	// fetchCmd.Flags().StringVarP(&logFile, "logfile", "l", execname+"-fetch.log", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	fetchCmd.Flags().VarP(&fetchCmdSources, "source", "L", SourcesOptionsText)

	fetchCmd.Flags().SortFlags = false
}

// Sources is a slice of licence data sources
type Sources []string

const SourcesOptionsText = "Override configured licence source.\n(Repeat as required)"

func (i *Sources) String() string {
	return strings.Join(*i, ", ")
}

func (i *Sources) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *Sources) Type() string {
	return "URL | PATH"
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

	if len(fetchCmdSources) == 0 {
		fetchCmdSources = cf.GetStringSlice("gdna.licd-sources")
	}
	log.Debug().Msgf("sources: %v", fetchCmdSources)
	for _, source := range fetchCmdSources {
		var s []string
		log.Debug().Msgf("reading from %s", source)
		if s, err = readLicdReports(ctx, cf, tx, source); err != nil {
			log.Error().Err(err).Msgf("readLicenseReports for %s failed", source)
		}
		sources = append(sources, s...)
	}

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
