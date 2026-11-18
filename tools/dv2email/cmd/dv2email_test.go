package cmd

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos/commands"
	"github.com/itrs-group/cordial/pkg/geneos/xpath"
	"github.com/itrs-group/cordial/tools/internal/defaultscheck"
)

// testConfig returns the embedded defaults, which must load without
// error, with the extra settings applied. It also sets globalCf for the
// duration of the test.
func testConfig(t *testing.T, settings map[string]any) *config.Config {
	t.Helper()
	return testConfigYAML(t, "", settings)
}

// testConfigYAML is like testConfig but also merges the YAML, which
// can be used for keys that Set cannot handle, such as those containing
// the key delimiter
func testConfigYAML(t *testing.T, y string, settings map[string]any) *config.Config {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	c, err := config.Read("dv2email",
		config.WithDefaults(defaults, "yaml"),
		config.StopOnInternalDefaultsErrors(),
		config.Format("yaml"),
		config.SkipWorkingDir(),
		config.SkipUserConfDir(),
		config.SkipSystemDir(),
		config.Reader(strings.NewReader(y)),
	)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	for k, v := range settings {
		config.Set(c, k, v)
	}
	old := globalCf
	globalCf = c
	t.Cleanup(func() { globalCf = old })
	return c
}

func TestDefaultsFile(t *testing.T) {
	m := defaultscheck.Parse(t, defaults)

	defaultscheck.RequireKeys(t, m,
		"gateway.host", "gateway.port", "gateway.use-tls", "gateway.allow-insecure",
		"email.smtp", "email.use-tls", "email.subject", "email.contents", "email.split",
		"files",
		"column-filter", "first-column", "row-filter", "headline-filter", "images",
		"text.template", "texttable.filename", "texttable.split",
		"html.filename", "html.template", "html.split",
		"xlsx.filename", "xlsx.sheetname", "xlsx.split", "xlsx.style", "xlsx.row-stripes",
		"xlsx.column-stripes", "xlsx.password", "xlsx.column-width",
		"text-template", "html-template",
	)

	defaultscheck.CheckRead(t, m, []string{".", "../../../pkg/email"},
		// proposed features that are not yet implemented
		"row-property-filter",
		"headline-property-filter",
		// only the main body and attachments from texttable are split
		"text.split",
		// the keys are dataview name globs, read with match()
		"first-column",
		// referenced from the text and html template settings using ${config:...}
		"text-template",
		"html-template",
	)
}

// the dataview names in the filters are also the keys of the
// configuration, so check that the flag help names the environment
// variable that the code actually reads
func TestVariablePathHelp(t *testing.T) {
	for _, name := range []string{"entity", "sampler", "type", "dataview"} {
		f := Cmd.PersistentFlags().Lookup(name)
		if f == nil {
			t.Fatalf("flag %q not found", name)
		}
		if !strings.Contains(f.Usage, "_VARIABLEPATH") || strings.Contains(f.Usage, "_VARIBLEPATH") {
			t.Errorf("flag %q usage = %q, want to refer to _VARIABLEPATH", name, f.Usage)
		}
	}
	// the setting read by the code, case-insensitively, from the environment
	c := testConfig(t, nil)
	importEnvironment(c, []string{"_VARIABLEPATH=//sampler"})
	if got := config.Get[string](c, "_variablepath"); got != "//sampler" {
		t.Errorf("_variablepath = %q", got)
	}
}

func TestExportCommandDescription(t *testing.T) {
	if exportCmd.Long == DV2EMAILCmdDescription || !strings.HasPrefix(exportCmd.Long, "Write Dataview(s) to local files") {
		t.Errorf("export command description is not from export.md: %q", exportCmd.Long)
	}
}

func TestInlineCSSFlagDeprecated(t *testing.T) {
	f := Cmd.Flags().Lookup("inline-css")
	if f == nil || f.Deprecated == "" || !f.Hidden {
		t.Errorf("inline-css should be a hidden, deprecated flag: %+v", f)
	}
}

func TestImportEnvironmentKeepsSections(t *testing.T) {
	c := testConfig(t, nil)
	env := importEnvironment(c, []string{
		"GATEWAY=bogus",
		"Email=bogus",
		"FILES=bogus",
		"TEXT=bogus",
		"HTML=bogus",
		"_FIRSTCOLUMN=name",
		"PLAIN=value",
		"MALFORMED",
	})

	if env["GATEWAY"] != "bogus" || env["PLAIN"] != "value" {
		t.Errorf("template env = %v", env)
	}
	if _, ok := env["MALFORMED"]; ok {
		t.Error("malformed environment entry imported")
	}
	if got := config.Get[string](c, "gateway.host"); got != "localhost" {
		t.Errorf("gateway.host = %q, GATEWAY env replaced the section", got)
	}
	if got := config.Get[string](c, "email.smtp"); got != "localhost" {
		t.Errorf("email.smtp = %q, EMAIL env replaced the section", got)
	}
	if got := config.Get[[]string](c, "files"); len(got) == 0 {
		t.Error("files setting replaced by environment")
	}
	if got := config.Get[string](c, "text.template"); got == "bogus" || got == "" {
		t.Errorf("text.template = %q", got)
	}
	if got := config.Get[string](c, "_firstcolumn"); got != "name" {
		t.Errorf("_firstcolumn = %q", got)
	}
	if got := config.Get[string](c, "plain"); got != "value" {
		t.Errorf("plain = %q", got)
	}
}

func testDataview(name string) *commands.Dataview {
	return &commands.Dataview{
		Name:          name,
		HeadlineOrder: []string{"samplingStatus", "other", "extra"},
		Headlines: map[string]commands.DataItem{
			"samplingStatus": {Value: "OK"},
			"other":          {Value: "1"},
			"extra":          {Value: "2"},
		},
		ColumnOrder: []string{"rowname", "a", "cpuA", "cpuB", "mem"},
		RowOrder:    []string{"sys1", "sys2", "app10", "app2"},
		Table: map[string]map[string]commands.DataItem{
			"sys1":  {"a": {Value: "z"}, "cpuA": {Value: "1"}, "cpuB": {Value: "1"}, "mem": {Value: "1"}},
			"sys2":  {"a": {Value: "y"}},
			"app10": {"a": {Value: "x"}},
			"app2":  {"a": {Value: "w"}},
		},
	}
}

func TestFilterDataview(t *testing.T) {
	tests := []struct {
		name      string
		settings  map[string]any
		firstcol  string
		wantHead  []string
		wantCols  []string
		wantRows  []string
		wantTable []string
	}{
		{
			name:     "no filters",
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "a", "cpuA", "cpuB", "mem"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
		{
			name:     "headline keep any",
			settings: map[string]any{"headline-filter": map[string][]string{"*": {"sampling*", "extra"}}},
			wantHead: []string{"extra", "samplingStatus"},
			wantCols: []string{"rowname", "a", "cpuA", "cpuB", "mem"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
		{
			name:     "column filter keeps first column with one pattern",
			settings: map[string]any{"column-filter": map[string][]string{"*": {"cpu*"}}},
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "cpuA", "cpuB"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
		{
			name:     "column filter keep any of several literals",
			settings: map[string]any{"column-filter": map[string][]string{"*": {"a", "mem"}}},
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "a", "mem"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
		{
			name:     "column filter with named first column",
			settings: map[string]any{"column-filter": map[string][]string{"*": {"cpu*"}}},
			firstcol: "host",
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"host", "cpuA", "cpuB"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
		{
			name:     "column order star does not panic",
			settings: map[string]any{"column-order": map[string][]string{"*": {"*"}}},
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "a", "cpuA", "cpuB", "mem"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
		{
			name:     "column order adjacent matches",
			settings: map[string]any{"column-order": map[string][]string{"*": {"cpu*", "a"}}},
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "cpuA", "cpuB", "a", "mem"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
		{
			name:     "column order desc",
			settings: map[string]any{"column-order": map[string][]string{"*": {"desc"}}},
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "mem", "cpuB", "cpuA", "a"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
		{
			name:     "row filter keep any also filters row order",
			settings: map[string]any{"row-filter": map[string][]string{"*": {"sys*", "app2"}}},
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "a", "cpuA", "cpuB", "mem"},
			wantRows: []string{"sys1", "sys2", "app2"},
		},
		{
			name:     "row order by rowname natural",
			settings: map[string]any{"row-order": map[string][]string{"*": {"rowname"}}},
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "a", "cpuA", "cpuB", "mem"},
			wantRows: []string{"app2", "app10", "sys1", "sys2"},
		},
		{
			name:     "row order by column descending",
			settings: map[string]any{"row-order": map[string][]string{"*": {"a-"}}},
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "a", "cpuA", "cpuB", "mem"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
		{
			name:     "invalid patterns do not filter",
			settings: map[string]any{"column-filter": map[string][]string{"*": {"[a"}}},
			wantHead: []string{"extra", "other", "samplingStatus"},
			wantCols: []string{"rowname", "a", "cpuA", "cpuB", "mem"},
			wantRows: []string{"sys1", "sys2", "app10", "app2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testConfig(t, test.settings)
			dv := testDataview("dv")

			filterDataview(dv, test.firstcol, "", "", "", "")

			if got := slices.Sorted(mapKeys(dv.Headlines)); !slices.Equal(got, test.wantHead) {
				t.Errorf("headlines = %v, want %v", got, test.wantHead)
			}
			if !slices.Equal(dv.HeadlineOrder, filterOrder([]string{"samplingStatus", "other", "extra"}, test.wantHead)) {
				t.Errorf("headline order = %v", dv.HeadlineOrder)
			}
			if !slices.Equal(dv.ColumnOrder, test.wantCols) {
				t.Errorf("columns = %v, want %v", dv.ColumnOrder, test.wantCols)
			}
			if !slices.Equal(dv.RowOrder, test.wantRows) {
				t.Errorf("rows = %v, want %v", dv.RowOrder, test.wantRows)
			}
			if len(dv.Table) != len(dv.RowOrder) {
				t.Errorf("table has %d rows but order has %d", len(dv.Table), len(dv.RowOrder))
			}
		})
	}
}

func mapKeys[V any](m map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func filterOrder(order, keep []string) (out []string) {
	for _, o := range order {
		if slices.Contains(keep, o) {
			out = append(out, o)
		}
	}
	return
}

func TestFilterDataviewOverrides(t *testing.T) {
	testConfig(t, nil)
	dv := testDataview("dv")
	// comma separated overrides, as from the environment, are also keep-any
	filterDataview(dv, "", "extra, other", "sys1,app2", "a,mem", "")
	if want := []string{"extra", "other"}; !slices.Equal(slices.Sorted(mapKeys(dv.Headlines)), want) {
		t.Errorf("headlines = %v", dv.Headlines)
	}
	if want := []string{"rowname", "a", "mem"}; !slices.Equal(dv.ColumnOrder, want) {
		t.Errorf("columns = %v, want %v", dv.ColumnOrder, want)
	}
	if want := []string{"sys1", "app2"}; !slices.Equal(dv.RowOrder, want) {
		t.Errorf("rows = %v, want %v", dv.RowOrder, want)
	}
}

func TestMatchTieBreak(t *testing.T) {
	testConfig(t, map[string]any{
		"column-order": map[string][]string{"a*": {"first"}, "*b": {"second"}},
	})
	// both patterns are the same length and match, so the choice must be stable
	for range 50 {
		got := match("ab", "column-order", "")
		if !slices.Equal(got, []string{"second"}) {
			t.Fatalf("match = %v, want [second]", got)
		}
	}
}

func TestBuildName(t *testing.T) {
	lookup := map[string]string{
		"serial": "1", "entity": "e", "sampler": "s", "dataview": "d",
		"timestamp": "20260101120000", "default": "dataviews",
	}
	tests := []struct {
		name   string
		in     string
		lookup map[string]string
		want   string
	}{
		{"auto", "auto", lookup, "1-e-s-d-20260101120000"},
		{"custom name is used as given", "report-2026", lookup, "report-2026"},
		{"empty uses default as given", "", lookup, "dataviews"},
		{"auto with no parts", "auto", map[string]string{"timestamp": "20260101120000", "default": "dataviews"}, "dataviews-20260101120000"},
		{"auto without timestamp", "auto", map[string]string{"entity": "e"}, "e"},
		{"nothing", "", map[string]string{}, "undefined"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := buildName(test.in, test.lookup); got != test.want {
				t.Errorf("buildName(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestBuildVarpath(t *testing.T) {
	oldE, oldS, oldT, oldD := entityArg, samplerArg, typeArg, dataviewArg
	t.Cleanup(func() { entityArg, samplerArg, typeArg, dataviewArg = oldE, oldS, oldT, oldD })

	entityArg, samplerArg, typeArg, dataviewArg = "e", "s", "", "d"
	if got, want := buildVarpath(false), `//managedEntity[(@name="e")]/sampler[(@name="s")]/dataview[(@name="d")]`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if got, want := buildVarpath(true), `//managedEntity[(@name="e")]/sampler[(@name="s")][(@type="")]/dataview[(@name="d")]`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func testXPath(t *testing.T, entity, sampler, typ, dataview string) *xpath.XPath {
	t.Helper()
	x, err := xpath.Parse(`/geneos/gateway[(@name="gw")]/directory/probe[(@name="np")]/managedEntity[(@name="` + entity + `")]/sampler[(@name="` + sampler + `")][(@type="` + typ + `")]/dataview[(@name="` + dataview + `")]`)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func fixtures(t *testing.T) DV2EMailData {
	t.Helper()
	var dvs []*commands.Dataview
	for _, e := range []string{"zeta", "alpha", "mid"} {
		dv := testDataview("dv")
		dv.XPath = testXPath(t, e, "sampler-"+e, "type-"+e, "dv")
		dv.SampleTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		dvs = append(dvs, dv)
	}
	return DV2EMailData{Dataviews: dvs, Env: map[string]string{"A": "b"}}
}

func TestInformationLabels(t *testing.T) {
	testConfig(t, nil)
	data := fixtures(t)
	data.Dataviews = data.Dataviews[:1]

	out, err := createTextTables(globalCf, data)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	buf.ReadFrom(out)
	rows := map[string]string{}
	for line := range strings.SplitSeq(buf.String(), "\n") {
		cells := strings.Split(strings.Trim(line, "| \t"), "|")
		if len(cells) == 2 {
			rows[strings.TrimSpace(cells[0])] = strings.TrimSpace(cells[1])
		}
	}
	for label, want := range map[string]string{"Entity": "zeta", "Type": "type-zeta", "Sampler": "sampler-zeta", "Dataview": "dv", "Probe": "np"} {
		if rows[label] != want {
			t.Errorf("text %s = %q, want %q\n%s", label, rows[label], want, buf.String())
		}
	}
}

func openXLSX(t *testing.T, r *bytes.Reader) *excelize.File {
	t.Helper()
	x, err := excelize.OpenReader(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { x.Close() })
	return x
}

func TestCreateXLSXInformation(t *testing.T) {
	testConfig(t, nil)
	data := fixtures(t)
	data.Dataviews = data.Dataviews[:1]

	out, err := createXLSX(globalCf, data)
	if err != nil {
		t.Fatal(err)
	}
	x := openXLSX(t, out)
	sheets := x.GetSheetList()
	if len(sheets) != 1 {
		t.Fatalf("sheets = %v", sheets)
	}
	for cell, want := range map[string]string{"D1": "Entity", "E1": "Type", "F1": "Sampler", "D2": "zeta", "E2": "type-zeta", "F2": "sampler-zeta"} {
		if got, _ := x.GetCellValue(sheets[0], cell); got != want {
			t.Errorf("%s = %q, want %q", cell, got, want)
		}
	}

	// the heading style covers the 9 information columns and no more
	last, _ := x.GetCellStyle(sheets[0], "I1")
	beyond, _ := x.GetCellStyle(sheets[0], "J1")
	if last == 0 || beyond != 0 {
		t.Errorf("heading style on I1 = %d, J1 = %d", last, beyond)
	}
}

func TestSerialStartsAtOne(t *testing.T) {
	testConfig(t, map[string]any{"xlsx.sheetname": "${serial}"})
	out, err := createXLSX(globalCf, fixtures(t))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := openXLSX(t, out).GetSheetList(), []string{"1", "2", "3"}; !slices.Equal(got, want) {
		t.Errorf("sheets = %v, want %v", got, want)
	}
}

func TestUniqueSheetName(t *testing.T) {
	used := map[string]bool{"sheet1": true}

	long := strings.Repeat("é", 40) // two bytes per character
	a := uniqueSheetName(long, used)
	b := uniqueSheetName(long, used)
	c := uniqueSheetName(strings.ToUpper(long), used)

	for _, n := range []string{a, b, c} {
		if l := len([]rune(n)); l > 31 {
			t.Errorf("%q is %d characters", n, l)
		}
		if !strings.EqualFold(strings.TrimRight(n, "~0123456789"), strings.Repeat("é", len([]rune(strings.TrimRight(n, "~0123456789"))))) {
			t.Errorf("%q is not valid UTF-8 prefix of the original", n)
		}
	}
	if a == b || b == c || a == c {
		t.Errorf("names are not unique: %q %q %q", a, b, c)
	}

	if got := uniqueSheetName("a:b/c?*[x]", map[string]bool{}); got != "a_b_c___x_" {
		t.Errorf("invalid characters not replaced: %q", got)
	}
	if got := uniqueSheetName("Sheet1", used); got == "Sheet1" {
		t.Error("default sheet name reused")
	}
}

func TestCreateXLSXLongNames(t *testing.T) {
	testConfig(t, nil)
	data := fixtures(t)
	// same first 31 bytes (and characters), different tails
	for i, dv := range data.Dataviews {
		dv.XPath = testXPath(t, strings.Repeat("é", 35)+string(rune('a'+i)), "s", "t", "dv")
	}
	config.Set(globalCf, "xlsx.sheetname", "${entity}")

	out, err := createXLSX(globalCf, data)
	if err != nil {
		t.Fatal(err)
	}
	x := openXLSX(t, out)
	sheets := x.GetSheetList()
	if len(sheets) != 3 {
		t.Fatalf("sheets = %q, want 3", sheets)
	}
	seen := map[string]bool{}
	for _, s := range sheets {
		if len([]rune(s)) > 31 {
			t.Errorf("sheet name %q too long", s)
		}
		if seen[strings.ToLower(s)] {
			t.Errorf("duplicate sheet name %q", s)
		}
		seen[strings.ToLower(s)] = true
	}
}

func TestEntitySplitIsOrdered(t *testing.T) {
	data := fixtures(t)
	for range 20 {
		var got []string
		for _, g := range groupByEntity(data.Dataviews) {
			got = append(got, g.entity)
		}
		if want := []string{"alpha", "mid", "zeta"}; !slices.Equal(got, want) {
			t.Fatalf("entities = %v, want %v", got, want)
		}
	}

	testConfig(t, map[string]any{"xlsx.split": "entity", "texttable.split": "entity", "html.split": "entity"})
	for range 10 {
		files, err := buildXLSXFiles(globalCf, data, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, f := range files {
			names = append(names, f.name)
		}
		if !slices.IsSorted(names) || len(names) != 3 {
			t.Fatalf("xlsx names = %v, want 3 sorted", names)
		}
		files, err = buildTextTableFiles(globalCf, data, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		names = names[:0]
		for _, f := range files {
			names = append(names, f.name)
		}
		if !slices.IsSorted(names) || len(names) != 3 {
			t.Fatalf("text names = %v, want 3 sorted", names)
		}
	}
}

func TestWriteFiles(t *testing.T) {
	testConfig(t, map[string]any{"files": []string{"xlsx", "html", "texttable"}})
	dir := filepath.Join(t.TempDir(), "out")

	if err := writeFiles(dir, fixtures(t)); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []string{"*.xlsx", "*.html", "*.txt"} {
		m, _ := filepath.Glob(filepath.Join(dir, pattern))
		if len(m) != 1 {
			t.Errorf("files matching %s = %v", pattern, m)
			continue
		}
		if st, err := os.Stat(m[0]); err != nil || st.Size() == 0 {
			t.Errorf("%s empty or missing: %v", m[0], err)
		}
	}
}

func TestEmbeddedImages(t *testing.T) {
	c := testConfigYAML(t, "images:\n  logo.png: /a/logo.png\n  banner.top.gif: /a/b.gif\n  plain: /a/p\n", nil)
	got := embeddedImages(c)
	want := map[string]string{"logo.png": "/a/logo.png", "banner.top.gif": "/a/b.gif", "plain": "/a/p"}
	if !maps.Equal(got, want) {
		t.Errorf("images = %v, want %v", got, want)
	}
}

func TestBuildMessage(t *testing.T) {
	logo := filepath.Join(t.TempDir(), "logo.png")
	if err := os.WriteFile(logo, []byte("not really a png"), 0600); err != nil {
		t.Fatal(err)
	}

	em := config.New()
	config.Set(em, "_to", "to@example.com")
	config.Set(em, "_from", "from@example.com")

	t.Run("default logo is embedded when images are enabled", func(t *testing.T) {
		testConfig(t, nil)
		m, err := buildMessage(globalCf, em, fixtures(t))
		if err != nil {
			t.Fatal(err)
		}
		embeds := m.GetEmbeds()
		if len(embeds) != 1 || embeds[0].Name != "logo.png" {
			t.Fatalf("embedded files = %v, want the default logo.png", embeds)
		}
		var message bytes.Buffer
		if _, err := m.WriteTo(&message); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(message.String(), "Content-Id: <logo.png>") {
			t.Fatal("default logo has no matching Content-ID")
		}
		var image bytes.Buffer
		if _, err := embeds[0].Writer(&image); err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(image.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
			t.Fatal("default logo is not a PNG image")
		}
	})

	t.Run("images use the file name as content id", func(t *testing.T) {
		testConfigYAML(t, "email:\n  contents: [text+html, images]\nimages:\n  logo.png: "+logo+"\n  other.png: "+logo+"\n", nil)
		m, err := buildMessage(globalCf, em, fixtures(t))
		if err != nil {
			t.Fatal(err)
		}
		if got := len(m.GetEmbeds()); got != 2 {
			t.Fatalf("embedded files = %d, want custom logo and other image only", got)
		}
		var buf bytes.Buffer
		if _, err := m.WriteTo(&buf); err != nil {
			t.Fatal(err)
		}
		msg := buf.String()
		if strings.Contains(strings.ToLower(msg), "x-attachment-id") {
			t.Error("message has a message-level X-Attachment-Id header")
		}
		for _, id := range []string{"<logo.png>", "<other.png>"} {
			if !strings.Contains(msg, "Content-Id: "+id) {
				t.Errorf("no Content-ID %s in message", id)
			}
		}
	})

	for _, key := range []string{"text.template", "html.template"} {
		t.Run("template error in "+key, func(t *testing.T) {
			testConfig(t, map[string]any{
				"email.contents": []string{"text+html"},
				key:              "{{.NoSuchField.Sub}}",
			})
			if _, err := buildMessage(globalCf, em, fixtures(t)); err == nil {
				t.Error("expected the template error to be returned")
			}
		})
	}
}
