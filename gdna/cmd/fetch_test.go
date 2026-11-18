package cmd

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

const (
	// a detail report from a licd before 7.0.0
	testDetailCSV = "req number,requestingcomponent,requestor,component,item,description,number\n" +
		"1,gateway:GW1,gateway:GW1,binary,netprobe,host1:7036 [abc123],1\n" +
		"2,gateway:GW1,gateway:GW1,plugin,cpu,host1:7036 [abc123],3\n"

	testSummaryCSV = "licenceName,Test Licence\nmode,licd\nexpiry,01 January 2030\nhostname,h1\nhostid,id1\n" +
		"Group,Token,Total,Used,Free\nOverall,server,10,3,7\n"
)

// fetchTestConfig returns a configuration, which is also the one the
// commands use, that loads the licd sources given and has a database in
// a directory for the test
func fetchTestConfig(t *testing.T, dir string, sources ...string) *config.Config {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "db:\n  file: %s\ngdna:\n  licd-timeout: 5s\n  licd-sources:\n", filepath.Join(dir, "gdna.sqlite"))
	for _, s := range sources {
		fmt.Fprintf(&b, "    - %q\n", s)
	}
	c := testConfig(t, dir, b.String())
	setGlobals(t, c)
	return c
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func count(t *testing.T, db *sql.DB, query string, args ...any) (n int) {
	t.Helper()
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return
}

// a source with a name that is not a plain word is not a problem for
// the queries that use the names of the sources
func TestFetchSourceWithQuoteInName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "o'brien.csv"), testDetailCSV)
	writeFile(t, filepath.Join(dir, "o'brien_summary.csv"), testSummaryCSV)
	cf := fetchTestConfig(t, dir, filepath.Join(dir, "o'brien.csv"))
	db := testDB(t, cf)

	postProcess = true
	if _, err := fetch(t.Context(), cf, db); err != nil {
		t.Fatal(err)
	}

	if n := count(t, db, "SELECT count(*) FROM sources WHERE source = ? AND valid", "file:o'brien"); n != 1 {
		t.Errorf("valid sources named file:o'brien = %d, want 1", n)
	}
	if n := count(t, db, "SELECT count(*) FROM probes_all"); n != 1 {
		t.Errorf("probes = %d, want 1", n)
	}
}

// the detail report of a file that has not changed is not loaded again,
// unless forced, but the summary report, which can be added later, is
func TestFetchFileSources(t *testing.T) {
	dir := t.TempDir()
	detail := filepath.Join(dir, "licd.csv")
	writeFile(t, detail, testDetailCSV)
	cf := fetchTestConfig(t, dir, detail)
	db := testDB(t, cf)

	fetchOnce := func() {
		t.Helper()
		if _, err := fetch(t.Context(), cf, db); err != nil {
			t.Fatal(err)
		}
	}

	// the first fetch after a schema update loads everything and clears
	// the flag
	sourcesChanged.Store(true)
	fetchOnce()
	if sourcesChanged.Load() {
		t.Error("the flag that forces the file sources to be loaded was not cleared")
	}
	if n := count(t, db, "SELECT count(*) FROM probes_all"); n != 1 {
		t.Fatalf("probes = %d, want 1", n)
	}
	if n := count(t, db, "SELECT count(*) FROM tokens_all"); n != 0 {
		t.Fatalf("tokens = %d, want none without a summary report", n)
	}

	// unchanged, so not loaded again
	if _, err := db.Exec("DELETE FROM probes_all"); err != nil {
		t.Fatal(err)
	}
	fetchOnce()
	if n := count(t, db, "SELECT count(*) FROM probes_all"); n != 0 {
		t.Errorf("probes of an unchanged file loaded again: %d", n)
	}

	// the summary report is added
	writeFile(t, filepath.Join(dir, "licd_summary.csv"), testSummaryCSV)
	fetchOnce()
	if n := count(t, db, "SELECT count(*) FROM tokens_all"); n != 1 {
		t.Errorf("tokens = %d, want 1 from the summary report added after the detail report", n)
	}
	if n := count(t, db, "SELECT count(*) FROM probes_all"); n != 0 {
		t.Errorf("probes of an unchanged file loaded again: %d", n)
	}

	// forced
	sourcesChanged.Store(true)
	fetchOnce()
	if n := count(t, db, "SELECT count(*) FROM probes_all"); n != 1 {
		t.Errorf("probes after the schema was updated = %d, want 1", n)
	}
}

func openFDs(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skip("cannot count the open file descriptors")
	}
	return len(entries)
}

func TestFetchFileSourcesDoNotLeakDescriptors(t *testing.T) {
	dir := t.TempDir()
	detail := filepath.Join(dir, "licd.csv")
	writeFile(t, detail, testDetailCSV)
	writeFile(t, filepath.Join(dir, "licd_summary.csv"), testSummaryCSV)
	cf := fetchTestConfig(t, dir, detail)
	db := testDB(t, cf)

	before := openFDs(t)
	for range 25 {
		sourcesChanged.Store(true)
		if _, err := fetch(t.Context(), cf, db); err != nil {
			t.Fatal(err)
		}
	}
	if after := openFDs(t); after > before+5 {
		t.Errorf("%d file descriptors open after 25 fetches, from %d before", after, before)
	}
}

// licdServer is a licd that serves the reports, counting the
// connections made to it
func licdServer(t *testing.T, tls bool) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/"+SummaryPath, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, testSummaryCSV) })
	mux.HandleFunc("/"+DetailsPath, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, testDetailCSV) })

	var conns atomic.Int32
	srv := httptest.NewUnstartedServer(mux)
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	if tls {
		srv.StartTLS()
	} else {
		srv.Start()
	}
	t.Cleanup(srv.Close)
	return srv, &conns
}

// licds on the same host, with different ports, are different sources
// and the connections to them are shared by all the fetches
func TestFetchHTTPSources(t *testing.T) {
	srv1, conns1 := licdServer(t, false)
	srv2, conns2 := licdServer(t, false)
	cf := fetchTestConfig(t, t.TempDir(), srv1.URL, srv2.URL)
	db := testDB(t, cf)

	for range 3 {
		if _, err := fetch(t.Context(), cf, db); err != nil {
			t.Fatal(err)
		}
	}

	for _, srv := range []*httptest.Server{srv1, srv2} {
		name := "http:" + strings.TrimPrefix(srv.URL, "http://")
		if n := count(t, db, "SELECT count(*) FROM sources WHERE source = ? AND status = 'OK'", name); n != 1 {
			t.Errorf("sources named %q = %d, want 1", name, n)
		}
	}
	if n := count(t, db, "SELECT count(*) FROM sources"); n != 2 {
		t.Errorf("%d sources, want 2", n)
	}
	if c1, c2 := conns1.Load(), conns2.Load(); c1 != 1 || c2 != 1 {
		t.Errorf("connections made to the licds = %d and %d after 3 fetches, want 1 each", c1, c2)
	}

	closeLicdClient()
	if _, err := fetch(t.Context(), cf, db); err != nil {
		t.Fatal(err)
	}
	if c1 := conns1.Load(); c1 != 2 {
		t.Errorf("connections made to the licd = %d after the idle connections were closed, want 2", c1)
	}
}

func TestFetchHTTPSSources(t *testing.T) {
	srv, _ := licdServer(t, true)
	dir := t.TempDir()

	t.Run("skip verify", func(t *testing.T) {
		cf := fetchTestConfig(t, dir, srv.URL)
		config.Set(cf, "gdna.licd-skip-verify", true)
		db := testDB(t, cf)
		if _, err := fetch(t.Context(), cf, db); err != nil {
			t.Fatal(err)
		}
		if n := count(t, db, "SELECT count(*) FROM sources WHERE source = ? AND status = 'OK'", "https:"+strings.TrimPrefix(srv.URL, "https://")); n != 1 {
			t.Errorf("source not loaded")
		}
	})

	t.Run("verified with the chain", func(t *testing.T) {
		dir := t.TempDir()
		chain := filepath.Join(dir, "chain.pem")
		writeFile(t, chain, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})))
		cf := fetchTestConfig(t, dir, srv.URL)
		config.Set(cf, "gdna.licd-chain", chain)
		db := testDB(t, cf)
		if _, err := fetch(t.Context(), cf, db); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not verified", func(t *testing.T) {
		cf := fetchTestConfig(t, t.TempDir(), srv.URL)
		db := testDB(t, cf)
		if _, err := fetch(t.Context(), cf, db); err == nil {
			t.Error("fetch from a server with an unknown certificate succeeded")
		}
	})
}

func TestLicdHTTPClient(t *testing.T) {
	quietLog(t)
	t.Cleanup(closeLicdClient)
	closeLicdClient()

	dir := t.TempDir()
	cf := testConfig(t, dir, "gdna:\n  licd-timeout: 5s\n")
	first := licdHTTPClient(cf)
	if first.Timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", first.Timeout)
	}
	if again := licdHTTPClient(cf); again != first {
		t.Error("a second request for the client did not return the shared one")
	}

	// a change to the settings that it uses
	changed := testConfig(t, dir, "gdna:\n  licd-timeout: 7s\n")
	if c := licdHTTPClient(changed); c == first || c.Timeout != 7*time.Second {
		t.Errorf("client for changed settings = %p (timeout %v), want a new one with a 7s timeout", c, c.Timeout)
	}

	closeLicdClient()
	if c := licdHTTPClient(changed); c.Timeout != 7*time.Second {
		t.Errorf("timeout = %v, want 7s", c.Timeout)
	}
}

// NULL values in the results of the queries of the reports are empty
// strings, they are not errors
func TestQueryToTableAndHeadlinesWithNulls(t *testing.T) {
	ctx := context.Background()
	cf := fetchTestConfig(t, t.TempDir())
	db := testDB(t, cf)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	table, err := queryToTable(ctx, tx, nil, "SELECT NULL AS a, 'x' AS b, 3 AS c UNION ALL SELECT 'y', NULL, NULL")
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"a", "b", "c"}, {"", "x", "3"}, {"y", "", ""}}
	if fmt.Sprint(table) != fmt.Sprint(want) {
		t.Errorf("table = %q, want %q", table, want)
	}

	names, headlines, err := queryHeadlines(ctx, tx, "SELECT 'one', NULL UNION ALL SELECT 'two', 2")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(names) != "[one two]" || headlines["one"] != "" || headlines["two"] != "2" {
		t.Errorf("headlines = %q %q", names, headlines)
	}
}

// the temporary tables, and an in-memory database, only exist on the
// connection that created them so there must only be one
func TestOpenDBUsesOneConnection(t *testing.T) {
	ctx := context.Background()
	cf := fetchTestConfig(t, t.TempDir())
	config.Set(cf, "db.dsn", ":memory:")
	db := testDB(t, cf)

	if n := db.Stats().MaxOpenConnections; n != 1 {
		t.Errorf("MaxOpenConnections = %d, want 1", n)
	}

	if _, err := db.ExecContext(ctx, "CREATE TABLE t (x)"); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	// a query using the pool while the transaction is open waits for it,
	// where another connection would be to an empty database
	result := make(chan error, 1)
	go func() {
		var n int
		result <- db.QueryRowContext(ctx, "SELECT count(*) FROM t").Scan(&n)
	}()
	time.Sleep(50 * time.Millisecond)
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = <-result; err != nil {
		t.Errorf("query using the pool: %v", err)
	}
}

func TestUpdateSourcesStatus(t *testing.T) {
	ctx := context.Background()
	cf := fetchTestConfig(t, t.TempDir())
	db := testDB(t, cf)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = createTables(ctx, cf, tx, "db.main-tables", "create"); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		source string
		status any
		want   string
	}{
		{"plain", errors.New("server returned 500"), "ERROR: server returned 500"},
		{"wrapped", fmt.Errorf("outer: %w", io.ErrUnexpectedEOF), "ERROR: unexpected EOF"},
		{"ok", "OK", "OK"},
	} {
		if err = updateSources(ctx, cf, tx, test.source, "file", "/x", false, time.Now(), test.status); err != nil {
			t.Fatal(err)
		}
		var got string
		if err = tx.QueryRowContext(ctx, "SELECT status FROM sources WHERE source = ?", test.source).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Errorf("status for %s = %q, want %q", test.source, got, test.want)
		}
	}
}

// reports that are not what is expected are errors and not a crash
func TestReportsToDBWithBadInput(t *testing.T) {
	ctx := context.Background()
	cf := fetchTestConfig(t, t.TempDir())
	db := testDB(t, cf)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = createTables(ctx, cf, tx, "db.main-tables", "create"); err != nil {
		t.Fatal(err)
	}

	summary := func(rows string) error {
		c := csv.NewReader(strings.NewReader("licenceName,L\nmode,licd\nGroup,Token,Total,Used,Free\n" + rows))
		return summaryReportToDB(ctx, cf, tx, c, "s", "file", "/x", time.Now())
	}

	// a row with too few columns is skipped, the rest is loaded
	if err = summary("Overall,short\nOverall,server,Unlimited,4,Unlimited\n"); err != nil {
		t.Fatal(err)
	}
	var total sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT total FROM tokens_all WHERE token = 'server'").Scan(&total); err != nil || total.Valid {
		t.Errorf("token loaded after a short row = %v, %v, want a NULL total for Unlimited", total, err)
	}

	// a row that cannot be parsed is an error
	if err = summary("Overall,\"unterminated\n"); err == nil {
		t.Error("summary report with a row that cannot be parsed succeeded")
	}

	// as is a detail report that cannot be read
	for name, in := range map[string]string{"empty": "", "header": "a,\"b\n"} {
		c := csv.NewReader(strings.NewReader(in))
		if err = detailReportToDB(ctx, cf, tx, c, "d", "file", "/x", time.Now()); err == nil {
			t.Errorf("%s detail report succeeded", name)
		}
	}
}

func TestReadLicdSummary(t *testing.T) {
	for _, test := range []struct {
		name, in, expiry, licence string
		wantErr                   bool
	}{
		{"both", "samplingStatus,ok\nlicenceName,Test\nexpiry,01 January 2030\n", "01 January 2030", "Test", false},
		{"short rows are skipped", "expiry\nlicenceName,Test\nexpiry,01 January 2030\n", "01 January 2030", "Test", false},
		{"not parsed", "licenceName,\"Test\n", "", "", true},
		{"none", "samplingStatus,ok\n", "", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			expiry, licence, err := readLicdSummary(csv.NewReader(strings.NewReader(test.in)))
			if (err != nil) != test.wantErr || expiry != test.expiry || licence != test.licence {
				t.Errorf("readLicdSummary() = %q, %q, %v, want %q, %q, error %t", expiry, licence, err, test.expiry, test.licence, test.wantErr)
			}
		})
	}
}
