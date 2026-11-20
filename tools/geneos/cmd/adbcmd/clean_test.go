package adbcmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/xpath"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"
)

func TestCleanMutationReports(t *testing.T) {
	cmdtest.Env(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "source.adb")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	dashboard, err := archive.Create("panel" + extension)
	if err != nil {
		t.Fatal(err)
	}
	xp := `/geneos/gateway[(@name="Old")]/directory/probe[(@name="P")]/managedEntity[(@name="E")]`
	modifier := strings.Join([]string{dataItemExprNodeClass, "7.0", "$0$", "Value", "dataitem(" + xp + ")wwwpage(http://example.invalid)", "saved-alias", "1"}, "~")
	link := geneosURLClass + `~3.0~wwwpage(http://example.invalid)~Example~`
	fmt.Fprintln(dashboard, modifier)
	fmt.Fprintln(dashboard, link)
	fmt.Fprintln(dashboard, dataItemExprNodeClass+"~short")
	if _, err := archive.Create("startup.dbo"); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(dir, "changes.log")
	if err := os.WriteFile(logPath, []byte("previous\n"), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.adb")
	result := cmdtest.Run(t, cmd.Cmd, "adb", "clean", "-v", "-l", logPath, "-o", filepath.Join(dir, "out"), "--wwwpage", "replace:#^http:#https:#", path, missing)
	if result.Err == nil {
		t.Fatal("expected an error for the missing input")
	}
	if result.Stdout != "" || result.Stderr != "" {
		t.Errorf("output was not redirected: stdout=%q stderr=%q", result.Stdout, result.Stderr)
	}
	report, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"previous\n", path + ": ", "panel.Activedashboard:1: dataitem:", strconv.Quote(xp),
		"panel.Activedashboard:1: wwwpage:", "panel.Activedashboard:2: wwwpage:",
		`"http://example.invalid" -> "https://example.invalid"`,
		`modifier alias cleared: "saved-alias" -> "$0$"`,
		`startup-data entry removed: "startup.dbo" -> ""`, missing + ": ",
		"Too few fields in modifier line",
	} {
		if !strings.Contains(string(report), want) {
			t.Errorf("report missing %q:\n%s", want, report)
		}
	}

	outputPath := filepath.Join(dir, "out", filepath.Base(path))
	verboseArchive, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	result = cmdtest.Run(t, cmd.Cmd, "adb", "clean", "-o", filepath.Join(dir, "out"), "--wwwpage", "replace:#^http:#https:#", path)
	if result.Err != nil {
		t.Fatalf("clean without verbose: %v; stderr=%s", result.Err, result.Stderr)
	}
	if !strings.Contains(result.Stdout, path+": ") || strings.Contains(result.Stdout, "panel.Activedashboard:") {
		t.Errorf("unexpected non-verbose report: %q", result.Stdout)
	}
	normalArchive, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(verboseArchive, normalArchive) {
		t.Error("verbose reporting changed the cleaned archive")
	}

	result = cmdtest.Run(t, cmd.Cmd, "adb", "clean", "-v", "-o", filepath.Join(dir, "out"), "--gateway", "keep", "--probe", "keep", "--entity", "keep", "--type", "keep", "--keep-startup-data", path)
	if result.Err != nil {
		t.Fatalf("verbose clean with no changes: %v; stderr=%s", result.Err, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "No links updated") || strings.Contains(result.Stdout, "panel.Activedashboard:") || strings.Contains(result.Stdout, "startup.dbo:") {
		t.Errorf("unchanged records should not produce mutation details: %q", result.Stdout)
	}

	result = cmdtest.Run(t, cmd.Cmd, "adb", "clean", "-v", "-o", filepath.Join(dir, "out"), path)
	if result.Err != nil || !strings.Contains(result.Stdout, "panel.Activedashboard:1: dataitem:") {
		t.Errorf("verbose console output = %q, error = %v", result.Stdout, result.Err)
	}

	if _, err := os.Stat("/dev/full"); err == nil {
		result = cmdtest.Run(t, cmd.Cmd, "adb", "clean", "-l", "/dev/full", "-o", filepath.Join(dir, "out"), path)
		if result.Err == nil || !strings.Contains(result.Err.Error(), "write log") {
			t.Errorf("full log device: got error %v, want a log write error", result.Err)
		}
	}
}

func TestCleanLogRejectsDashboardPaths(t *testing.T) {
	cmdtest.Env(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "source.adb")
	if err := os.WriteFile(source, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}
	for _, logPath := range []string{source, alias, filepath.Join(dir, "out", "source.adb"), filepath.Join(dir, "out", "source.adb.cleaned")} {
		result := cmdtest.Run(t, cmd.Cmd, "adb", "clean", "-l", logPath, "-o", filepath.Join(dir, "out"), source)
		if result.Err == nil || !strings.Contains(result.Err.Error(), "conflicts with dashboard") {
			t.Errorf("log path %q: got error %v, want conflict", logPath, result.Err)
		}
	}
	got, err := os.ReadFile(source)
	if err != nil || string(got) != "original" {
		t.Fatalf("source changed: %q, %v", got, err)
	}
}

func TestApplyCleanDefaults(t *testing.T) {
	originalOptions := cleanOptions
	originalKeepAll := cleanCmdKeepAll
	originalKeepStartupData := cleanCmdKeepStartupData
	defer func() {
		cleanOptions = originalOptions
		cleanCmdKeepAll = originalKeepAll
		cleanCmdKeepStartupData = originalKeepStartupData
	}()

	cleanOptions = cleaningOptions{
		Gateways: optionSlice{"remove-all"},
		Probes:   optionSlice{"remove-all"},
		Entities: optionSlice{"remove-all"},
		Types:    optionSlice{"remove-empty"},
	}
	cleanCmdKeepAll = false
	cleanCmdKeepStartupData = false
	applyCleanDefaults(true, false)
	if cleanOptions.Gateways[0] != "keep" || cleanOptions.Probes[0] != "keep" || cleanOptions.Entities[0] != "keep" {
		t.Fatalf("edit name defaults = %v, %v, %v; want keep", cleanOptions.Gateways, cleanOptions.Probes, cleanOptions.Entities)
	}
	if cleanOptions.Types[0] != "keep" {
		t.Errorf("edit Type default = %q, want keep", cleanOptions.Types[0])
	}
	if !cleanCmdKeepStartupData {
		t.Error("edit should preserve startup data by default")
	}

	cleanOptions.Types = optionSlice{"remove-empty", "set:System"}
	cleanCmdKeepStartupData = false
	applyCleanDefaults(true, true)
	if cleanOptions.Types[1] != "set:System" {
		t.Errorf("explicit Type action = %q, want set:System", cleanOptions.Types[1])
	}
	if cleanCmdKeepStartupData {
		t.Error("explicit startup-data choice should be preserved")
	}

	cleanOptions = cleaningOptions{
		Gateways: optionSlice{"remove-all"},
		Probes:   optionSlice{"remove-all"},
		Entities: optionSlice{"remove-all"},
		Types:    optionSlice{"remove-empty"},
	}
	cleanCmdKeepAll = true
	cleanCmdKeepStartupData = false
	applyCleanDefaults(false, false)
	if cleanOptions.Gateways[0] != "keep" || cleanOptions.Types[0] != "remove-empty" {
		t.Errorf("regular --keep-all defaults = gateway %q, type %q; want keep and remove-empty", cleanOptions.Gateways[0], cleanOptions.Types[0])
	}
	if cleanCmdKeepStartupData {
		t.Error("regular --keep-all should not preserve startup data by itself")
	}
}

func TestCleanEditAliasIsRegistered(t *testing.T) {
	command, _, err := adbCmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("adbCmd.Find(edit) error = %v", err)
	}
	if command != cleanCmd {
		t.Fatalf("edit resolves to %v, want clean command", command)
	}
}

func TestUpdateDataitemXpathTypes(t *testing.T) {
	typeValue := func(value string) *string { return &value }
	tests := []struct {
		name     string
		original *string
		actions  optionSlice
		want     *string
	}{
		{name: "default preserves non-empty", original: typeValue("System"), actions: optionSlice{"remove-empty"}, want: typeValue("System")},
		{name: "default removes empty", original: typeValue(""), actions: optionSlice{"remove-empty"}},
		{name: "default preserves absent", actions: optionSlice{"remove-empty"}},
		{name: "keep preserves non-empty", original: typeValue("System"), actions: optionSlice{"keep"}, want: typeValue("System")},
		{name: "keep preserves empty", original: typeValue(""), actions: optionSlice{"keep"}, want: typeValue("")},
		{name: "keep preserves absent", actions: optionSlice{"keep"}},
		{name: "no actions preserves non-empty", original: typeValue("System"), want: typeValue("System")},
		{name: "no actions preserves empty", original: typeValue(""), want: typeValue("")},
		{name: "non-matching removal preserves", original: typeValue("System"), actions: optionSlice{"remove:Application*"}, want: typeValue("System")},
		{name: "matching removal removes", original: typeValue("System"), actions: optionSlice{"remove:Sys*"}},
		{name: "remove-all removes", original: typeValue("System"), actions: optionSlice{"remove-all"}},
		{name: "set creates absent", actions: optionSlice{"set:System"}, want: typeValue("System")},
		{name: "set-empty creates empty", actions: optionSlice{"set-empty"}, want: typeValue("")},
		{name: "replacement preserves absence", actions: optionSlice{"replace:/^/Prefix/"}},
		{name: "unknown action preserves", original: typeValue("System"), actions: optionSlice{"unknown"}, want: typeValue("System")},
		{name: "set then replace", original: typeValue("System"), actions: optionSlice{"set:Demo-Application", "replace:/^Demo-/Production-/"}, want: typeValue("Production-Application")},
		{name: "chained replacements", original: typeValue("Demo-System"), actions: optionSlice{"replace:/^Demo-/Production-/", "replace:/Production/Live/"}, want: typeValue("Live-System")},
		{name: "set then matching removal", original: typeValue("System"), actions: optionSlice{"set:Application", "remove:App*"}},
		{name: "set-empty then remove-empty", original: typeValue("System"), actions: optionSlice{"set-empty", "remove-empty"}},
		{name: "remove then keep stays absent", original: typeValue("System"), actions: optionSlice{"remove-all", "keep"}},
		{name: "remove then set", original: typeValue("System"), actions: optionSlice{"remove-all", "set:Application"}, want: typeValue("Application")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var originalValue string
			if test.original != nil {
				originalValue = *test.original
			}
			input := &xpath.XPath{Sampler: &xpath.Sampler{Name: "CPU", Type: test.original}}
			got := updateDataitemXpath(input, cleaningOptions{Types: test.actions})
			if got.Sampler == nil {
				t.Fatal("Sampler was removed")
			}
			actual := got.Sampler.Type
			if (actual == nil) != (test.want == nil) {
				t.Fatalf("Type present = %t, want %t", actual != nil, test.want != nil)
			}
			if actual != nil && *actual != *test.want {
				t.Fatalf("Type = %q, want %q", *actual, *test.want)
			}
			if test.original != nil && *test.original != originalValue {
				t.Fatalf("original Type was modified: %q, want %q", *test.original, originalValue)
			}
		})
	}

	t.Run("no Type action does not create Sampler", func(t *testing.T) {
		got := updateDataitemXpath(&xpath.XPath{}, cleaningOptions{})
		if got.Sampler != nil {
			t.Fatal("Sampler was created without an action")
		}
	})
	t.Run("set Type creates Sampler", func(t *testing.T) {
		got := updateDataitemXpath(&xpath.XPath{}, cleaningOptions{Types: optionSlice{"set:System"}})
		if got.Sampler == nil || got.Sampler.Type == nil || *got.Sampler.Type != "System" {
			t.Fatal("Sampler with Type System was not created")
		}
	})
}

func TestUpdateDataitemXpathAttributes(t *testing.T) {
	t.Run("add initializes nil Attribute map", func(t *testing.T) {
		input := &xpath.XPath{Entity: &xpath.Entity{Name: "server"}}
		got := updateDataitemXpath(input, cleaningOptions{
			Attributes: optionSlice{"add:Location=London"},
		})
		if got.Entity == nil {
			t.Fatal("Managed Entity was removed")
		}
		if got.Entity.Attributes["Location"] != "London" {
			t.Fatalf("Attributes = %v, want Location=London", got.Entity.Attributes)
		}
	})

	t.Run("Attribute action does not create missing Managed Entity", func(t *testing.T) {
		got := updateDataitemXpath(&xpath.XPath{}, cleaningOptions{
			Attributes: optionSlice{"add:Location=London"},
		})
		if got.Entity != nil {
			t.Fatalf("Managed Entity was created: %#v", got.Entity)
		}
	})

	t.Run("explicit Managed Entity action allows Attribute addition", func(t *testing.T) {
		got := updateDataitemXpath(&xpath.XPath{}, cleaningOptions{
			Entities:   optionSlice{"set:server"},
			Attributes: optionSlice{"add:Location=London"},
		})
		if got.Entity == nil {
			t.Fatal("Managed Entity was not created")
		}
		if got.Entity.Name != "server" || got.Entity.Attributes["Location"] != "London" {
			t.Fatalf("Managed Entity = %#v, want server with Location=London", got.Entity)
		}
	})
}

func TestCleanActivedashboardModifierAlias(t *testing.T) {
	originalFilter := cleanFilterXPath
	originalOptions := cleanOptions
	originalSkipModifiers := cleanCmdSkipModifiers
	originalLog := log
	defer func() {
		cleanFilterXPath = originalFilter
		cleanOptions = originalOptions
		cleanCmdSkipModifiers = originalSkipModifiers
		log = originalLog
	}()

	cleanCmdSkipModifiers = false
	log = slog.Default()
	dataitemPath := `/geneos/gateway[(@name="Other")]/directory/probe[(@name="Probe")]/managedEntity[(@name="Entity")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]`
	tests := []struct {
		name          string
		filterGateway string
		gatewayAction optionSlice
		paths         []string
		wantPaths     []string
		wantAlias     string
	}{
		{
			name:          "filtered-out path preserves alias",
			filterGateway: "Target",
			gatewayAction: optionSlice{"set:Updated"},
			paths:         []string{dataitemPath},
			wantPaths:     []string{dataitemPath},
			wantAlias:     "saved-alias",
		},
		{
			name:          "changed path clears alias",
			filterGateway: "Other",
			gatewayAction: optionSlice{"set:Updated"},
			paths:         []string{dataitemPath},
			wantPaths:     []string{`/geneos/gateway[(@name="Updated")]/directory/probe[(@name="Probe")]/managedEntity[(@name="Entity")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]`},
			wantAlias:     EMPTY_STRING,
		},
		{
			name:          "unchanged matched path preserves alias",
			filterGateway: "Other",
			gatewayAction: optionSlice{"keep"},
			paths:         []string{dataitemPath},
			wantPaths:     []string{dataitemPath},
			wantAlias:     "saved-alias",
		},
		{
			name:          "changed path clears shared alias when another path is filtered out",
			filterGateway: "Other",
			gatewayAction: optionSlice{"set:Updated"},
			paths: []string{
				dataitemPath,
				strings.Replace(dataitemPath, `name="Other"`, `name="Target"`, 1),
			},
			wantPaths: []string{
				`/geneos/gateway[(@name="Updated")]/directory/probe[(@name="Probe")]/managedEntity[(@name="Entity")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]`,
				strings.Replace(dataitemPath, `name="Other"`, `name="Target"`, 1),
			},
			wantAlias: EMPTY_STRING,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanFilterXPath = &xpath.XPath{Gateway: &xpath.Gateway{Name: test.filterGateway}}
			cleanOptions = cleaningOptions{Gateways: test.gatewayAction}
			dataitems := make([]string, len(test.paths))
			wantDataitems := make([]string, len(test.wantPaths))
			for i, path := range test.paths {
				dataitems[i] = "dataitem(" + path + ")"
			}
			for i, path := range test.wantPaths {
				wantDataitems[i] = "dataitem(" + path + ")"
			}
			line := strings.Join([]string{
				dataItemExprNodeClass,
				"7.0",
				"$0$",
				"Value",
				strings.Join(dataitems, ""),
				"saved-alias",
				"1",
			}, string(fieldSeparator))

			var output strings.Builder
			if err := cleanActivedashboard(strings.NewReader(line), &output); err != nil {
				t.Fatalf("cleanActivedashboard() error = %v", err)
			}
			parts := strings.Split(strings.TrimSuffix(output.String(), "\n"), string(fieldSeparator))
			wantURLTarget := strings.Join(wantDataitems, "")
			if parts[4] != wantURLTarget {
				t.Errorf("URLTarget = %q, want %q", parts[4], wantURLTarget)
			}
			if parts[5] != test.wantAlias {
				t.Errorf("URLAlias = %q, want %q", parts[5], test.wantAlias)
			}
		})
	}
}

func TestCleanActivedashboardWebPageTargets(t *testing.T) {
	originalFilter := cleanFilterXPath
	originalOptions := cleanOptions
	originalSkipModifiers := cleanCmdSkipModifiers
	originalSkipLinks := cleanCmdSkipLinks
	originalURLFilters := cleanURLFilters
	originalLog := log
	defer func() {
		cleanFilterXPath = originalFilter
		cleanOptions = originalOptions
		cleanCmdSkipModifiers = originalSkipModifiers
		cleanCmdSkipLinks = originalSkipLinks
		cleanURLFilters = originalURLFilters
		log = originalLog
	}()

	cleanFilterXPath = nil
	cleanOptions = cleaningOptions{
		Gateways: optionSlice{"set:Updated"},
		WebPages: optionSlice{"replace:#^http://([^/]+)#https://${1}#"},
	}
	cleanCmdSkipModifiers = false
	cleanCmdSkipLinks = false
	cleanURLFilters = []*regexp.Regexp{
		regexp.MustCompile(`^https://other\.example/`),
		regexp.MustCompile(`^http://host\.example/`),
	}
	log = slog.Default()

	dataitemPath := `/geneos/gateway[(@name="Old")]/directory/probe[(@name="Probe")]/managedEntity[(@name="Entity")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]`
	updatedPath := strings.Replace(dataitemPath, `name="Old"`, `name="Updated"`, 1)
	unknownTarget := `droppedItem()/replace(/cell[(@column="field")])`
	modifierLine := strings.Join([]string{
		dataItemExprNodeClass,
		"7.0",
		"$0$",
		"Value",
		"dataitem(" + dataitemPath + ")wwwpage(http://host.example/path)" + unknownTarget,
		"saved-alias",
		"1",
	}, string(fieldSeparator))

	var output strings.Builder
	if err := cleanActivedashboard(strings.NewReader(modifierLine), &output); err != nil {
		t.Fatalf("cleanActivedashboard() error = %v", err)
	}
	modifierFields := strings.Split(strings.TrimSuffix(output.String(), "\n"), string(fieldSeparator))
	wantTarget := "dataitem(" + updatedPath + ")wwwpage(https://host.example/path)" + unknownTarget
	if modifierFields[4] != wantTarget {
		t.Errorf("modifier URLTarget = %q, want %q", modifierFields[4], wantTarget)
	}
	if modifierFields[5] != EMPTY_STRING {
		t.Errorf("modifier URLAlias = %q, want %q", modifierFields[5], EMPTY_STRING)
	}

	filteredLine := strings.Replace(modifierLine, "http://host.example/path", "http://unmatched.example/path", 1)
	output.Reset()
	if err := cleanActivedashboard(strings.NewReader(filteredLine), &output); err != nil {
		t.Fatalf("cleanActivedashboard() with filtered URL error = %v", err)
	}
	filteredFields := strings.Split(strings.TrimSuffix(output.String(), "\n"), string(fieldSeparator))
	if !strings.Contains(filteredFields[4], "wwwpage(http://unmatched.example/path)") {
		t.Errorf("non-matching wwwpage URL was changed: %q", filteredFields[4])
	}

	geneosLine := strings.Join([]string{
		geneosURLClass,
		"3.0",
		"wwwpage(http://host.example/path)",
		"Example",
		"",
	}, string(fieldSeparator))
	output.Reset()
	if err := cleanActivedashboard(strings.NewReader(geneosLine), &output); err != nil {
		t.Fatalf("cleanActivedashboard() for GeneosURL error = %v", err)
	}
	geneosFields := strings.Split(strings.TrimSuffix(output.String(), "\n"), string(fieldSeparator))
	if geneosFields[2] != "wwwpage(https://host.example/path)" {
		t.Errorf("GeneosURL target = %q, want wwwpage(https://host.example/path)", geneosFields[2])
	}

	concatenatedLine := geneosURLClass + `~3.0~wwwpage(http://host.example/path)wwwpage(http://host.example/second)~Example~`
	output.Reset()
	if err := cleanActivedashboard(strings.NewReader(concatenatedLine), &output); err != nil {
		t.Fatalf("cleanActivedashboard() for concatenated GeneosURL error = %v", err)
	}
	if got := strings.TrimSuffix(output.String(), "\n"); got != concatenatedLine {
		t.Errorf("concatenated GeneosURL = %q, want unchanged %q", got, concatenatedLine)
	}
}

func TestCleanActivedashboardStats(t *testing.T) {
	originalFilter := cleanFilterXPath
	originalOptions := cleanOptions
	originalSkipModifiers := cleanCmdSkipModifiers
	originalSkipLinks := cleanCmdSkipLinks
	originalURLFilters := cleanURLFilters
	originalLog := log
	defer func() {
		cleanFilterXPath = originalFilter
		cleanOptions = originalOptions
		cleanCmdSkipModifiers = originalSkipModifiers
		cleanCmdSkipLinks = originalSkipLinks
		cleanURLFilters = originalURLFilters
		log = originalLog
	}()

	cleanFilterXPath = nil
	cleanOptions = cleaningOptions{
		Gateways: optionSlice{"set:Updated"},
		WebPages: optionSlice{"replace:#^http://#https://#"},
	}
	cleanCmdSkipModifiers = false
	cleanCmdSkipLinks = false
	cleanURLFilters = nil
	log = slog.Default()

	dataitemPath := `/geneos/gateway[(@name="Old")]/directory/probe[(@name="Probe")]/managedEntity[(@name="Entity")]/sampler[(@name="CPU")]/dataview[(@name="CPU")]`
	modifierLine := strings.Join([]string{
		dataItemExprNodeClass,
		"7.0",
		"$0$",
		"Value",
		"dataitem(" + dataitemPath + ")wwwpage(http://example.invalid/path)",
		"saved-alias",
		"1",
	}, string(fieldSeparator))
	geneosLine := strings.Join([]string{
		geneosURLClass,
		"3.0",
		"dataitem(" + dataitemPath + ")",
		"Example",
		"",
	}, string(fieldSeparator))

	var output strings.Builder
	stats := cleanStats{}
	if err := cleanActivedashboardWithStats(strings.NewReader(modifierLine+"\n"+geneosLine), &output, &stats); err != nil {
		t.Fatalf("cleanActivedashboardWithStats() error = %v", err)
	}
	if stats.xpathLinks != 2 || stats.urlLinks != 1 {
		t.Errorf("updated links = %d XPaths, %d URLs; want 2 XPaths, 1 URL", stats.xpathLinks, stats.urlLinks)
	}
	if stats.gatewaysChanged != 2 || stats.modifierAliasesCleared != 1 {
		t.Errorf("changes = %d Gateways, %d modifier aliases; want 2 and 1", stats.gatewaysChanged, stats.modifierAliasesCleared)
	}
	wantSummary := "3 links updated (2 XPaths, 1 URL); XPath changes: 2 Gateways; 1 modifier alias cleared"
	if got := stats.summary(); got != wantSummary {
		t.Errorf("summary = %q, want %q", got, wantSummary)
	}

	if got := (cleanStats{startupDataRemoved: 2}).summary(); got != "No links updated; 2 startup-data entries removed" {
		t.Errorf("no-link summary = %q", got)
	}
}

func TestUpdateDataitemXpathDoesNotAddEmptySampler(t *testing.T) {
	xp, err := xpath.Parse(`/geneos/gateway[(@name="GW")]/directory/probe[(@name="P")]/managedEntity[(@name="E")]`)
	if err != nil {
		t.Fatal(err)
	}
	got := updateDataitemXpath(xp, cleaningOptions{Samplers: optionSlice{"keep"}, Types: optionSlice{"remove-empty"}}).String()
	if strings.Contains(got, "sampler") {
		t.Errorf("path = %s, want no sampler added", got)
	}
}

func TestPluralCount(t *testing.T) {
	for _, test := range []struct {
		count    int
		singular string
		want     string
	}{
		{1, "Managed Entity", "1 Managed Entity"},
		{2, "Managed Entity", "2 Managed Entities"},
		{2, "startup-data entry", "2 startup-data entries"},
		{2, "modifier alias", "2 modifier aliases"},
		{3, "Sampler", "3 Samplers"},
	} {
		if got := pluralCount(test.count, test.singular); got != test.want {
			t.Errorf("pluralCount(%d, %q) = %q, want %q", test.count, test.singular, got, test.want)
		}
	}
}

func TestCleanActivedashboardShortModifierLine(t *testing.T) {
	originalLog := log
	log = slog.Default()
	defer func() { log = originalLog }()

	// a short line is passed through and does not panic
	line := dataItemExprNodeClass + "~7.0~$0$"
	var out strings.Builder
	if err := cleanActivedashboard(strings.NewReader(line+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != line+"\n" {
		t.Errorf("output = %q, want the line unchanged", out.String())
	}
}

func TestValidateReplaceSpecs(t *testing.T) {
	if err := validateReplaceSpecs(cleaningOptions{Gateways: optionSlice{"replace:/a/b/"}, Attributes: optionSlice{"replace:NAME=/a/b/", "replace:NAME=value"}, WebPages: optionSlice{"replace:#a#b#"}}); err != nil {
		t.Errorf("valid specs rejected: %v", err)
	}
	for _, opts := range []cleaningOptions{
		{Gateways: optionSlice{"replace:/a/b"}},
		{Dataviews: optionSlice{"replace:/(/b/"}},
		{Attributes: optionSlice{"replace:NAME=/a"}},
		{WebPages: optionSlice{"replace:x"}},
	} {
		if err := validateReplaceSpecs(opts); err == nil {
			t.Errorf("invalid specs accepted: %+v", opts)
		}
	}
}
