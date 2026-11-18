package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/internal/defaultscheck"
)

// testConfig returns the embedded defaults (which must parse without
// error) with the output directory set to dir
func testConfig(t *testing.T, dir string) *config.Config {
	t.Helper()
	c, err := config.Read("gateway-reporter",
		config.WithDefaults(defaults, "yaml"),
		config.StopOnInternalDefaultsErrors(),
		config.Format("yaml"),
		config.SkipWorkingDir(),
		config.SkipUserConfDir(),
		config.SkipSystemDir(),
	)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	config.Set(c, "output.directory", dir)
	return c
}

// useConfig sets the package level config for the duration of the test
func useConfig(t *testing.T, c *config.Config) {
	t.Helper()
	old := cf
	cf = c
	t.Cleanup(func() { cf = old })
}

func testEntities() ([]Entity, map[string]geneos.Probe) {
	return []Entity{
		{
			Name:       "server1",
			Probe:      Probe{Name: "np1", Hostname: "host1", Port: 7036},
			Attributes: map[string]string{"ENV": "prod"},
			Samplers: []Sampler{
				{Type: "base", Name: "fkm1", Plugin: "fkm", Column1: []string{"/var/log/a.log", "/var/log/very/long/path/to/some/file/b.log"}},
				{Type: "base", Name: "sql1", Plugin: "sql-toolkit", Column1: []string{"db1"}, Column2: []string{`"q1": [select 1]`}},
			},
		},
		{
			Name:  "server2",
			Probe: Probe{Name: "np2"},
		},
	}, map[string]geneos.Probe{"np1": {}, "np2": {}}
}

func TestDefaultsFile(t *testing.T) {
	m := defaultscheck.Parse(t, defaults)

	defaultscheck.RequireKeys(t, m,
		"site",
		"output.directory",
		"output.formats.xlsx",
		"output.formats.csv",
		"output.formats.csvdir",
		"output.formats.json",
		"output.formats.xml",
		"output.toolkit-include.enable",
		"output.toolkit-include.include-file",
		"output.toolkit-include.probe-name",
		"output.toolkit-include.entity-name",
		"output.toolkit-include.sampler-group",
		"output.toolkit-include.sampler-name",
		"output.toolkit-include.sampler-script",
		"output.plugins.single-column",
		"output.plugins.two-column",
		"output.reports.summary.filename",
		"output.reports.entities.sheetname",
	)

	// plugin lists name only configured reports
	for _, list := range []string{"output.plugins.single-column", "output.plugins.two-column"} {
		v, _ := defaultscheck.Lookup(m, list)
		for _, p := range v.([]any) {
			defaultscheck.RequireKeys(t, m, "output.reports."+p.(string)+".filename", "output.reports."+p.(string)+".sheetname")
		}
	}

	defaultscheck.CheckRead(t, m, []string{"."})
}

// a broken embedded defaults file used to be silently ignored, giving
// an empty output directory and no formats
func TestDefaultsLoad(t *testing.T) {
	c := testConfig(t, "/x")
	for _, k := range []string{"output.formats.xlsx", "output.formats.csv", "output.formats.csvdir", "output.formats.json", "output.formats.xml"} {
		if config.Get[string](c, k) == "" {
			t.Errorf("%s is empty after loading defaults", k)
		}
	}
}

func TestWriteReports(t *testing.T) {
	dir := t.TempDir()
	c := testConfig(t, dir)
	useConfig(t, c)
	entities, probes := testEntities()

	xmlin := bytes.NewBufferString("<gateway/>")
	if err := writeReports(c, "gw", entities, probes, xmlin); err != nil {
		t.Fatalf("writeReports: %v", err)
	}

	for _, pattern := range []string{"gw-Report-*.xlsx", "gw-Report-*.zip", "gw-Report-*.json", "gw-Merged-*.xml", "gw-CSV-Reports/0-summary.csv", "gw-CSV-Reports/fkm.csv", "gw-CSV-Reports/gw-report.setup.xml"} {
		if m, _ := filepath.Glob(filepath.Join(dir, pattern)); len(m) != 1 {
			t.Errorf("expected one file matching %s, got %v", pattern, m)
		}
	}

	// the include file is world-readable like the other outputs
	if runtime.GOOS != "windows" {
		st, err := os.Stat(filepath.Join(dir, "gw-CSV-Reports", "gw-report.setup.xml"))
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Perm() != 0644 {
			t.Errorf("include file mode = %v, want 0644", st.Mode().Perm())
		}
	}
}

func TestOutputCSVColumnCounts(t *testing.T) {
	entities, probes := testEntities()

	tests := []struct {
		name string
		set  map[string][]string
	}{
		{"defaults", nil},
		{"no configured plugin columns", map[string][]string{
			"output.reports.fkm.columns":         nil,
			"output.reports.sql-toolkit.columns": nil,
		}},
		{"entity columns", map[string][]string{
			"output.reports.entities.columns": {"Managed Entity", "Probe"},
		}},
		{"more entity columns", map[string][]string{
			"output.reports.entities.columns": {"a", "b", "c", "d", "e", "f"},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			c := testConfig(t, dir)
			useConfig(t, c)
			for k, v := range test.set {
				if v == nil {
					// use the built-in column defaults
					config.Set(c, k, []string(nil))
					continue
				}
				config.Set(c, k, v)
			}

			csvfiles, subdir, err := outputCSVDir(c, "gw", entities, probes)
			if err != nil {
				t.Fatal(err)
			}
			if len(csvfiles) == 0 {
				t.Fatal("no files written")
			}
			for _, f := range csvfiles {
				if f.sheetname == "0-Summary" {
					continue
				}
				b, err := os.ReadFile(f.path)
				if err != nil {
					t.Fatal(err)
				}
				checkCSVRagged(t, f.path, b)
			}
			_ = subdir

			if err := outputCSVZip(c, "gw", entities, probes); err != nil {
				t.Fatal(err)
			}
			zips, _ := filepath.Glob(filepath.Join(dir, "*.zip"))
			if len(zips) != 1 {
				t.Fatalf("zip files = %v", zips)
			}
			zr, err := zip.OpenReader(zips[0])
			if err != nil {
				t.Fatalf("zip is not valid: %v", err)
			}
			defer zr.Close()
			for _, zf := range zr.File {
				if strings.HasPrefix(zf.Name, "0-summary") {
					continue
				}
				r, _ := zf.Open()
				b, _ := io.ReadAll(r)
				r.Close()
				checkCSVRagged(t, zf.Name, b)
			}
		})
	}
}

func checkCSVRagged(t *testing.T, name string, b []byte) {
	t.Helper()
	r := csv.NewReader(bytes.NewReader(b))
	// default FieldsPerRecord of 0 makes every record match the header
	if _, err := r.ReadAll(); err != nil {
		t.Errorf("%s: inconsistent column counts: %v", name, err)
	}
}

func TestWriteReportsErrorsAreReturned(t *testing.T) {
	entities, probes := testEntities()

	for range 20 {
		dir := t.TempDir()
		c := testConfig(t, dir)
		useConfig(t, c)

		// a file where the csvdir parent needs to be
		blocker := filepath.Join(dir, "blocker")
		if err := os.WriteFile(blocker, nil, 0600); err != nil {
			t.Fatal(err)
		}
		config.Set(c, "output.formats", map[string]any{"csvdir": filepath.Join(blocker, "csv"), "json": "x.json"})

		err := writeReports(c, "gw", entities, probes, new(bytes.Buffer))
		if err == nil || !strings.Contains(err.Error(), "csvdir") {
			t.Fatalf("err = %v, want csvdir error", err)
		}
	}
}

func TestWriteReportsToolkitIncludeError(t *testing.T) {
	entities, probes := testEntities()
	dir := t.TempDir()
	c := testConfig(t, dir)
	useConfig(t, c)

	config.Set(c, "output.formats", map[string]any{"csvdir": "csv"})
	config.Set(c, "output.toolkit-include.include-file", filepath.Join("no-such-dir", "include.xml"))

	if err := writeReports(c, "gw", entities, probes, new(bytes.Buffer)); err == nil {
		t.Error("expected an error for an include file in a missing directory")
	}
}

func TestOutputDirFlag(t *testing.T) {
	cfgDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "out")
	c := testConfig(t, cfgDir)
	useConfig(t, c)

	setup := filepath.Join(t.TempDir(), "setup.xml")
	input := `<gateway><operatingEnvironment><gatewayId>1</gatewayId><gatewayName>gw</gatewayName></operatingEnvironment>` +
		`<probes><probe name="np"><hostname>localhost</hostname></probe></probes>` +
		`<managedEntities><managedEntity name="server"><probe ref="np"/></managedEntity></managedEntities></gateway>`
	if err := os.WriteFile(setup, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}

	oldOut := reportCmdOutDir
	reportCmdOutDir = outDir
	t.Cleanup(func() { reportCmdOutDir = oldOut })

	if err := reportCmd.RunE(reportCmd, []string{setup}); err != nil {
		t.Fatal(err)
	}

	if m, _ := filepath.Glob(filepath.Join(outDir, "gw-Report-*.xlsx")); len(m) != 1 {
		t.Errorf("no report in -o directory: %v", m)
	}
	if m, _ := filepath.Glob(filepath.Join(outDir, "gw-Merged-*.xml")); len(m) != 1 {
		t.Errorf("no merged XML in -o directory: %v", m)
	}
	if m, _ := os.ReadDir(cfgDir); len(m) != 0 {
		t.Errorf("files written to configured directory: %v", m)
	}
}

func TestMergedXMLOnlyKeptWhenEnabled(t *testing.T) {
	dir := t.TempDir()
	c := testConfig(t, dir)
	useConfig(t, c)
	config.Set(c, "output.formats", map[string]any{"json": "r.json", "xml": ""})

	input := `<gateway><operatingEnvironment><gatewayId>1</gatewayId><gatewayName>gw</gatewayName></operatingEnvironment></gateway>`
	if _, err := generateReports(strings.NewReader(input), ""); err != nil {
		t.Fatal(err)
	}
	if m, _ := filepath.Glob(filepath.Join(dir, "*.json")); len(m) != 1 {
		t.Errorf("json report not written: %v", m)
	}
	if m, _ := filepath.Glob(filepath.Join(dir, "*.xml")); len(m) != 0 {
		t.Errorf("unexpected XML files: %v", m)
	}
}

func TestGenerateReportsBadInput(t *testing.T) {
	c := testConfig(t, t.TempDir())
	useConfig(t, c)

	tests := []struct {
		name, input, prefix string
	}{
		{"not xml", "not xml", ""},
		{"not xml with prefix", "not xml", "pfx"},
		{"no operating environment", "<gateway/>", "pfx"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := generateReports(strings.NewReader(test.input), test.prefix); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestMergeConfigNoXML(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "gateway2.linux_64")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho no xml here\n"), 0700); err != nil {
		t.Fatal(err)
	}
	setup := filepath.Join(dir, "setup.xml")
	if err := os.WriteFile(setup, []byte("<gateway/>"), 0600); err != nil {
		t.Fatal(err)
	}

	out, err := mergeConfig(dir, setup)
	if err == nil {
		t.Errorf("expected an error, got output %q", out)
	}
}

func TestSQLToolkitEmptyMessage(t *testing.T) {
	c := testConfig(t, t.TempDir())
	useConfig(t, c)

	tests := []struct {
		name    string
		queries []geneos.Query
		want    []string
	}{
		{"valid then empty", []geneos.Query{testQuery("q1", "select 1"), testQuery("q2", "")}, []string{"select 1"}},
		{"all empty", []geneos.Query{testQuery("q1", ""), testQuery("q2", " ")}, []string{"NO QUERIES"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := &Sampler{}
			pluginInfo(s, &geneos.SQLToolkitPlugin{Queries: test.queries}, nil)
			if len(s.Column2) != len(test.want) {
				t.Fatalf("Column2 = %q, want %d entries", s.Column2, len(test.want))
			}
			for i, w := range test.want {
				if !strings.Contains(s.Column2[i], w) {
					t.Errorf("Column2[%d] = %q, want it to contain %q", i, s.Column2[i], w)
				}
			}
		})
	}
}

func testQuery(name, sql string) geneos.Query {
	return geneos.Query{
		Name: geneos.NewSingleLineString(`"` + name + `"`),
		SQL:  geneos.NewSingleLineString(sql),
	}
}

func TestXLSXColumnWidthsFromData(t *testing.T) {
	dir := t.TempDir()
	c := testConfig(t, dir)
	useConfig(t, c)
	entities, probes := testEntities()
	entities[0].Name = strings.Repeat("e", 60)

	if err := outputXLSX(c, "gw", entities, probes); err != nil {
		t.Fatal(err)
	}
	m, _ := filepath.Glob(filepath.Join(dir, "*.xlsx"))
	if len(m) != 1 {
		t.Fatalf("xlsx files = %v", m)
	}
	x, err := excelize.OpenFile(m[0])
	if err != nil {
		t.Fatal(err)
	}
	defer x.Close()

	// the entity name (column A) is far wider than its heading
	w, err := x.GetColWidth("1-Entities", "A")
	if err != nil {
		t.Fatal(err)
	}
	if w < 60 {
		t.Errorf("column A width = %v, want >= 60", w)
	}
	// the data column of the FKM sheet is sized from its data
	w, err = x.GetColWidth("FKM", "D")
	if err != nil {
		t.Fatal(err)
	}
	if w < 50 {
		t.Errorf("FKM column D width = %v, want >= 50", w)
	}
}

func TestXLSXNoAttributesOrPlugins(t *testing.T) {
	dir := t.TempDir()
	c := testConfig(t, dir)
	useConfig(t, c)
	entities := []Entity{{Name: "e", Probe: Probe{Name: "p", Hostname: "h"}}}

	if err := outputXLSX(c, "gw", entities, nil); err != nil {
		t.Fatal(err)
	}
	m, _ := filepath.Glob(filepath.Join(dir, "*.xlsx"))
	x, err := excelize.OpenFile(m[0])
	if err != nil {
		t.Fatal(err)
	}
	defer x.Close()
	merges, err := x.GetMergeCells("1-Entities")
	if err != nil {
		t.Fatal(err)
	}
	if len(merges) != 1 || merges[0].GetStartAxis() != "A1" || merges[0].GetEndAxis() != "D1" {
		var got []string
		for _, mc := range merges {
			got = append(got, mc.GetStartAxis()+":"+mc.GetEndAxis())
		}
		t.Errorf("merged cells = %v, want only A1:D1", got)
	}
}

// captureStdout runs f and returns what it wrote to os.Stdout
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	func() {
		defer func() { os.Stdout = old }()
		f()
	}()
	w.Close()
	out := <-done
	r.Close()
	return out
}

func TestValidationHook(t *testing.T) {
	// keep the hook's log file out of the real temp directory
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("TMP", os.Getenv("TMPDIR"))
	t.Setenv("TEMP", os.Getenv("TMPDIR"))
	t.Cleanup(func() { logger.Init(execname, logger.SetLogfile(os.DevNull)) })

	valid := `<gateway><operatingEnvironment><gatewayId>1</gatewayId><gatewayName>gw</gatewayName></operatingEnvironment>` +
		`<probes><probe name="np"><hostname>localhost</hostname></probe></probes>` +
		`<managedEntities><managedEntity name="server"><probe ref="np"/></managedEntity></managedEntities></gateway>`

	tests := []struct {
		name         string
		setup        string
		wantSeverity string
		wantToolkit  bool
	}{
		{"valid", valid, "None", true},
		{"invalid XML", "<gateway><unclosed & </gateway>", "Error", false},
		{"no operating environment", "<gateway/>", "Error", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			c := testConfig(t, dir)
			useConfig(t, c)

			setup := filepath.Join(t.TempDir(), "setup.xml")
			if err := os.WriteFile(setup, []byte(test.setup), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("_SETUP", setup)
			t.Setenv("_VALIDATE_TYPE", "Validate")

			var runErr error
			out := captureStdout(t, func() { runErr = Cmd.RunE(Cmd, nil) })

			var v struct {
				Issues []struct {
					Severity string `xml:"severity"`
					Message  string `xml:"message"`
				} `xml:"issues>issue"`
			}
			if err := xml.Unmarshal([]byte(out), &v); err != nil {
				t.Fatalf("hook output is not well-formed XML: %v\n%s", err, out)
			}
			if len(v.Issues) != 1 || v.Issues[0].Severity != test.wantSeverity {
				t.Fatalf("issues = %+v, want one with severity %s", v.Issues, test.wantSeverity)
			}
			if (runErr != nil) != (test.wantSeverity == "Error") {
				t.Errorf("RunE error = %v", runErr)
			}

			m, _ := filepath.Glob(filepath.Join(dir, "gw-CSV-Reports", "gw-report.setup.xml"))
			if (len(m) == 1) != test.wantToolkit {
				t.Errorf("toolkit include files = %v, want present=%v", m, test.wantToolkit)
			}
		})
	}
}
