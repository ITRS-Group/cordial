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
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	if strings.HasPrefix(dsn, "file:") {
		// check and replace short form home in a file DSN
		dsn = "file:" + config.ExpandHome(strings.TrimPrefix(dsn, "file:"))
	}
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

		log.Info().Msgf("checked: %q -> %d", checkQuery, checked)
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

			if err = tx.Commit(); err != nil {
				return
			}
			// flag an update has happened, force reload of file sources later on, which then clears it
			cf.Set("db.updated", true)
			log.Debug().Msgf("completed update to version %d", i)
		} else {
			log.Debug().Msgf("update %d not required", i)
		}

		// update user_version
		log.Debug().Msgf("set user_version=%d", i)
		if _, err = db.ExecContext(ctx, fmt.Sprintf(userVersionUpdate, i)); err != nil {
			return
		}
	}

	return
}

// queryToTable returns the result of running query, inside the
// transaction tx, as a table of data as a two dimensional slice of
// strings. The first row is always the column names and should be
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

// detailReportToDB reads lines from a csv.Reader c and based on the
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
func detailReportToDB(ctx context.Context, cf *config.Config, tx *sql.Tx, c *csv.Reader, source, sourceType, sourcePath string, sourceTimestamp time.Time) (err error) {
	var licdExtended bool
	isoTime := sourceTimestamp.UTC().Format(time.RFC3339)

	// read column names
	colNames, err := c.Read()
	if err == io.EOF {
		return
	}
	// remember to clone colNames as reuserecords is in use
	colNames = slices.Clone(colNames)

	if len(colNames) >= 22 {
		// reporting file or licd >= 7.0.0, new columns available
		licdExtended = true
	}

	// column names in the old output includes space separators
	for i, c := range colNames {
		colNames[i] = strings.TrimSpace(c)
	}

	columns := map[string]int{}
	for i, c := range colNames {
		columns[strings.ToLower(c)] = i
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

		if len(fields) != len(colNames) {
			panic("number of fields in CSV differs from heading for " + source)
		}

		values := make(map[string]string, len(colNames))
		for i, c := range colNames {
			values[strings.ToLower(c)] = strings.TrimSpace(fields[i])
		}

		var host_name, port, token_id, host_id string
		individual := sql.NullBool{
			Valid: false,
			Bool:  true,
		}

		if licdExtended {
			host_name, port, host_id = values["host_name"], values["port"], values["host_id"]
			token_id = host_id
			if strings.Contains(values["description"], "[INDIVIDUAL]") {
				individual.Valid = true
				token_id = "INDIVIDUAL"
			}
		} else if len(values["description"]) > 0 {
			matches := re.FindStringSubmatch(values["description"])
			if len(matches) == 4 {
				host_name, port, host_id = matches[1], matches[2], matches[3]
				token_id = host_id
				if host_id == "[INDIVIDUAL]" {
					individual.Valid = true
					token_id = "INDIVIDUAL"
				}
				// line, col := c.FieldPos(columns["Description"])
				// return fmt.Errorf("only found %d matches in 'description' column at line %d, column %d: %q", len(matches), line, col, source)
			}
		}

		switch values["component"] {
		case "binary":
			// 5,gateway:LOB_GATEWAY_1102,gateway:LOB_GATEWAY_1102,binary,netprobe,itrsrh6005160:7036 [240a1690],1
			if values["item"] != "netprobe" {
				line, col := c.FieldPos(4)
				return fmt.Errorf("unknown binary type %q at line %d, column %d: %q", values["item"], line, col, source)
			}

			_, err = probesInsertStmt.ExecContext(ctx,
				sql.Named("gateway", strings.TrimPrefix(values["requestingcomponent"], "gateway:")),
				sql.Named("probeName", host_name),
				sql.Named("probePort", port),
				sql.Named("tokenID", token_id),
				sql.Named("time", isoTime),
				sql.Named("source", source),
				sql.Named("os", colOrNull("os", columns, fields)),
				sql.Named("version", colOrNull("version", columns, fields)),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting probe")
			}

		case "plugin":
			// 21,gateway:LOB_GATEWAY_1102,gateway:LOB_GATEWAY_1102,binary,netprobe,itrsrh002111:7036 [410a355f],1
			_, err = samplersInsertStmt.ExecContext(ctx,
				sql.Named("gateway", strings.TrimPrefix(values["requestingcomponent"], "gateway:")),
				sql.Named("plugin", values["item"]),
				sql.Named("probeName", host_name),
				sql.Named("probePort", port),
				sql.Named("tokenID", token_id),
				sql.Named("number", values["number"]),
				sql.Named("individual", individual),
				sql.Named("time", isoTime),
				sql.Named("source", source),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting sampler")
			}

		case "ca_plugin":
			// 2851,gateway:LOB_GATEWAY_1085,gateway:LOB_GATEWAY_1085,ca_plugin,prometheus-plugin,position-cd-transformer-c1-6-gem,1
			var entity string
			if licdExtended {
				entity = values["managed_entity"]
			} else {
				entity = values["description"]
			}
			_, err = caSamplersInsertStmt.ExecContext(ctx,
				sql.Named("gateway", strings.TrimPrefix(values["requestingcomponent"], "gateway:")),
				sql.Named("plugin", values["item"]),
				sql.Named("entity", entity),
				sql.Named("probeName", colOrNull("host_name", columns, fields)),
				sql.Named("probePort", colOrNull("port", columns, fields)),
				sql.Named("tokenID", colOrNull("host_id", columns, fields)),
				sql.Named("number", values["number"]),
				sql.Named("time", isoTime),
				sql.Named("source", source),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting CA sampler")
			}

		case "gateway_component":
			// 1,gateway:LOB_GATEWAY_1102,gateway:LOB_GATEWAY_1102,gateway_component,database-logging,,1
			_, err = gwComponentsInsertStmt.ExecContext(ctx,
				sql.Named("gateway", strings.TrimPrefix(values["requestingcomponent"], "gateway:")),
				sql.Named("component", values["item"]),
				sql.Named("number", values["number"]),
				sql.Named("time", isoTime),
				sql.Named("source", source),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting gateway component")
			}

			// also add gateways directly to a gateways table
			if fields[4] == "gateway" {
				_, err = gatewaysInsertStmt.ExecContext(ctx,
					sql.Named("gateway", strings.TrimPrefix(values["requestingcomponent"], "gateway:")),
					sql.Named("host", colOrNull("gateway_host", columns, fields)),
					sql.Named("port", colOrNull("gateway_port", columns, fields)),
					sql.Named("version", colOrNull("version", columns, fields)),
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
				sql.Named("gateway", strings.TrimPrefix(values["requestingcomponent"], "gateway:")),
				sql.Named("plugin", values["item"]),
				sql.Named("number", values["number"]),
				sql.Named("time", isoTime),
				sql.Named("source", source),
			)
			if err != nil {
				log.Error().Err(err).Msg("inserting gateway sampler")
			}

		default:
			// error
			line, col := c.FieldPos(1)
			log.Error().Msgf("ignoring unknown entry %q on line %d column %d in %s", values["component"], line, col, source)
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

func summaryReportToDB(ctx context.Context, cf *config.Config, tx *sql.Tx, c *csv.Reader, source, sourceType, sourcePath string, sourceTimestamp time.Time) (err error) {
	var expiry time.Time
	var mode, licencename, hostname, hostid string

	var row []string
	isoTime := sourceTimestamp.UTC().Format(time.RFC3339)

	for {
		// consumes lines as name/value pairs until more columns show up, e.g. the "Group,..." as CSV header
		row, err = c.Read()
		if err == io.EOF {
			return
		}
		if len(row) > 2 {
			break
		}
		if len(row) != 2 {
			return os.ErrInvalid
		}

		switch strings.ToLower(row[0]) {
		case "expiry":
			expiry, err = time.Parse("02 January 2006", row[1])
			if err != nil {
				log.Error().Err(err).Msgf("license expiry date %q", row[1])
			}
		case "mode":
			mode = strings.ToLower(row[1])
		case "licencename":
			licencename = row[1]
		case "hostname":
			hostname = row[1]
		case "hostid":
			hostid = row[1]
		default:
			// ignore
		}
	}

	// don't record token information for monitor mode, just metadata
	if mode == "monitoring" {
		// update sources with extra info
		return execSQL(ctx, cf, tx, "db.sources-licence", "insert", nil,
			sql.Named("source", source),
			sql.Named("licenceexpiry", expiry),
			sql.Named("licencemode", mode),
			sql.Named("licencename", licencename),
			sql.Named("hostname", hostname),
			sql.Named("hostid", hostid),
		)
	}

	// at this point row should contain the column headings

	columns := map[string]int{}
	for i, c := range row {
		columns[strings.ToLower(c)] = i
	}

	groupsIndex, ok := columns["group"]
	if !ok {
		log.Error().Msgf("cannot find `groups` columns in summary report for %s", source)
		return
	}
	tokenIndex, ok := columns["token"]
	if !ok {
		log.Error().Msgf("cannot find `token` columns in summary report for %s", source)
		return
	}
	totalIndex, ok := columns["total"]
	if !ok {
		log.Error().Msgf("cannot find `total` columns in summary report for %s", source)
		return
	}
	usedIndex, ok := columns["used"]
	if !ok {
		log.Error().Msgf("cannot find `used` columns in summary report for %s", source)
		return
	}
	freeIndex, ok := columns["free"]
	if !ok {
		log.Error().Msgf("cannot find `free` columns in summary report for %s", source)
		return
	}

	tokensInsertStmt, err := tx.PrepareContext(ctx, cf.GetString("db.tokens.insert"))
	if err != nil {
		log.Error().Msgf("cannot prepare statement: %s\n%s", err, cf.GetString("db.tokens.insert"))
		return
	}
	defer tokensInsertStmt.Close()

	for {
		row, err = c.Read()
		if err == io.EOF {
			break
		}
		if len(row) != len(columns) {
			line, _ := c.FieldPos(1)
			log.Error().Msgf("incorrect column count for %q, line %d", source, line)
		}
		if row[groupsIndex] != "Overall" {
			// we ignore all non "Overall" group lines
			continue
		}

		// update license token table

		_, err = tokensInsertStmt.ExecContext(ctx,
			sql.Named("token", row[tokenIndex]),
			sql.Named("total", sql.NullString{
				Valid:  row[totalIndex] != "Unlimited",
				String: row[totalIndex],
			}),
			sql.Named("used", row[usedIndex]),
			sql.Named("free", sql.NullString{
				Valid:  row[freeIndex] != "Unlimited",
				String: row[freeIndex],
			}),
			sql.Named("time", isoTime),
			sql.Named("source", source),
		)
		if err != nil {
			log.Error().Err(err).Msg("inserting token")
		}
	}

	// update sources with extra info
	return execSQL(ctx, cf, tx, "db.sources-licence", "insert", nil,
		sql.Named("source", source),
		sql.Named("licenceexpiry", expiry),
		sql.Named("licencemode", mode),
		sql.Named("licencename", licencename),
		sql.Named("hostname", hostname),
		sql.Named("hostid", hostid),
	)
}

// colOrNull returns a sql.NullString which is populated if the column
// exists and the value in that column is not an empty string. column is
// the name, columns is a mapping of column names to field offsets and
// fields are the values.
func colOrNull(column string, columns map[string]int, fields []string) (val sql.NullString) {
	if v, ok := columns[column]; ok && fields[v] != "" {
		val = sql.NullString{
			Valid:  true,
			String: fields[v],
		}
	}
	return
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

	// split root on dot. If `root` were empty it would have been caught
	// earlier, so prefixFields is always at least one element long
	prefixFields := strings.FieldsFunc(root, func(r rune) bool { return r == '.' })

	log.Debug().Msgf("called with root %q and selector %q", root, createSelector)

	// the `root` could be a config item which is either a slice of
	// strings with named config items, which are self-contained table
	// configurations or a config item that contains multiple table
	// configurations, by name
	var createQueries []string
	switch r := cf.Get(root).(type) {
	case []any:
		// an indirect list of table names
		for _, t := range r {
			// if `root` is two words or longer (e.g. `db.main-tables`)
			// then replace the last component (`main-tables`) with the
			// table name, as the convention is that the list of tables
			// is at then same level as the table definitions.
			createQueries = append(createQueries, config.Join(append(prefixFields[0:len(prefixFields)-1], fmt.Sprint(t), createSelector)...))
		}
	case map[string]any:
		// a list of config sections with a table name
		for t := range r {
			createQueries = append(createQueries, config.Join(append(prefixFields, fmt.Sprint(t), createSelector)...))
		}
	default:
		err = fmt.Errorf("createTables: unsupported config type %s", root)
		return
	}

	fc := buildFilterSQL(cf)

	for _, table := range createQueries {
		if !cf.IsSet(table) {
			log.Debug().Msgf("table %q not in config, skipping", table)
			continue
		}

		// add a ${values:} prefix that returns SQL that can be combined
		// like this:
		//
		//    WITH ig(gateway) AS (
		//      SELECT 1 WHERE 1 == 0
		//      ${values:db.gateways.filter.include}
		//    ),
		//	  ...
		//
		// to create a CTE that can then be tested with EXISTS/NOT EXISTS like
		// this:
		//
		//    WHERE EXISTS (SELECT 1 FROM ig WHERE ${db.gateways.table}.gateway GLOB ig.gateway)
		//      AND NOT EXISTS (SELECT 1 FROM eg WHERE ${db.gateways.table}.gateway GLOB ig.gateway)
		//
		// The `SELECT 1 WHERE 1 == 0` is needed so that if the
		// ${values} return is empty, so the CTE returns no rows and not
		// an error.
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

		query := cf.GetString(table, fc, values)
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

	if err = createTables(ctx, cf, tx, config.Join("filters", "include"), "create"); err != nil {
		return
	}
	if err = processFilters(ctx, cf, tx, "include"); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, config.Join("filters", "exclude"), "create"); err != nil {
		return
	}
	if err = processFilters(ctx, cf, tx, "exclude"); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, config.Join("filters", "group"), "create"); err != nil {
		return
	}
	if err = processGroups(ctx, cf, tx); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, config.Join("filters", "allocations"), "create"); err != nil {
		return
	}
	if err = processAllocations(ctx, cf, tx); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, config.Join("db", "main-tables"), "create-active"); err != nil {
		return
	}
	if err = createTables(ctx, cf, tx, config.Join("db", "main-tables"), "create-inactive"); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, "plugins", "create"); err != nil {
		return
	}

	if err = loadPluginTables(ctx, cf, tx); err != nil {
		return
	}

	if err = createTables(ctx, cf, tx, config.Join("db", "report-tables"), "create"); err != nil {
		return
	}

	return execSQL(ctx, cf, tx, config.Join("db", "reporting-updates"), "update", nil)
}

// execSQL is a simple wrapper to run ExecContext for the transaction tx
// and query found in cf under `root`.`queryName` passing the arguments
// in args. Any error is returned.
func execSQL(ctx context.Context, cf *config.Config, tx *sql.Tx, root, queryName string, lookupTable map[string]string, args ...any) (err error) {
	query := cf.GetString(config.Join(root, queryName), config.LookupTable(lookupTable))
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		log.Error().Err(err).Msg(query)
	}
	return
}

// readLicdReports called a function with a io.ReadCloser to read the
// contents and process/load them. The caller must close the reader.
// Support for http/https/file and plain paths as well as "~/" prefix to
// mean home directory.
func readLicdReports(ctx context.Context, cf *config.Config, tx *sql.Tx, source string) (sources []string, err error) {
	source = config.ExpandHome(source)
	u, err := url.Parse(source)
	if err != nil {
		return
	}

	dbUpdated := cf.GetBool("db.updated")

	switch u.Scheme {
	case "https", "http":
		sources = append(sources, u.Scheme+":"+u.Hostname())
		t := time.Now()
		tr := http.DefaultTransport
		if u.Scheme == "https" {
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

			tr = &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:            roots,
					InsecureSkipVerify: skip,
				},
			}
		}

		client := &http.Client{Transport: tr, Timeout: cf.GetDuration("gdna.licd-timeout")}

		// read summary data
		uSummary := u.JoinPath(SummaryPath)
		req, err := http.NewRequestWithContext(ctx, "GET", uSummary.String(), nil)
		if err != nil {
			updateSources(ctx, cf, tx, u.Scheme+":"+uSummary.Hostname(), u.Scheme, source, false, t, err)
			return sources, err
		}
		resp, err := client.Do(req)
		if err != nil {
			updateSources(ctx, cf, tx, u.Scheme+":"+uSummary.Hostname(), u.Scheme, source, false, t, err)
			return sources, err
		}
		if resp.StatusCode > 299 {
			resp.Body.Close()
			err = fmt.Errorf("server returned %s", resp.Status)
			updateSources(ctx, cf, tx, u.Scheme+":"+uSummary.Hostname(), u.Scheme, source, false, t, err)
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
		c.ReuseRecord = false
		c.Comment = '#'

		if err = summaryReportToDB(ctx, cf, tx, c, u.Scheme+":"+uSummary.Hostname(), u.Scheme, source, t); err != nil {
			log.Error().Err(err).Msg("")
			updateSources(ctx, cf, tx, u.Scheme+":"+uSummary.Hostname(), u.Scheme, source, false, t, err)
			return sources, err
		}

		// read detail data
		uDetail := u.JoinPath(DetailsPath)
		req, err = http.NewRequestWithContext(ctx, "GET", uDetail.String(), nil)
		if err != nil {
			updateSources(ctx, cf, tx, u.Scheme+":"+uDetail.Hostname(), u.Scheme, source, false, t, err)
			return sources, err
		}
		resp, err = client.Do(req)
		if err != nil {
			updateSources(ctx, cf, tx, u.Scheme+":"+uDetail.Hostname(), u.Scheme, source, false, t, err)
			return sources, err
		}
		if resp.StatusCode > 299 {
			resp.Body.Close()
			err = fmt.Errorf("server returned %s", resp.Status)
			updateSources(ctx, cf, tx, u.Scheme+":"+uDetail.Hostname(), u.Scheme, source, false, t, err)
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

		c = csv.NewReader(resp.Body)
		c.ReuseRecord = true
		c.Comment = '#'

		if err = detailReportToDB(ctx, cf, tx, c, u.Scheme+":"+uDetail.Hostname(), u.Scheme, source, t); err != nil {
			updateSources(ctx, cf, tx, u.Scheme+":"+uDetail.Hostname(), u.Scheme, source, false, t, err)
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

		// any file with the suffix `_summary` before the extension is
		// loaded as a license summary
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

			if strings.HasSuffix(strings.TrimSuffix(sourceName, filepath.Ext(sourceName)), "_summary") {
				sn := strings.TrimSuffix(sourceName, "_summary"+filepath.Ext(sourceName)) + filepath.Ext(sourceName)
				if err = summaryReportToDB(ctx, cf, tx, c, sn, "file", source, t); err != nil {
					log.Error().Err(err).Msg("")
					updateSources(ctx, cf, tx, sn, "file", source, false, t, err)
					return sources, err
				}
			} else {
				if err = detailReportToDB(ctx, cf, tx, c, sourceName, "file", source, t); err != nil {
					log.Error().Err(err).Msg(source)
					// record error
					updateSources(ctx, cf, tx, sourceName, "file", source, false, t, err)
				}
			}
		}
		return sources, nil
	}

	return sources, nil
}

func readSummaryReport() {

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
