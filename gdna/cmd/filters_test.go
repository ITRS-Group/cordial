package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
)

// filtersTestConfig returns a configuration, which is also the one the
// commands use, with the filters file and database in a directory for the
// test, and the path to the filters file, which has the contents given.
func filtersTestConfig(t *testing.T, filters string) (*config.Config, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gdna-filters.yaml")
	if filters != "" {
		if err := os.WriteFile(path, []byte(filters), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c := testConfig(t, dir, fmt.Sprintf("filters:\n  file: %s\ndb:\n  file: %s\n", path, filepath.Join(dir, "gdna.sqlite")))
	setGlobals(t, c)
	return c, path
}

// captureStdout returns what fn writes to the standard output
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	err = fn()
	w.Close()
	return <-done, err
}

// entries in the filters file are not required to have a timestamp, and
// the list commands default their format
const filtersWithoutTimestamps = `
filters:
  include:
    gateway:
      - name: GW1
        origin: test
  exclude:
    source:
      - name: file:old
        user: me
  group:
    gateway:
      - name: PROD
        patterns: [ "prod-*" ]
  allocations:
    gateway:
      - name: PROD
        token: server
        allocation: 5
`

func TestListWithoutTimestampsOrFormat(t *testing.T) {
	filtersTestConfig(t, filtersWithoutTimestamps)

	for _, test := range []struct {
		name string
		fn   func() error
		want []string
	}{
		{"includes, no format", func() error { return listFilters("include", "", "") }, []string{"gateway:GW1"}},
		{"excludes of a category, plural form", func() error { return listFilters("exclude", "sources", "") }, []string{"file:old", "me"}},
		{"groups", func() error { return listGroupCmd.RunE(listGroupCmd, nil) }, []string{"gateway:PROD", "prod-*"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			out, err := captureStdout(t, test.fn)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range test.want {
				if !strings.Contains(out, want) {
					t.Errorf("output does not contain %q:\n%s", want, out)
				}
			}
		})
	}

	if err := listFilters("include", "", "no-such-format"); err == nil {
		t.Error("listFilters with an unknown format succeeded")
	}
}

func TestProcessFiltersWithoutTimestamps(t *testing.T) {
	ctx := context.Background()
	filtersTestConfig(t, filtersWithoutTimestamps)
	db := testDB(t, cf)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	for _, root := range []string{"include", "exclude", "group", "allocations"} {
		if err = createTables(ctx, cf, tx, config.Join("filters", root), "create"); err != nil {
			t.Fatal(err)
		}
	}
	for _, filterType := range []string{"include", "exclude"} {
		if err = processFilters(ctx, cf, tx, filterType); err != nil {
			t.Fatalf("processFilters(%s): %v", filterType, err)
		}
	}
	if err = processGroups(ctx, cf, tx); err != nil {
		t.Fatalf("processGroups: %v", err)
	}
	if err = processAllocations(ctx, cf, tx); err != nil {
		t.Fatalf("processAllocations: %v", err)
	}

	for _, test := range []struct{ table, name string }{
		{"gateway_include", "GW1"},
		{"source_exclude", "file:old"},
	} {
		var name string
		var timestamp sql.NullString
		if err = tx.QueryRowContext(ctx, "SELECT "+strings.Split(test.table, "_")[0]+", timestamp FROM "+test.table).Scan(&name, &timestamp); err != nil {
			t.Fatalf("%s: %v", test.table, err)
		}
		if name != test.name || timestamp.Valid {
			t.Errorf("%s = (%q, %v), want %q with a NULL timestamp", test.table, name, timestamp, test.name)
		}
	}
}

func TestProcessFiltersWithTimestamp(t *testing.T) {
	ctx := context.Background()
	filtersTestConfig(t, `
filters:
  include:
    gateway:
      - name: GW1
        timestamp: 2024-01-02T03:04:05Z
`)
	db := testDB(t, cf)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = createTables(ctx, cf, tx, "filters.include", "create"); err != nil {
		t.Fatal(err)
	}
	if err = processFilters(ctx, cf, tx, "include"); err != nil {
		t.Fatal(err)
	}
	var timestamp sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT timestamp FROM gateway_include").Scan(&timestamp); err != nil {
		t.Fatal(err)
	}
	if !timestamp.Valid || !strings.HasPrefix(timestamp.String, "2024-01-02") {
		t.Errorf("timestamp = %v, want the one in the file", timestamp)
	}
}

// errors loading the filters must be returned and not end the process or
// leave tables that are only partly loaded to be committed
func TestProcessFiltersErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("table that cannot be loaded", func(t *testing.T) {
		filtersTestConfig(t, filtersWithoutTimestamps)
		db := testDB(t, cf)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		// the tables have not been created
		if err = processFilters(ctx, cf, tx, "include"); err == nil {
			t.Error("processFilters succeeded without the tables")
		}
		if err = processGroups(ctx, cf, tx); err == nil {
			t.Error("processGroups succeeded without the tables")
		}
		if err = processAllocations(ctx, cf, tx); err == nil {
			t.Error("processAllocations succeeded without the tables")
		}
	})

	t.Run("insert that fails", func(t *testing.T) {
		filtersTestConfig(t, filtersWithoutTimestamps)
		db := testDB(t, cf)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if err = createTables(ctx, cf, tx, "filters.include", "create"); err != nil {
			t.Fatal(err)
		}
		// a table that does not accept the values
		if _, err = tx.ExecContext(ctx, "DROP TABLE gateway_include; CREATE TABLE gateway_include (gateway TEXT, user TEXT, origin TEXT, comment TEXT, timestamp TIMESTAMP, CHECK (gateway = 'x'))"); err != nil {
			t.Fatal(err)
		}
		if err = processFilters(ctx, cf, tx, "include"); err == nil || !strings.Contains(err.Error(), "gateway") {
			t.Errorf("processFilters error = %v, want the failed insert for the gateway category", err)
		}
	})

	t.Run("filters file that cannot be read", func(t *testing.T) {
		filtersTestConfig(t, "filters: [ unterminated\n")
		db := testDB(t, cf)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		// using the defaults instead would silently drop the filters
		if err = processFilters(ctx, cf, tx, "include"); err == nil {
			t.Error("processFilters used the defaults for a filters file that cannot be read")
		}
	})
}

func TestValidateCategory(t *testing.T) {
	for arg, want := range map[string]string{"gateway": "gateway", "gateways": "gateway", "servers": "server", "hostid": "hostid", "plugins": "plugin", "sources": "source"} {
		if got, err := validateCategory(arg); err != nil || got != want {
			t.Errorf("validateCategory(%q) = %q, %v, want %q", arg, got, err, want)
		}
	}

	_, err := validateCategory("nonsense")
	if err == nil || !strings.Contains(err.Error(), "gateway, gateways, hostid, hostids, plugin, plugins, server, servers, source, sources") {
		t.Errorf("validateCategory(\"nonsense\") error = %v, want the sorted list of categories", err)
	}
}

func TestAddRemoveFilters(t *testing.T) {
	filtersTestConfig(t, "")

	read := func(filterType, category string) (found []Filter) {
		t.Helper()
		ig, err := readFilters(cf)
		if err != nil {
			t.Fatal(err)
		}
		if err = ig.UnmarshalKey(config.Join("filters", filterType, category), &found); err != nil {
			t.Fatal(err)
		}
		return
	}

	// the plural form of a category must be stored under the category
	if err := addIncludeCmd.RunE(addIncludeCmd, []string{"gateways", "GW1,GW2\nGW3"}); err != nil {
		t.Fatal(err)
	}
	if got := read("include", "gateway"); len(got) != 3 || got[0].Name != "GW1" || got[2].Name != "GW3" || got[0].Timestamp == nil {
		t.Errorf("include gateway = %+v, want GW1, GW2 and GW3 with timestamps", got)
	}
	if got := read("include", "gateways"); len(got) != 0 {
		t.Errorf("entries stored under the plural category: %+v", got)
	}

	// a category and names, without --all
	removeCmdAll = false
	if err := removeIncludeCmd.RunE(removeIncludeCmd, []string{"gateways", "GW2"}); err != nil {
		t.Fatal(err)
	}
	if got := read("include", "gateway"); len(got) != 2 {
		t.Errorf("include gateway after removing GW2 = %+v", got)
	}

	// only a category with --all
	removeCmdAll = true
	if err := removeIncludeCmd.RunE(removeIncludeCmd, []string{"gateway"}); err != nil {
		t.Fatal(err)
	}
	if got := read("include", "gateway"); len(got) != 0 {
		t.Errorf("include gateway after removing all = %+v", got)
	}

	// a failure to save is an error, not a silent success
	for _, fn := range []func() error{
		func() error { return addFilter("include", "gateway", []string{"GW9"}) },
		func() error { return removeFilter("include", "gateway", []string{"GW9"}) },
	} {
		// a file in the place of the directory for the filters file
		blocker := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(blocker, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		config.Set(cf, "filters.file", filepath.Join(blocker, "gdna-filters.yaml"))
		if err := fn(); err == nil {
			t.Error("a filter that cannot be saved was reported as saved")
		}
	}
}

func TestRemoveArgs(t *testing.T) {
	for _, test := range []struct {
		name    string
		all     bool
		args    []string
		wantErr bool
	}{
		{"nothing", false, nil, true},
		{"category only", false, []string{"gateway"}, true},
		{"category and name", false, []string{"gateway", "GW1"}, false},
		{"all, nothing", true, nil, true},
		{"all, category", true, []string{"gateway"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			old := removeCmdAll
			t.Cleanup(func() { removeCmdAll = old })
			removeCmdAll = test.all
			for _, cmd := range []*cobra.Command{removeIncludeCmd, removeExcludeCmd, removeGroupCmd} {
				if err := cmd.Args(cmd, test.args); (err != nil) != test.wantErr {
					t.Errorf("%s: Args(%v) error = %v, want error %t", cmd.Name(), test.args, err, test.wantErr)
				}
			}
		})
	}

	if err := addGroupCmd.Args(addGroupCmd, []string{"gateway"}); err == nil {
		t.Error("add group accepted a category without a group name")
	}
}

func TestGroupCommands(t *testing.T) {
	filtersTestConfig(t, "")

	groups := func() (g []Group) {
		t.Helper()
		ig, err := readFilters(cf)
		if err != nil {
			t.Fatal(err)
		}
		if err = ig.UnmarshalKey("filters.group.gateway", &g); err != nil {
			t.Fatal(err)
		}
		return
	}

	for _, name := range []string{"B", "A"} {
		if err := addGroupCmd.RunE(addGroupCmd, []string{"gateways", name, "p-*"}); err != nil {
			t.Fatal(err)
		}
	}
	if got := groups(); len(got) != 2 {
		t.Fatalf("groups = %+v, want two", got)
	}

	// the groups are listed sorted
	out, err := captureStdout(t, func() error { return listGroupCmd.RunE(listGroupCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	if a, b := strings.Index(out, "gateway:A"), strings.Index(out, "gateway:B"); a < 0 || b < 0 || a > b {
		t.Errorf("groups are not listed in order:\n%s", out)
	}

	// --all with only a category removes the groups, without an index
	// out of range or just printing the usage
	removeCmdAll = true
	if err = removeGroupCmd.RunE(removeGroupCmd, []string{"gateway"}); err != nil {
		t.Fatal(err)
	}
	if got := groups(); len(got) != 0 {
		t.Errorf("groups after removing all = %+v", got)
	}
}

// the prefix for the filter clauses is bound to the configuration that
// it was built from, so that a run that started before a reload and one
// after use their own
func TestFilterPrefixIsBoundToItsConfig(t *testing.T) {
	dir := t.TempDir()
	old := testConfig(t, dir, "filters:\n  include:\n    gateway:\n      table: old_include\n  exclude:\n    gateway:\n      table: old_exclude\n")
	updated := testConfig(t, dir, "filters:\n  include:\n    gateway:\n      table: new_include\n  exclude:\n    gateway:\n      table: new_exclude\n")

	const query = "SELECT 1 FROM gw WHERE ${filters:gw:gateway}"
	for i := range 3 {
		// alternating, as runs in progress do
		for _, test := range []struct {
			name string
			cf   *config.Config
			want string
		}{
			{"old", old, "old_include"},
			{"new", updated, "new_include"},
		} {
			// the prefix on every expansion, as the load of the configuration sets
			got := config.Expand[string](test.cf, query)
			if !strings.Contains(got, test.want) || !strings.Contains(got, strings.Replace(test.want, "include", "exclude", 1)) {
				t.Errorf("round %d: %s configuration expanded to %q, want its own tables %s", i, test.name, got, test.want)
			}
			// and passed explicitly, as the creation of tables does
			got = config.Expand[string](test.cf, query, buildFilterSQL(test.cf))
			if !strings.Contains(got, test.want) {
				t.Errorf("round %d: %s configuration expanded explicitly to %q, want its own tables %s", i, test.name, got, test.want)
			}
		}
	}

	// a category without tables is an error and not a broken query
	if got := config.Expand[string](old, "${filters:gw:nonsense}"); strings.Contains(got, "EXISTS") {
		t.Errorf("unknown category expanded to %q", got)
	}
}
