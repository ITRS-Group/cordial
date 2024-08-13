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
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

const dbtype = "sqlite3"

// openDB opens the given DSN and returns both a *sql.DB object and a
// ready to go single *sql.Conn object. Remember to close the conn
// first, then the db, else WAL files are left behind.
//
// The single conn connection forces all queries onto one connection to
// avoid SQLite deadlocks
//
// check if a `gdna-version` table already exists and then do version
// specific updates as necessary. currently none, just update version
func openDB(ctx context.Context, cf *config.Config, dsnBase string, readonly bool) (db *sql.DB, err error) {
	dsn := cf.GetString(dsnBase)
	log.Info().Msgf("opening database using DSN `%s`", dsn)
	db, err = sql.Open(dbtype, dsn)
	if err != nil {
		log.Error().Msgf("cannot connect to database `%s`: %s", dsn, err)
		return
	}

	if err = db.PingContext(ctx); err != nil {
		return
	}

	if readonly {
		return
	}

	// if `db.on-open` exists, run it
	onOpen := cf.GetString("db.on-open")
	if onOpen != "" {
		if _, err = db.ExecContext(ctx, onOpen); err != nil {
			return
		}
	}

	if err = updateSchema(ctx, db, cf); err != nil {
		return
	}

	versionQuery := cf.GetString("db.gdna-version.query")
	if versionQuery != "" {
		var version string
		if err = db.QueryRowContext(ctx, versionQuery).Scan(&version); err != nil {
			// assume, poorly, that this is because the table does not
			// yet exist (as the ping and on-open did not error above)
			//
			// create a new version table and update it
			createVersion := cf.GetString("db.gdna-version.create")
			if _, err := db.ExecContext(ctx, createVersion); err != nil {
				return db, err
			}
		}
		insertVersion := cf.GetString("db.gdna-version.insert")
		_, err = db.ExecContext(ctx, insertVersion, sql.Named("version", cordial.VERSION))
		if err != nil {
			log.Error().Err(err).Msg("updating gdna_version")
		}
	}

	return
}

func updateSchema(ctx context.Context, db *sql.DB, cf *config.Config) (err error) {
	// update schema as required
	userVersionQuery := "PRAGMA user_version"
	userVersionUpdate := "PRAGMA user_version = %d"
	var userVersion int64
	if err = db.QueryRowContext(ctx, userVersionQuery).Scan(&userVersion); err != nil {
		return
	}
	log.Debug().Msgf("user_version = %d", userVersion)

	// now look for larger values in config
	for i := userVersion + 1; ; i++ {
		updateBase := config.Join("db", "schema-updates", strconv.FormatInt(i, 10))
		if !cf.IsSet(updateBase) {
			break
		}
		log.Debug().Msgf("found update %d", i)
		checkQuery := cf.GetString(config.Join(updateBase, "check"))
		updateQuery := cf.GetString(config.Join(updateBase, "update"))

		if updateQuery == "" {
			err = fmt.Errorf("no update query for version %d", i)
			return
		}

		var checked int
		if err = db.QueryRowContext(ctx, checkQuery).Scan(&checked); err != nil {
			return
		}

		if checked > 0 {
			var tx *sql.Tx
			tx, err = db.BeginTx(ctx, nil)
			if err != nil {
				return
			}
			defer tx.Rollback()

			log.Trace().Msgf("updateQuery:\n%s", updateQuery)
			if _, err = tx.ExecContext(ctx, updateQuery); err != nil {
				return
			}

			// update user_version
			log.Trace().Msgf("userVersionQuery:\n%s", userVersionQuery)
			if _, err = tx.ExecContext(ctx, fmt.Sprintf(userVersionUpdate, i)); err != nil {
				return
			}

			if err = tx.Commit(); err != nil {
				return
			}
			log.Debug().Msgf("completed update to version %d", i)
		} else {
			log.Debug().Msgf("update %d not required", i)
		}
	}

	return
}

// queryToTable returns the result of running query, inside the
// transaction tx, as a table of data as a two dimensional slice of
// strings. The first row ise always the column names and should be
// discarded if using your own.
func queryToTable(ctx context.Context, tx *sql.Tx, columns []string, query string) (table [][]string, err error) {
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	if len(columns) > 0 {
		table = append(table, columns)
	} else {
		c, err := rows.Columns()
		if err != nil {
			return table, err
		}
		table = append(table, c)
	}

	// prepare a slice of "any" with underlying string pointers for scanning the row
	r1 := make([]any, len(table[0]))
	for i := range table[0] {
		str := ""
		r1[i] = &str
	}

	for rows.Next() {
		r2 := []string{}
		if err = rows.Scan(r1...); err != nil {
			return
		}
		// pull out strings into a string slice
		for _, c := range r1 {
			r2 = append(r2, *(c.(*string)))
		}

		table = append(table, r2)
	}
	err = rows.Err()
	return
}

// queryHeadlines run query, inside the transaction tx, and returns a
// ordered slice of headlines names and a map of those names to values.
func queryHeadlines(ctx context.Context, tx *sql.Tx, query string) (names []string, headlines map[string]string, err error) {
	headlines = make(map[string]string)

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	c, err := rows.Columns()
	if err != nil {
		return
	}
	if len(c) < 2 {
		err = os.ErrInvalid
		return
	}

	// prepare a slice of "any" with underlying string pointers for scanning the row
	r1 := make([]any, len(c))
	for i := range len(c) {
		str := ""
		r1[i] = &str
	}

	for rows.Next() {
		r2 := []string{}
		if err = rows.Scan(r1...); err != nil {
			return
		}
		// pull out strings into a string slice
		for _, c := range r1 {
			r2 = append(r2, *(c.(*string)))
		}

		names = append(names, r2[0])
		headlines[r2[0]] = r2[1]
	}
	err = rows.Err()
	return
}

// licenseReportToDB reads lines from a csv.Reader c and based on the
// contents transforms and inserts them into the database using the
// active transaction tx. The format of the fields in the csv file are
// from the detail report documented here:
// <https://docs.itrsgroup.com/docs/geneos/current/administration/licence-daemon/index.html#csv-files>
//
// The source, sourceType, sourcePath and sourceTimestamp parameters are
// populated by the calling functions that access the original data.
// source is a unique and long lived label for the data source,
// sourceType is one of "http", "https", "file" or "licd". sourcePath is
// the original path or URL to the data and sourceTimestamp is the best
// value of the time the data was generated - for files this is the
// modification time while for http/https is either the `Last-Modified`
// header value or the time of the request.
func licenseReportToDB(ctx context.Context, cf *config.Config, tx *sql.Tx, c *csv.Reader, source, sourceType, sourcePath string, sourceTimestamp time.Time) (err error) {
	isoTime := sourceTimestamp.UTC().Format(time.RFC3339)

	// read column names
	colNames, err := c.Read()
	if err == io.EOF {
		return
	}
	columns := map[string]int{}
	for i, c := range colNames {
		columns[c] = i
	}

	gatewaysInsertStmt, err := tx.PrepareContext(ctx, cf.GetString("db.gateways.insert"))
	if err != nil {
		log.Error().Msgf("cannot prepare statement: %s\n%s", err, cf.GetString("db.gateways.insert"))
		return
	}
	defer gatewaysInsertStmt.Close()
	probesInsertStmt, err := tx.PrepareContext(ctx, cf.GetString("db.probes.insert"))
	if err != nil {
		log.Error().Msgf("cannot prepare statement: %s", err)
		return
	}
	defer probesInsertStmt.Close()

	samplersInsertStmt, err := tx.PrepareContext(ctx, cf.GetString("db.samplers.insert"))
	if err != nil {
		log.Error().Msgf("cannot prepare statement: %s", err)
		return
	}
	defer samplersInsertStmt.Close()

	caSamplersInsertStmt, err := tx.PrepareContext(ctx, cf.GetString("db.ca-samplers.insert"))
	if err != nil {
		log.Error().Msgf("cannot prepare statement: %s", err)
		return
	}
	defer caSamplersInsertStmt.Close()

	gwSamplersInsertStmt, err := tx.PrepareContext(ctx, cf.GetString("db.gw-samplers.insert"))
	if err != nil {
		log.Error().Msgf("cannot prepare statement: %s", err)
		return
	}
	defer gwSamplersInsertStmt.Close()

	gwComponentsInsertStmt, err := tx.PrepareContext(ctx, cf.GetString("db.gw-components.insert"))
	if err != nil {
		log.Error().Msgf("cannot prepare statement: %s", err)
		return
	}
	defer gwComponentsInsertStmt.Close()

	// transform and insert
	re := regexp.MustCompile(`^(.+?):(\d+)\s\[(.*)\]$`)

	for {
		fields, err := c.Read()
		if err == io.EOF {
			// all good, end of CSV input, just return
			break
		}
		if err != nil {
			return err
		}

		switch fields[3] {
		case "binary":
			// 5,gateway:LOB_GATEWAY_1102,gateway:LOB_GATEWAY_1102,binary,netprobe,itrsrh6005160:7036 [240a1690],1
			if fields[4] != "netprobe" {
				line, col := c.FieldPos(4)
				return fmt.Errorf("unknown binary type %q at line %d, column %d: %q", fields[4], line, col, source)
			}
			matches := re.FindStringSubmatch(fields[5])
			if len(matches) != 4 {
				line, col := c.FieldPos(5)
				return fmt.Errorf("only found %d at line %d, column %d: %q", len(matches), line, col, source)
			}

			// set os and version if they are available, else NULL
			os := sql.NullString{}
			if v, ok := columns["os"]; ok {
				os = sql.NullString{
					Valid:  true,
					String: fields[v],
				}
			}
			version := sql.NullString{}
			if v, ok := columns["version"]; ok {
				version = sql.NullString{
					Valid:  true,
					String: fields[v],
				}
			}

			_, err = probesInsertStmt.ExecContext(ctx,
				sql.Named("gateway", strings.TrimPrefix(fields[2], "gateway:")),
				sql.Named("probeName", matches[1]),
				sql.Named("probePort", matches[2]),
				sql.Named("tokenID", matches[3]),
				sql.Named("time", isoTime),
				sql.Named("source", source),
				sql.Named("os", os),
				sql.Named("version", version),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting probe")
			}

		case "plugin":
			// 21,gateway:LOB_GATEWAY_1102,gateway:LOB_GATEWAY_1102,binary,netprobe,itrsrh002111:7036 [410a355f],1
			matches := re.FindStringSubmatch(fields[5])
			if len(matches) != 4 {
				line, col := c.FieldPos(5)
				return fmt.Errorf("only found %d at line %d, column %d: %q", len(matches), line, col, source)
			}

			_, err = samplersInsertStmt.ExecContext(ctx,
				sql.Named("gateway", strings.TrimPrefix(fields[2], "gateway:")),
				sql.Named("plugin", fields[4]),
				sql.Named("probeName", matches[1]),
				sql.Named("probePort", matches[2]),
				sql.Named("tokenID", matches[3]),
				sql.Named("number", fields[6]),
				sql.Named("time", isoTime),
				sql.Named("source", source),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting sampler")
			}

		case "ca_plugin":
			// 2851,gateway:LOB_GATEWAY_1085,gateway:LOB_GATEWAY_1085,ca_plugin,prometheus-plugin,position-cd-transformer-c1-6-gem,1
			_, err = caSamplersInsertStmt.ExecContext(ctx,
				sql.Named("gateway", strings.TrimPrefix(fields[2], "gateway:")),
				sql.Named("plugin", fields[4]),
				sql.Named("entity", fields[5]),
				sql.Named("number", fields[6]),
				sql.Named("time", isoTime),
				sql.Named("source", source),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting CA sampler")
			}

		case "gateway_component":
			// 1,gateway:LOB_GATEWAY_1102,gateway:LOB_GATEWAY_1102,gateway_component,database-logging,,1
			_, err = gwComponentsInsertStmt.ExecContext(ctx,
				sql.Named("gateway", strings.TrimPrefix(fields[2], "gateway:")),
				sql.Named("component", fields[4]),
				sql.Named("number", fields[6]),
				sql.Named("time", isoTime),
				sql.Named("source", source),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting gateway component")
			}

			// also add gateways directly to a gateways table
			if fields[4] == "gateway" {
				_, err = gatewaysInsertStmt.ExecContext(ctx,
					sql.Named("gateway", strings.TrimPrefix(fields[2], "gateway:")),
					sql.Named("time", isoTime),
					sql.Named("source", source),
				)
				if err != nil {
					log.Error().Err(err).Msg("inserting gateway")
				}
			}

		case "gateway-plugin":
			// 8188,EQ,gateway:LOB_GATEWAY_1165,gateway-plugin,gateway-breachpredictor,,1
			_, err = gwSamplersInsertStmt.ExecContext(ctx,
				sql.Named("gateway", strings.TrimPrefix(fields[2], "gateway:")),
				sql.Named("plugin", fields[4]),
				sql.Named("number", fields[6]),
				sql.Named("time", isoTime),
				sql.Named("source", source),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting gateway sampler")
			}

		default:
			// error
			line, col := c.FieldPos(1)
			log.Error().Msgf("ignoring unknown entry on line %d column %d in %s", line, col, source)
		}

		if err != nil {
			return err
		}
	}

	valid := true
	if maxAge := cf.GetDuration("gdna.stale-after"); maxAge > 0 && time.Since(sourceTimestamp) > maxAge {
		valid = false
	}

	return updateSources(ctx, cf, tx, source, sourceType, sourcePath, valid, sourceTimestamp, "OK")
}

// createTables iterates over the top-level settings in root and runs
// the query in createSelector under each one.
func createTables(ctx context.Context, cf *config.Config, tx *sql.Tx, root, createSelector string) (err error) {
	if !cf.IsSet(root) {
		err = errors.New("createTables: no root config given")
		return
	}

	if createSelector == "" {
		err = fmt.Errorf("createTable: createSelector for %s empty", root)
		return
	}

	// the `root` could be either a slice of strings with named config
	// items, which are self-contained table configurations or a config
	// item that contains multiple table configurations, by name
	var tables []string
	switch r := cf.Get(root).(type) {
	case []string:
		// Get() should never return a slice of strings, but just in
		// case...
		tables = r
	case []any:
		for _, t := range r {
			tables = append(tables, fmt.Sprint(t))
		}
	case map[string]any:
		for t := range r {
			tables = append(tables, t)
		}
	default:
		err = fmt.Errorf("createTables: unsupported config type %s", root)
		return
	}

	// split root on dot, for use in the loop. If `root` were empty it
	// would have been caught earlier, so prefixFields is always at
	// least one element long
	prefixFields := strings.FieldsFunc(root, func(r rune) bool { return r == '.' })
	ignore := ignores(cf)

	for _, table := range tables {
		var q string
		var l []string

		if len(prefixFields) == 1 {
			// if `root` is one word (e.g. `plugins`) then use it as a
			// prefix to the table and create values
			l = append(prefixFields, table, createSelector)
		} else {
			// if `root` is two words or longer (e.g. `db.main-tables`)
			// then replace the last component (`main-tables`) with the
			// table name, as the convention is that the list of tables
			// is at then same level as the table definitions.
			l = append(prefixFields[0:len(prefixFields)-1], table, createSelector)
		}
		// build a config item for the create statement from the list of
		// levels
		q = config.Join(l...)
		if !cf.IsSet(q) {
			continue
		}

		// add a ${values:} prefix that returns SQL that can be
		// combines like this:
		//
		//    WITH ig(gateway) AS (
		//      SELECT 1 WHERE 1 == 0
		//      ${values:db.gateways.filter.include}
		//    )
		//
		// to create a CTE that can then be tested with EXISTS like
		// this:
		//
		//    WHERE EXISTS (SELECT 1 FROM ig WHERE ${db.gateways.table}.gateway GLOB ig.gateway)
		//
		// The `SELECT 1 WHERE 1 == 0` is needed if the ${values}
		// return is empty, so the CTE returns no rows.
		//
		values := config.Prefix("values", func(cf *config.Config, s string, b bool) (result string, err error) {
			s = strings.TrimPrefix(s, "values:")
			l := cf.GetStringSlice(s)
			if len(l) == 0 {
				return
			}
			m := []string{}
			for _, v := range l {
				m = append(m, "('"+v+"')")
			}
			result = "UNION ALL VALUES " + strings.Join(m, ", ")
			if b {
				result = strings.TrimSpace(result)
			}
			return
		})

		query := cf.GetString(q, ignore, values)
		log.Trace().Msg(query)
		if _, err = tx.ExecContext(ctx, query); err != nil {
			return
		}
	}
	return
}

// runPostInsertHooks is a manual trigger to clean-up the most recent
// data inserted into the data tables. This is not done using triggers
// as some of the clean-up is based on aggregate functions over all the
// data.
func runPostInsertHooks(ctx context.Context, cf *config.Config, tx *sql.Tx) (err error) {
	log.Debug().Msg("running post-insert hooks")

	for _, table := range cf.GetStringSlice("db.main-tables") {
		if postInsertQuery := cf.GetString(config.Join("db", table, "post-insert")); postInsertQuery != "" {
			log.Trace().Msgf("post-insert %s:\n%s", table, postInsertQuery)
			if _, err = tx.ExecContext(ctx, postInsertQuery); err != nil {
				log.Error().Err(err).Msgf("post-insert for %s failed", table)
				return err
			}
		}
	}

	return
}

// updateReportingDatabase
func updateReportingDatabase(ctx context.Context, cf *config.Config, tx *sql.Tx, validSources []string) (err error) {
	// update sources `valid` column
	var oldestTime time.Time
	var oldestTimeUnix int64
	maxAge := cf.GetDuration("gdna.stale-after")
	if maxAge != 0 {
		// subtract stale-after from current time for comparison in update
		oldestTime = time.Now().Add(-maxAge)
		oldestTimeUnix = oldestTime.Unix()
	}

	if validSources == nil {
		rows, err := tx.QueryContext(ctx, "SELECT source FROM sources WHERE valid = 1")
		if err != nil {
			return err
		}
		defer rows.Close()
		var source string
		for rows.Next() {
			if err = rows.Scan(&source); err != nil {
				return err
			}
			validSources = append(validSources, source)
		}
	}

	for i, s := range validSources {
		validSources[i] = "'" + s + "'"
	}

	s := strings.Join(validSources, ", ")

	if err = execSQL(ctx, cf, tx, "db.sources", "update-valid", map[string]string{"sources": s},
		sql.Named("oldestValidTime", oldestTimeUnix),
		sql.Named("oldestValueTimeISO", oldestTime.UTC().Format(time.RFC3339)),
		sql.Named("maxAge", maxAge.String()),
	); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, "ignore", "create"); err != nil {
		return
	}
	if err = loadIgnores(ctx, cf, tx); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, "groupings", "create"); err != nil {
		return
	}
	if err = loadGroupings(ctx, cf, tx); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, "db.main-tables", "create-active"); err != nil {
		return
	}
	if err = createTables(ctx, cf, tx, "db.main-tables", "create-inactive"); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, "plugins", "create"); err != nil {
		return
	}

	if err = loadPluginTables(ctx, cf, tx); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, "db.report-tables", "create"); err != nil {
		return
	}

	return execSQL(ctx, cf, tx, "db.reporting-updates", "update", nil)
}

// execSQL is a simple wrapper to run ExecContext for the transaction tx
// and query found in cf under `root`.`queryName` passing the arguments
// in args. Any error is returned.
func execSQL(ctx context.Context, cf *config.Config, tx *sql.Tx, root, queryName string, lookupTable map[string]string, args ...any) (err error) {
	query := cf.GetString(config.Join(root, queryName), config.LookupTable(lookupTable))
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		log.Debug().Err(err).Msg(query)
	}
	return
}
