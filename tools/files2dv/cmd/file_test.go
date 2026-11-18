package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/internal/defaultscheck"
)

// testDataview loads the YAML, which is merged over the embedded
// defaults (that must be valid), and returns the first dataview
func testDataview(t *testing.T, y string) *config.Config {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	c, err := config.Read("files2dv",
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
	// the first dataview from the embedded defaults is replaced by
	// the one in y
	if !c.IsSet("dataviews.0") {
		t.Fatal("no dataview in configuration")
	}
	return c.Sub("dataviews.0")
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func newRegexp(t *testing.T, expr string) *regexp.Regexp {
	t.Helper()
	re, err := regexp.Compile(expr)
	if err != nil {
		t.Fatal(err)
	}
	return re
}

func quote(s string) string { return strconv.Quote(s) }

// there are no embedded defaults yet, but they must still be valid
func TestDefaultsFile(t *testing.T) {
	defaultscheck.Parse(t, defaults)
}

// the sample configuration must also parse strictly, have the expected
// keys and only use settings that the code reads
func TestSampleConfig(t *testing.T) {
	sample, err := os.ReadFile("../files2dv.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m := defaultscheck.Parse(t, sample)
	defaultscheck.RequireKeys(t, m, "dataviews", "push.type", "push.netprobe", "push.secure", "push.interval")

	// push mode is not yet implemented, so the settings are only
	// documentation of the planned configuration
	defaultscheck.CheckRead(t, m, []string{"."}, "push")

	var cfg struct {
		Dataviews []map[string]any `yaml:"dataviews"`
	}
	if err := yaml.Unmarshal(sample, &cfg); err != nil {
		t.Fatal(err)
	}
	known := []string{"name", "row-limit", "paths", "types", "ignore-file-errors", "ignore-lines", "max-lines", "columns", "values", "headlines", "on-fail"}
	for _, d := range cfg.Dataviews {
		for k := range d {
			if !slices.Contains(known, k) {
				t.Errorf("sample dataview %v has unknown setting %q", d["name"], k)
			}
		}
	}
}

func TestProcessFilesMatchesPerFile(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"a.txt": "header\nORDER 1\nother\n",
		"b.txt": "header\nORDER 7\n",
		"c.txt": "nothing to see\n",
		"d.txt": "ORDER 9\n",
	})

	dv := testDataview(t, `
dataviews:
  - name: orders
    paths: [`+quote(filepath.Join(dir, "*.txt"))+`]
    columns:
      - name: filename
        value: ${filename}
      - name: order
        match: '^ORDER (\d+)'
        value: ${1}
        fail: NONE
`)
	dataview, err := processFiles(dv)
	if err != nil {
		t.Fatal(err)
	}

	want := [][]string{
		{"filename", "order"},
		{"a.txt", "1"},
		{"b.txt", "7"},
		{"c.txt", "NONE"},
		{"d.txt", "9"},
	}
	if !slices.EqualFunc(dataview.Table, want, slices.Equal) {
		t.Errorf("table = %v, want %v", dataview.Table, want)
	}
}

func TestProcessFilesSkipDropsRow(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"a.txt": "x", "b.txt": "y"})
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0700); err != nil {
		t.Fatal(err)
	}

	dv := testDataview(t, `
dataviews:
  - name: files
    paths: [`+quote(filepath.Join(dir, "*"))+`]
    types: [file]
    columns: [ name, type ]
    values: [ "${filename}", "${type}" ]
`)
	dataview, err := processFiles(dv)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"name", "type"}, {"a.txt", "file"}, {"b.txt", "file"}}
	if !slices.EqualFunc(dataview.Table, want, slices.Equal) {
		t.Errorf("table = %v, want %v", dataview.Table, want)
	}
}

func TestProcessFilesRowLimitKeepsHeadlines(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"a.txt": "x", "b.txt": "y", "c.txt": "z"})

	dv := testDataview(t, `
dataviews:
  - name: files
    row-limit: 2
    paths: [`+quote(filepath.Join(dir, "*.txt"))+`]
    columns: [ name ]
    values: [ "${filename}" ]
    headlines:
      - name: total
        value: limited
`)
	dataview, err := processFiles(dv)
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"name"}, {"a.txt"}, {"b.txt"}}; !slices.EqualFunc(dataview.Table, want, slices.Equal) {
		t.Errorf("table = %v, want %v", dataview.Table, want)
	}
	if len(dataview.Headlines) != 1 || dataview.Headlines[0].Name != "total" || dataview.Headlines[0].Value != "limited" {
		t.Errorf("headlines = %v", dataview.Headlines)
	}
}

func TestProcessFilesPathErrorsDoNotFailTheRest(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"a.txt": "x"})

	dv := testDataview(t, `
dataviews:
  - name: files
    paths: [ "/nonexistent/f-<today %Q>.log", `+quote(filepath.Join(dir, "*.txt"))+`]
    columns: [ name ]
    values: [ "${filename}" ]
`)
	dataview, err := processFiles(dv)
	if err != nil {
		t.Fatalf("unexpected error from an earlier path: %v", err)
	}
	if len(dataview.Table) != 2 {
		t.Errorf("table = %v", dataview.Table)
	}
}

func TestProcessFilesLongLinesAndGroups(t *testing.T) {
	dir := t.TempDir()
	// a line longer than the default scanner limit, then the match
	writeFiles(t, dir, map[string]string{
		"long.txt": strings.Repeat("x", 200*1024) + "\nKEY=value\n",
		"alt.txt":  "second\n",
	})

	dv := testDataview(t, `
dataviews:
  - name: files
    paths: [`+quote(filepath.Join(dir, "*.txt"))+`]
    columns:
      - name: key
        match: '^KEY=(.*)'
        value: ${1}
      - name: alt
        match: '^(first)|(second)$'
        value: "[${1}][${2}]"
`)
	dataview, err := processFiles(dv)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"key", "alt"}, {"", "[][second]"}, {"value", ""}}
	if !slices.EqualFunc(dataview.Table, want, slices.Equal) {
		t.Errorf("table = %q, want %q", dataview.Table, want)
	}
}

func TestSubmatchLookupUnmatchedGroups(t *testing.T) {
	re := newRegexp(t, `(?P<a>x)|(?P<b>y)`)
	line := "y"
	got := submatchLookup(re, line, re.FindStringSubmatchIndex(line))
	for k, want := range map[string]string{"0": "y", "1": "", "2": "y", "a": "", "b": "y"} {
		if got[k] != want {
			t.Errorf("lookup[%q] = %q, want %q", k, got[k], want)
		}
	}
}

func TestScanFileEmptyMatchValue(t *testing.T) {
	dv := testDataview(t, `
dataviews:
  - name: files
    max-lines: 3
    columns: [ a ]
    values: [ "x" ]
`)
	columns := []Column{
		{Name: "a", Match: newRegexp(t, `^a`), Value: ""},
		{Name: "b", Match: newRegexp(t, `^b=(.*)`), Value: "${1}"},
	}
	values := make([]string, 2)
	// column a matches with an empty value on line one and must not be
	// counted again, stopping the scan before the last line
	err := scanFile(dv, strings.NewReader("a\nb=found\nb=again\nb=late\n"), columns, nil, nil, values)
	if err != nil {
		t.Fatal(err)
	}
	if values[1] != "found" {
		t.Errorf("values = %q", values)
	}
}

func TestPushNotImplemented(t *testing.T) {
	if err := pushCmd.RunE(pushCmd, nil); err == nil {
		t.Error("push should return an error until implemented")
	}
}

func TestHelpText(t *testing.T) {
	if FILE2DVCmd.Use != "files2dv" {
		t.Errorf("Use = %q", FILE2DVCmd.Use)
	}
	usage := FILE2DVCmd.PersistentFlags().Lookup("config").Usage
	if strings.Contains(usage, "dv2email") || !strings.Contains(usage, execname+".yaml") {
		t.Errorf("config flag usage = %q", usage)
	}
}

func TestFileLookupVariables(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("variables are checked for Linux")
	}
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"a.txt": "x"})
	dv := testDataview(t, `
dataviews:
  - name: files
    columns: [ a ]
    values: [ x ]
`)
	lookup, skip := buildFileLookupTable(dv, filepath.Join(dir, "a.txt"), "pattern")
	if skip {
		t.Fatal("skipped")
	}
	for _, k := range []string{"path", "fullpath", "filename", "status", "owner", "user", "uid", "gid", "group", "inode", "device", "size", "modtime", "type"} {
		if lookup[k] == "" {
			t.Errorf("lookup[%q] is empty: %v", k, lookup)
		}
	}
	if lookup["owner"] != lookup["user"] {
		t.Errorf("owner = %q, user = %q", lookup["owner"], lookup["user"])
	}
}
