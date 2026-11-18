package cmd

import (
	"archive/zip"
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestValidateLayerConfig(t *testing.T) {
	tests := []struct {
		old string
		new string
	}{
		{"plugins.level1.plugins", "plugins.layer1"},
		{"plugins.level1-optional.plugins", "plugins.layer1-optional"},
		{"plugins.level2.plugins", "plugins.layer2"},
		{"reports.coverage-by-level-and-plugin.query", "reports.coverage-by-layer-and-plugin"},
	}
	for _, test := range tests {
		t.Run(test.old, func(t *testing.T) {
			cf := config.New(config.WithDefaults(defaults, "yaml"))
			config.Set(cf, test.old, "custom")
			if err := validateLayerConfig(cf); err == nil || !strings.Contains(err.Error(), test.new) {
				t.Fatalf("validateLayerConfig() = %v, want migration to %q", err, test.new)
			}
		})
	}

	cf := config.New(config.WithDefaults(defaults, "yaml"))
	config.Set(cf, "plugins.layer1.plugins", []string{"custom"})
	if err := validateLayerConfig(cf); err != nil {
		t.Fatalf("valid layer configuration rejected: %v", err)
	}

	t.Run("legacy environment override", func(t *testing.T) {
		t.Setenv("GDNA_PLUGINS_LEVEL1_PLUGINS", "cpu")
		cf := config.New(config.WithDefaults(defaults, "yaml"), config.WithEnvs("GDNA", "_"))
		if err := validateLayerConfig(cf); err == nil || !strings.Contains(err.Error(), "plugins.layer1") {
			t.Fatalf("validateLayerConfig() = %v, want migration to plugins.layer1", err)
		}
	})

	t.Run("legacy nested report environment override", func(t *testing.T) {
		t.Setenv("GDNA_REPORTS_COVERAGE_BY_LEVEL_AND_PLUGIN_DATAVIEW_GROUP", "custom")
		cf := config.New(config.WithDefaults(defaults, "yaml"), config.WithEnvs("GDNA", "_"))
		if err := validateLayerConfig(cf); err == nil || !strings.Contains(err.Error(), "reports.coverage-by-layer-and-plugin") {
			t.Fatalf("validateLayerConfig() = %v, want migration to the layer report", err)
		}
	})

	t.Run("legacy optional plugin environment override", func(t *testing.T) {
		t.Setenv("GDNA_PLUGINS_LEVEL1_OPTIONAL_CUSTOM", "custom")
		cf := config.New(config.WithDefaults(defaults, "yaml"), config.WithEnvs("GDNA", "_"))
		if err := validateLayerConfig(cf); err == nil || !strings.Contains(err.Error(), "plugins.layer1-optional") {
			t.Fatalf("validateLayerConfig() = %v, want migration to plugins.layer1-optional", err)
		}
	})

	t.Run("legacy email report selector", func(t *testing.T) {
		cf := config.New(config.WithDefaults(defaults, "yaml"))
		config.Set(cf, "email.body-reports", "gdna-summary,coverage-by-level*")
		if err := validateLayerConfig(cf); err == nil || !strings.Contains(err.Error(), "email.body-reports") {
			t.Fatalf("validateLayerConfig() = %v, want error for the email report selector", err)
		}
	})
}

func TestValidateLayerReportSelection(t *testing.T) {
	for _, test := range []struct {
		selection string
		wantError bool
	}{
		{"coverage-by-level-and-plugin", true},
		{"servers,coverage-by-level*:example", true},
		{"*level-and-plugin", true},
		{"coverage-by-layer-and-plugin", false},
		{"coverage-by-*", false},
		{"servers", false},
		{"", false},
	} {
		t.Run(test.selection, func(t *testing.T) {
			err := validateLayerReportSelection(test.selection)
			if (err != nil) != test.wantError {
				t.Errorf("validateLayerReportSelection(%q) = %v, want error %t", test.selection, err, test.wantError)
			}
		})
	}
}

func TestHistoricalCustomPluginTables(t *testing.T) {
	db, err := sql.Open(dbtype, filepath.Join(t.TempDir(), "gdna.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE gdna_version (version TEXT NOT NULL);
		INSERT INTO gdna_version VALUES ('v1.17');
		CREATE TABLE custom_required (plugin TEXT NOT NULL UNIQUE);
		INSERT INTO custom_required VALUES ('cpu');
		CREATE TABLE custom_optional (plugin TEXT NOT NULL UNIQUE);
		INSERT INTO custom_optional VALUES ('wmi');
		CREATE TABLE custom_application (plugin TEXT NOT NULL UNIQUE);
		INSERT INTO custom_application VALUES ('fkm');
		PRAGMA user_version = 3;`); err != nil {
		t.Fatal(err)
	}

	cf := config.New(config.WithDefaults(defaults, "yaml"))
	for key, table := range map[string]string{
		"plugins.layer1.table":          "custom_required",
		"plugins.layer1-optional.table": "custom_optional",
		"plugins.layer2.table":          "custom_application",
	} {
		config.Set(cf, key, table)
	}
	if err := updateSchema(context.Background(), db, cf); err != nil {
		t.Fatalf("updateSchema() with custom historical tables: %v", err)
	}
	for table, plugin := range map[string]string{
		"custom_required":    "cpu",
		"custom_optional":    "wmi",
		"custom_application": "fkm",
	} {
		if _, err := db.Exec("INSERT INTO "+table+" (plugin) VALUES (?)", plugin); err != nil {
			t.Fatalf("%s did not retain data and permit duplicates: %v", table, err)
		}
		var count int
		if err := db.QueryRow("SELECT count(*) FROM "+table+" WHERE plugin = ?", plugin).Scan(&count); err != nil || count != 2 {
			t.Errorf("%s plugin count = %d, %v; want 2", table, count, err)
		}
	}
}

func TestLayerPluginTableMigration(t *testing.T) {
	for _, test := range []struct {
		name         string
		setup        string
		wantLayer1   []string
		wantLayer2   []string
		wantOptional bool
	}{
		{
			name: "existing destination and missing optional table",
			setup: `
				CREATE TABLE level1_plugins (plugin TEXT NOT NULL);
				INSERT INTO level1_plugins VALUES ('cpu'), ('cpu');
				CREATE TABLE level2_plugins (plugin TEXT NOT NULL);
				INSERT INTO level2_plugins VALUES ('fkm');
				CREATE TABLE layer2_plugins (plugin TEXT NOT NULL);
				INSERT INTO layer2_plugins VALUES ('custom');
				PRAGMA user_version = 5;`,
			wantLayer1: []string{"cpu", "cpu"},
			wantLayer2: []string{"custom"},
		},
		{
			name: "upgrade through historical schema",
			setup: `
				CREATE TABLE gdna_version (version TEXT NOT NULL);
				INSERT INTO gdna_version VALUES ('v1.17');
				CREATE TABLE level1_plugins (plugin TEXT NOT NULL);
				INSERT INTO level1_plugins VALUES ('cpu');
				CREATE TABLE level1_optional_plugins (plugin TEXT NOT NULL);
				INSERT INTO level1_optional_plugins VALUES ('wmi');
				CREATE TABLE level2_plugins (plugin TEXT NOT NULL);
				INSERT INTO level2_plugins VALUES ('fkm');
				PRAGMA user_version = 3;`,
			wantLayer1:   []string{"cpu"},
			wantLayer2:   []string{"fkm"},
			wantOptional: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, err := sql.Open(dbtype, filepath.Join(t.TempDir(), "gdna.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec(test.setup); err != nil {
				t.Fatal(err)
			}
			cf := config.New(config.WithDefaults(defaults, "yaml"))
			if err := updateSchema(context.Background(), db, cf); err != nil {
				t.Fatalf("updateSchema() = %v", err)
			}
			for table, want := range map[string][]string{
				"layer1_plugins": test.wantLayer1,
				"layer2_plugins": test.wantLayer2,
			} {
				rows, err := db.Query("SELECT plugin FROM " + table + " ORDER BY rowid")
				if err != nil {
					t.Fatal(err)
				}
				var got []string
				for rows.Next() {
					var plugin string
					if err := rows.Scan(&plugin); err != nil {
						t.Fatal(err)
					}
					got = append(got, plugin)
				}
				if err := rows.Err(); err != nil {
					t.Fatal(err)
				}
				rows.Close()
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s = %v, want %v", table, got, want)
				}
			}
			var optional, legacy, version int
			if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'layer1_optional_plugins'").Scan(&optional); err != nil {
				t.Fatal(err)
			}
			if (optional == 1) != test.wantOptional {
				t.Errorf("optional table exists = %t, want %t", optional == 1, test.wantOptional)
			}
			if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'level1_plugins'").Scan(&legacy); err != nil || legacy != 1 {
				t.Errorf("legacy table count = %d, %v; want 1", legacy, err)
			}
			if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 8 {
				t.Errorf("schema version = %d, %v; want 8", version, err)
			}
		})
	}
}

func TestLayerReportsAndDashboards(t *testing.T) {
	ctx := context.Background()
	cf := config.New(config.WithDefaults(defaults, "yaml"))
	config.Set(cf, "db.dsn", filepath.Join(t.TempDir(), "gdna.sqlite"))
	db, err := openDB(ctx, cf, "db.dsn", false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := createTables(ctx, cf, tx, "db.main-tables", "create"); err != nil {
		t.Fatal(err)
	}
	if err := updateReportingDatabase(ctx, cf, tx, nil); err != nil {
		t.Fatal(err)
	}

	report := func(name string) [][]string {
		t.Helper()
		base := "reports." + name
		table, err := queryToTable(ctx, tx, config.Get[[]string](cf, base+".columns"), config.Get[string](cf, base+".query"))
		if err != nil {
			t.Fatalf("report %s: %v", name, err)
		}
		return table
	}
	coverage := report("coverage-by-layer-and-plugin")
	if len(coverage) < 2 || len(coverage[0]) == 0 || coverage[0][0] != "layer # plugin" {
		t.Fatalf("coverage report has incorrect header or rows: %v", coverage)
	}
	rows := make(map[string]bool)
	for _, row := range coverage[1:] {
		rows[row[0]] = true
	}
	for _, name := range []string{"layer1", "layer1 # cpu", "layer2 # fkm", "layer3"} {
		if !rows[name] {
			t.Errorf("coverage report missing row %q", name)
		}
	}

	plugins := report("plugins")
	var pluginColumn, layerColumn int = -1, -1
	for i, column := range plugins[0] {
		switch column {
		case "plugin":
			pluginColumn = i
		case "layer":
			layerColumn = i
		}
	}
	if pluginColumn < 0 || layerColumn < 0 {
		t.Fatalf("plugins report has incorrect columns: %v", plugins[0])
	}
	layers := make(map[string]string)
	for _, row := range plugins[1:] {
		layers[row[pluginColumn]] = row[layerColumn]
	}
	for plugin, want := range map[string]string{"cpu": "1", "fkm": "2", "postgres-plugin": "3"} {
		if got := layers[plugin]; got != want {
			t.Errorf("plugin %q layer = %q, want %q", plugin, got, want)
		}
	}

	for _, dashboard := range []struct {
		name        string
		report      string
		table       [][]string
		legacyNames []string
	}{
		{"monitoring-coverage-PROD.adb", "coverage-by-layer-and-plugin", coverage, []string{"Coverage By Level And Plugin", "level1 #", "level2 #"}},
		{"plugin-utilization-PROD.adb", "plugin-coverage", report("plugin-coverage"), []string{"Level 1 Coverage", "Level 2 Coverage", "Level 3 Coverage"}},
	} {
		archive, err := zip.OpenReader(filepath.Join("..", "geneos", dashboard.name))
		if err != nil {
			t.Fatal(err)
		}
		var contents []byte
		for _, file := range archive.File {
			if strings.HasSuffix(file.Name, ".Activedashboard") {
				reader, err := file.Open()
				if err != nil {
					t.Fatal(err)
				}
				contents, err = io.ReadAll(reader)
				reader.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		archive.Close()
		for _, legacy := range dashboard.legacyNames {
			if strings.Contains(string(contents), legacy) {
				t.Errorf("%s still contains legacy binding %q", dashboard.name, legacy)
			}
		}
		rows := make(map[string]bool)
		for _, row := range dashboard.table[1:] {
			rows[row[0]] = true
		}
		columns := make(map[string]bool)
		for _, column := range dashboard.table[0] {
			columns[column] = true
		}
		title := config.Get[string](cf, "reports."+dashboard.report+".name")
		pattern := regexp.MustCompile(regexp.QuoteMeta(title) + ` / ([^/]+?) / ([^~]+?)~`)
		bindings := pattern.FindAllStringSubmatch(string(contents), -1)
		if len(bindings) == 0 {
			t.Fatalf("%s has no bindings to %q", dashboard.name, title)
		}
		for _, binding := range bindings {
			if !rows[binding[1]] || !columns[binding[2]] {
				t.Errorf("%s binding %q / %q does not match report output", dashboard.name, binding[1], binding[2])
			}
		}
	}
}

func TestMatchReport(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		wantMatch     bool
		wantSubreport string
	}{
		{name: "literal", pattern: "gateway-status,licd-summary", wantMatch: true},
		{name: "glob", pattern: "gateway-*", wantMatch: true},
		{name: "matched subreport", pattern: "gateway-status:west,licd-summary", wantMatch: true, wantSubreport: "west"},
		{name: "suffix belongs to matched pattern", pattern: "other:stale,gateway-status", wantMatch: true},
		{name: "no match", pattern: "other,licd-*"},
	}
	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			gotMatch, gotSubreport := matchReport("gateway-status", test.pattern)
			if gotMatch != test.wantMatch || gotSubreport != test.wantSubreport {
				t.Errorf("matchReport() = (%t, %q), want (%t, %q)", gotMatch, gotSubreport, test.wantMatch, test.wantSubreport)
			}
		})
	}
}

func TestColOrNull(t *testing.T) {
	columns := map[string]int{"name": 0, "empty": 1}
	fields := []string{"Gateway A", ""}

	if got := colOrNull("name", columns, fields); !got.Valid || got.String != "Gateway A" {
		t.Errorf("colOrNull(name) = %#v, want valid Gateway A", got)
	}
	for _, column := range []string{"empty", "missing"} {
		if got := colOrNull(column, columns, fields); got.Valid {
			t.Errorf("colOrNull(%q) = %#v, want NULL", column, got)
		}
	}
}

func TestReadLicdReportSections(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "licd.csv")
	contents := "samplingStatus,status\n1,ok\x00Group,usage\ngroup-a,5\x00Req Number,details\n123,detail\x00"
	if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	summary, tokenUsage, details, err := readLicdReport(filename)
	if err != nil {
		t.Fatalf("readLicdReport() error = %v", err)
	}
	for _, section := range []struct {
		name   string
		reader interface{ Read() ([]string, error) }
		want   [][]string
	}{
		{name: "summary", reader: summary, want: [][]string{{"samplingStatus", "status"}, {"1", "ok"}}},
		{name: "token usage", reader: tokenUsage, want: [][]string{{"Group", "usage"}, {"group-a", "5"}}},
		{name: "details", reader: details, want: [][]string{{"Req Number", "details"}, {"123", "detail"}}},
	} {
		t.Run(section.name, func(t *testing.T) {
			var got [][]string
			for {
				row, err := section.reader.Read()
				if err != nil {
					if err != io.EOF {
						t.Fatalf("Read() error = %v", err)
					}
					break
				}
				got = append(got, row)
			}
			if !reflect.DeepEqual(got, section.want) {
				t.Errorf("section rows = %#v, want %#v", got, section.want)
			}
		})
	}
}

func TestReadLicdReportRejectsMissingSection(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "incomplete.csv")
	if err := os.WriteFile(filename, []byte("samplingStatus\n1\x00Group\n2\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := readLicdReport(filename); err == nil {
		t.Fatal("readLicdReport() succeeded without all three sections")
	}
}

func TestColOrNullUsesSQLNullString(t *testing.T) {
	got := colOrNull("count", map[string]int{"count": 0}, []string{"0"})
	want := sql.NullString{String: "0", Valid: true}
	if got != want {
		t.Errorf("colOrNull(count) = %#v, want %#v", got, want)
	}
}

func TestEmailConfToString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "string", in: "alerts@example.com", want: "alerts@example.com"},
		{name: "string slice", in: []string{"a@example.com", "b@example.com"}, want: "a@example.com,b@example.com"},
		{name: "generic slice", in: []any{"a@example.com", 7}, want: "a@example.com,7"},
		{name: "unsupported", in: 42, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := emailConfToString(test.in); got != test.want {
				t.Errorf("emailConfToString() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestReportLookupTableMasksIdentity(t *testing.T) {
	lookup := reportLookupTable("licd-summary", "licd", true)
	if lookup["hostname"] != "********" || lookup["username"] != "********" {
		t.Errorf("scrambled identity = (%q, %q), want masked values", lookup["hostname"], lookup["username"])
	}
	if lookup["report-name"] != "licd-summary" || lookup["report-group"] != "licd" {
		t.Errorf("report metadata = %#v", lookup)
	}
	if _, err := time.Parse(time.DateOnly, lookup["date"]); err != nil {
		t.Errorf("date value %q is not a date: %v", lookup["date"], err)
	}
	if _, err := time.Parse(time.TimeOnly, lookup["time"]); err != nil {
		t.Errorf("time value %q is not a time: %v", lookup["time"], err)
	}
}

// quietLog discards the log of the program for the duration of a test
func quietLog(t *testing.T) {
	t.Helper()
	old := log
	log = slog.New(slog.DiscardHandler)
	t.Cleanup(func() { log = old })
}

// testConfig loads a configuration, with the defaults, from a file with
// the YAML given, which is always in the directory of the test
func testConfig(t *testing.T, dir, yaml string) *config.Config {
	t.Helper()
	path := filepath.Join(dir, "gdna.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cf, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	return cf
}

// testDB opens the database in the configuration, which the test closes
func testDB(t *testing.T, cf *config.Config) *sql.DB {
	t.Helper()
	db, err := openDB(context.Background(), cf, cf.Join("db", "dsn"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// setGlobals sets the package level variables that the commands use for
// the test and restores them afterwards
func setGlobals(t *testing.T, c *config.Config) {
	t.Helper()
	oldCf, oldPost, oldOnStart, oldOnStartEmail, oldTime := cf, postProcess, onStart, onStartEMail, overrideFiletime
	oldNames, oldAll, oldScramble, oldZip := reportNames, removeCmdAll, scrambleNames, outputZip
	t.Cleanup(func() {
		cf, postProcess, onStart, onStartEMail, overrideFiletime = oldCf, oldPost, oldOnStart, oldOnStartEmail, oldTime
		reportNames, removeCmdAll, scrambleNames, outputZip = oldNames, oldAll, oldScramble, oldZip
		fetchCmdSources = nil
		sourcesChanged.Store(false)
		closeLicdClient()
	})
	cf = c
	fetchCmdSources = nil
	quietLog(t)
}
