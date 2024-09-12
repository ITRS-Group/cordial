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
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"time"

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
		if s, err = readLicenseReports(ctx, cf, tx, source, licenseReportToDB); err != nil {
			log.Error().Err(err).Msgf("readLicenseReports for %s failed", source)
		}
		sources = append(sources, s...)
	}

	// for all file sources in the sources table also check if they still exist

	sourceFilesQuery := cf.GetString("db.sources.invalid-files")
	log.Trace().Msgf("query:\n%s", sourceFilesQuery)
	rows, err := tx.QueryContext(ctx, sourceFilesQuery)
	if err != nil {
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
		if s, err = readLicdReports(ctx, cf, tx, source, licenseReportToDB); err != nil {
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

func updateSources(ctx context.Context, cf *config.Config, tx *sql.Tx, source, sourceType, path string, valid bool, t time.Time, status any) error {
	isoTime := t.UTC().Format(time.RFC3339)
	// is source is an error then unwrap it as prefix with a plain
	// "ERROR:"
	s, ok := status.(error)
	if ok {
		status = fmt.Errorf("ERROR: %w", errors.Unwrap(s))
	}
	return execSQL(ctx, cf, tx, "db.sources", "insert", nil,
		sql.Named("source", source),
		sql.Named("sourceType", sourceType),
		sql.Named("path", path),
		sql.Named("firstSeen", isoTime),
		sql.Named("lastSeen", isoTime),
		sql.Named("status", fmt.Sprint(status)),
		sql.Named("valid", valid),
	)
}

// readLicenseReports called a function with a io.ReadCloser to read the
// contents and process/load them. The caller must close the reader.
// Support for http/https/file and plain paths as well as "~/" prefix to
// mean home directory.
func readLicenseReports(ctx context.Context, cf *config.Config, tx *sql.Tx, source string,
	fn func(context.Context, *config.Config, *sql.Tx, *csv.Reader, string, string, string, time.Time) error) (sources []string, err error) {
	source = config.ExpandHome(source)
	u, err := url.Parse(source)
	if err != nil {
		return
	}

	dbUpdated := cf.GetBool("db.updated")

	switch u.Scheme {
	case "https":
		sources = append(sources, "https:"+u.Hostname())
		t := time.Now()
		skip := cf.GetBool("gdna.licd-skip-verify")
		roots, err := x509.SystemCertPool()
		if err != nil {
			log.Warn().Err(err).Msg("cannot read system certificates, continuing anyway")
		}

		if !skip {
			if chainfile := cf.GetString("gdna.licd-chain"); chainfile != "" {
				if chainbytes, err := os.ReadFile(chainfile); err != nil {
					log.Warn().Err(err).Msg("cannot read licd certificate chain, continuing with system certificates only")
				} else {
					roots.AppendCertsFromPEM(chainbytes) // ignore ok/not ok
				}
			}
		}

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            roots,
				InsecureSkipVerify: skip,
			},
		}
		client := &http.Client{Transport: tr, Timeout: cf.GetDuration("gdna.licd-timeout")}
		u = u.JoinPath(DetailsPath)
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			updateSources(ctx, cf, tx, "https:"+u.Hostname(), "https", source, false, t, err)
			return sources, err
		}
		resp, err := client.Do(req)
		if err != nil {
			updateSources(ctx, cf, tx, "https:"+u.Hostname(), "https", source, false, t, err)
			return sources, err
		}
		if resp.StatusCode > 299 {
			resp.Body.Close()
			err = fmt.Errorf("server returned %s", resp.Status)
			updateSources(ctx, cf, tx, "https:"+u.Hostname(), "https", source, false, t, err)
			return sources, err
		}
		defer resp.Body.Close()

		// set the source time to either the last-modified header or now
		if lm := resp.Header.Get("last-modified"); lm != "" {
			lmt, err := http.ParseTime(lm)
			if err == nil {
				t = lmt
			}
		}

		c := csv.NewReader(resp.Body)
		c.ReuseRecord = true
		c.Comment = '#'

		if err = fn(ctx, cf, tx, c, "https:"+u.Hostname(), "https", source, t); err != nil {
			updateSources(ctx, cf, tx, "https:"+u.Hostname(), "https", source, false, t, err)
			return sources, err
		}
	case "http":
		sources = append(sources, "http:"+u.Hostname())
		t := time.Now()
		client := &http.Client{Timeout: cf.GetDuration("gdna.licd-timeout")}
		u = u.JoinPath(DetailsPath)
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			updateSources(ctx, cf, tx, "http:"+u.Hostname(), "http", source, false, t, err)
			return sources, err
		}
		resp, err := client.Do(req)
		if err != nil {
			updateSources(ctx, cf, tx, "http:"+u.Hostname(), "http", source, false, t, err)
			return sources, err
		}
		if resp.StatusCode > 299 {
			resp.Body.Close()
			err = fmt.Errorf("server returned %s", resp.Status)
			updateSources(ctx, cf, tx, "http:"+u.Hostname(), "http", source, false, t, err)
			return sources, err
		}
		defer resp.Body.Close()

		// set the source time to either the last-modified header or now
		if lm := resp.Header.Get("last-modified"); lm != "" {
			lmt, err := http.ParseTime(lm)
			if err == nil {
				t = lmt
			}
		}

		c := csv.NewReader(resp.Body)
		c.ReuseRecord = true
		c.Comment = '#'

		if err = fn(ctx, cf, tx, c, "http:"+u.Hostname(), "http", source, t); err != nil {
			updateSources(ctx, cf, tx, "http:"+u.Hostname(), "http", source, false, t, err)
			return sources, err
		}
	default:
		log.Debug().Msgf("looking for files matching '%s'", source)

		files, err := filepath.Glob(source)
		if err != nil {
			return sources, err
		}

		if len(files) == 0 {
			log.Warn().Msgf("no matches for %s", source)
			return sources, nil
		}

		for _, source := range files {
			var s os.FileInfo
			t := time.Now()
			source, _ = filepath.Abs(source)
			source = filepath.ToSlash(source)

			sourceName := "file:" + strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
			sources = append(sources, sourceName)
			s, err = os.Stat(source)
			if err != nil {
				log.Error().Err(err).Msg("")
				// record the failure
				updateSources(ctx, cf, tx, sourceName, "file", source, false, t, err)
				return sources, err
			}
			if s.IsDir() {
				// record failure as source file is a directory
				updateSources(ctx, cf, tx, sourceName, "file", source, false, t, os.ErrInvalid)
				return sources, os.ErrInvalid // geneos.ErrIsADirectory
			}
			if !overrideFiletime {
				t = s.ModTime()
			}

			var tm sql.NullString
			query := cf.ExpandString(`SELECT lastSeen FROM ${db.sources.table} WHERE source = ?`)
			r1 := tx.QueryRowContext(ctx, query, sourceName)
			if err := r1.Scan(&tm); err != nil {
				log.Debug().Err(err).Msgf("no data for query %s (source '%s')", query, sourceName)
			}
			if tm.Valid {
				last, err := time.Parse(time.RFC3339, tm.String)
				if err != nil {
					log.Error().Err(err).Msgf("parse time failed for %s", tm.String)
					// drop through, time is nil
				}

				if t.Truncate(time.Second).Equal(last) && !(onStart || onStartEMail) && !dbUpdated {
					log.Debug().Msgf("no update since %s", tm.String)
					continue
				}
			}

			r, err := os.Open(source)
			if err != nil {
				log.Error().Err(err).Msg(source)
				continue
			}
			defer r.Close()
			c := csv.NewReader(r)
			c.ReuseRecord = true
			c.Comment = '#'
			if err = fn(ctx, cf, tx, c, sourceName, "file", source, t); err != nil {
				log.Error().Err(err).Msg(source)
				// record error
				updateSources(ctx, cf, tx, sourceName, "file", source, false, t, err)
			}
		}
		return sources, nil
	}

	return sources, nil
}
