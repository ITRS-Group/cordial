package includes

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	geneosxml "github.com/itrs-group/cordial/pkg/geneos"
	genecmd "github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func TestFixIncludeLocation(t *testing.T) {
	h := newTestGatewayHost(t)
	home := h.PathTo("gateway", "gateways", "test")
	shared := h.PathTo("gateway", "includes", "shared.xml")
	if err := os.MkdirAll(filepath.Dir(shared), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shared, []byte("<gateway/>"), 0644); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "shared.xml")
	if err := os.WriteFile(external, []byte("<gateway/>"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		location   string
		candidates []string
		want       string
	}{
		{name: "relative", location: "missing/shared.xml", candidates: []string{shared}, want: "../../includes/shared.xml"},
		{name: "absolute", location: "missing/shared.xml", candidates: []string{external}, want: external},
		{name: "valid", location: shared, candidates: []string{external}, want: shared},
		{name: "http", location: "https://example.invalid/shared.xml", candidates: []string{shared}, want: "https://example.invalid/shared.xml"},
		{name: "ambiguous", location: "missing/shared.xml", candidates: []string{shared, external}, want: "missing/shared.xml"},
		{name: "missing", location: "missing/shared.xml", want: "missing/shared.xml"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := fixIncludeLocation(h, home, test.location, map[string][]string{"shared.xml": test.candidates})
			if err != nil || got != test.want {
				t.Fatalf("fixIncludeLocation() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestFixGatewayIncludes(t *testing.T) {
	for _, test := range []struct {
		name    string
		dryRun  bool
		pattern string
	}{
		{name: "dry-run", dryRun: true, pattern: "prod-*"},
		{name: "glob", pattern: "prod-*"},
		{name: "exact", pattern: "prod-a"},
		{name: "no-match", pattern: "missing-*"},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newTestGatewayHost(t)
			root := h.PathTo("gateway")
			for _, name := range []string{"includes/v2.0.0/unique.xml", "includes/ambiguous.xml", "shared/ambiguous.xml", "includes/v1.0.0/archived.xml"} {
				filename := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filename, []byte("<gateway/>"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(filepath.Join("v2.0.0", "unique.xml"), filepath.Join(root, "includes", "unique.xml")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("missing.xml", filepath.Join(root, "includes", "dangling.xml")); err != nil {
				t.Fatal(err)
			}
			external := filepath.Join(t.TempDir(), "external.xml")
			if err := os.WriteFile(external, []byte("<gateway/>"), 0644); err != nil {
				t.Fatal(err)
			}
			setup := `<gateway><includes><include><location>old/unique.xml</location></include><includeGroup name="nested" disabled="true"><include><location>old/external.xml</location></include><include><location>old/ambiguous.xml</location></include><include><location>old/missing.xml</location></include><include><location>old/archived.xml</location></include><include><location>https://example.invalid/unique.xml</location></include></includeGroup></includes><customSection><value>keep</value></customSection></gateway>`
			writeTestGatewaySetup(t, h, setup, "prod-a", "prod-b", "test-a")
			otherSetup := `<gateway><includes><include><location>` + external + `</location></include></includes></gateway>`
			if err := os.WriteFile(h.PathTo("gateway", "gateways", "test-a", "gateway.setup.xml"), []byte(otherSetup), 0644); err != nil {
				t.Fatal(err)
			}
			jsonData := []byte(`{"includes":{"1000":"Title | Group | old/unique.xml","2000":["old/external.xml","old/ambiguous.xml"]},"unrelated":{"location":"old/unique.xml","label":"Title | Group | old/unique.xml"},"number":7039,"autostart":true}`)
			for _, name := range []string{"prod-a", "prod-b", "test-a"} {
				if err := os.WriteFile(h.PathTo("gateway", "gateways", name, "gateway.json"), jsonData, 0640); err != nil {
					t.Fatal(err)
				}
			}
			previousHostname, previousContext := genecmd.Hostname, includesFixCmd.Context()
			flag := includesFixCmd.Flags().Lookup("dry-run")
			if flag == nil || flag.Shorthand != "n" {
				t.Fatal("fix must expose --dry-run/-n")
			}
			previousDryRun, previousChanged := flag.Value.String(), flag.Changed
			genecmd.Hostname = "localhost"
			includesFixCmd.SetContext(context.WithValue(context.Background(), genecmd.CmdKey, &genecmd.CmdValType{}))
			t.Cleanup(func() {
				genecmd.Hostname = previousHostname
				includesFixCmd.SetContext(previousContext)
				flag.Value.Set(previousDryRun)
				flag.Changed = previousChanged
			})
			if err := includesFixCmd.Flags().Set("dry-run", strconv.FormatBool(test.dryRun)); err != nil {
				t.Fatal(err)
			}
			if err := genecmd.ParseArgs(includesFixCmd, []string{test.pattern}); err != nil {
				t.Fatal(err)
			}
			if err := includesFixCmd.RunE(includesFixCmd, []string{test.pattern}); err != nil {
				t.Fatalf("fix command error = %v", err)
			}
			for _, name := range []string{"prod-a", "prod-b", "test-a"} {
				home := h.PathTo("gateway", "gateways", name)
				setupPath, jsonPath := filepath.Join(home, "gateway.setup.xml"), filepath.Join(home, "gateway.json")
				changed := !test.dryRun && (test.pattern == "prod-*" && name != "test-a" || test.pattern == name)
				xmlData, err := os.ReadFile(setupPath)
				if err != nil {
					t.Fatal(err)
				}
				updatedJSON, err := os.ReadFile(jsonPath)
				if err != nil {
					t.Fatal(err)
				}
				if !changed {
					original := setup
					if name == "test-a" {
						original = otherSetup
					}
					if string(xmlData) != original || !bytes.Equal(updatedJSON, jsonData) {
						t.Fatalf("dry-run or unselected gateway %q was changed", name)
					}
					if backups, err := filepath.Glob(filepath.Join(home, "*.history")); err != nil || len(backups) != 0 {
						t.Fatalf("unexpected XML backups for %q: %v, %v", name, backups, err)
					}
					if backups, err := filepath.Glob(filepath.Join(home, "*.bak")); err != nil || len(backups) != 0 {
						t.Fatalf("unexpected JSON backups for %q: %v, %v", name, backups, err)
					}
					continue
				}
				fixed, err := geneosxml.ReadGateway(setupPath)
				if err != nil {
					t.Fatal(err)
				}
				locations := make(map[string]bool)
				walkFixIncludes(fixed.Includes.Includes, fixed.Includes.IncludeGroups, func(location *string) { locations[*location] = true })
				for _, location := range []string{"../../includes/unique.xml", external, "old/ambiguous.xml", "old/missing.xml", "../../includes/v1.0.0/archived.xml", "https://example.invalid/unique.xml"} {
					if !locations[location] {
						t.Errorf("gateway %q missing location %q: %v", name, location, locations)
					}
				}
				for _, value := range []string{"Title | Group | ../../includes/unique.xml", external, "7039", "old/ambiguous.xml"} {
					if !bytes.Contains(updatedJSON, []byte(value)) {
						t.Errorf("gateway %q JSON missing %q: %s", name, value, updatedJSON)
					}
				}
				if !json.Valid(updatedJSON) || !bytes.Contains(xmlData, []byte("<customSection>")) {
					t.Fatal("repair lost valid JSON or unrelated XML")
				}
				var stored struct {
					Unrelated map[string]string
				}
				if err := json.Unmarshal(updatedJSON, &stored); err != nil {
					t.Fatal(err)
				}
				if stored.Unrelated["location"] != "old/unique.xml" || stored.Unrelated["label"] != "Title | Group | old/unique.xml" {
					t.Fatalf("repair changed settings outside includes: %v", stored.Unrelated)
				}
				if info, err := os.Stat(jsonPath); err != nil || info.Mode().Perm() != 0640 {
					t.Fatalf("JSON permissions changed: %v, %v", info, err)
				}
				if original, err := os.ReadFile(setupPath + "_0001.history"); err != nil || string(original) != setup {
					t.Fatalf("XML backup does not match original: %v", err)
				}
				if backups, err := filepath.Glob(jsonPath + ".*.bak"); err != nil || len(backups) != 0 {
					t.Fatalf("unexpected JSON backups = %v, %v", backups, err)
				}
			}
			if !test.dryRun {
				if err := Fix(h, false, test.pattern); err != nil {
					t.Fatalf("second repair failed: %v", err)
				}
				if backups, err := filepath.Glob(h.PathTo("gateway", "gateways", "*", "gateway.setup.xml_0002.history")); err != nil || len(backups) != 0 {
					t.Fatalf("second repair rewrote unchanged setups: %v, %v", backups, err)
				}
			}
		})
	}
}

func TestFixInvalidInstanceConfigLeavesSetupUnchanged(t *testing.T) {
	h := newTestGatewayHost(t)
	filename := h.PathTo("gateway", "includes", "unique.xml")
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte("<gateway/>"), 0644); err != nil {
		t.Fatal(err)
	}
	setup := `<gateway><includes><include><location>missing/unique.xml</location></include></includes></gateway>`
	writeTestGatewaySetup(t, h, setup)
	home := h.PathTo("gateway", "gateways", "test")
	for _, document := range []string{"{", "{} trailing"} {
		if err := os.WriteFile(filepath.Join(home, "gateway.json"), []byte(document), 0644); err != nil {
			t.Fatal(err)
		}
		for _, dryRun := range []bool{true, false} {
			if err := Fix(h, dryRun); err == nil || !strings.Contains(err.Error(), "instance configuration") {
				t.Fatalf("Fix() error = %v; want malformed instance config error", err)
			}
			if data, err := os.ReadFile(filepath.Join(home, "gateway.setup.xml")); err != nil || string(data) != setup {
				t.Fatalf("malformed JSON allowed XML write: %v", err)
			}
		}
	}
}

func TestFixYAMLInstanceConfiguration(t *testing.T) {
	previousFormat := config.Get[string](config.Global(), "configtype")
	wasSet := config.Global().IsSet("configtype")
	config.Set(config.Global(), "configtype", "yaml")
	t.Cleanup(func() {
		if wasSet {
			config.Set(config.Global(), "configtype", previousFormat)
		} else {
			config.Delete(config.Global(), "configtype")
		}
	})
	h := newTestGatewayHost(t)
	include := h.PathTo("gateway", "includes", "unique.xml")
	if err := os.MkdirAll(filepath.Dir(include), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(include, []byte("<gateway/>"), 0644); err != nil {
		t.Fatal(err)
	}
	setup := `<gateway><includes><include><location>old/unique.xml</location></include></includes></gateway>`
	writeTestGatewaySetup(t, h, setup)
	filename := h.PathTo("gateway", "gateways", "test", "gateway.yaml")
	original := []byte("setup: gateway.setup.xml\nincludes:\n  '1000': 'Title | Group | old/unique.xml'\nport: 7039\n")
	if err := os.WriteFile(filename, original, 0640); err != nil {
		t.Fatal(err)
	}
	if err := Fix(h, true, "test"); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filename); err != nil || !bytes.Equal(data, original) {
		t.Fatalf("dry-run changed YAML instance config: %v", err)
	}
	gateways := instance.Instances(h, geneos.ParseComponent("gateway"))
	if len(gateways) != 1 || config.Get[string](gateways[0].Config(), "includes.1000") != "Title | Group | old/unique.xml" {
		t.Fatal("dry-run changed loaded config values")
	}
	if err := Fix(h, false, "test"); err != nil {
		t.Fatal(err)
	}
	updated, err := config.Read("gateway", config.FilePath(filename), config.Format("yaml"), config.UseDefaults(false), config.MustExist())
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Get[string](updated, "includes.1000"); got != "Title | Group | ../../includes/unique.xml" {
		t.Fatalf("YAML repair = %q", got)
	}
	if config.Get[int](updated, "port") != 7039 {
		t.Fatal("repair changed unrelated YAML setting")
	}
	if _, err := os.Stat(strings.TrimSuffix(filename, ".yaml") + ".json"); !os.IsNotExist(err) {
		t.Fatalf("YAML repair created JSON: %v", err)
	}
	if backups, err := filepath.Glob(filename + ".*.bak"); err != nil || len(backups) != 0 {
		t.Fatalf("unexpected YAML backups = %v, %v", backups, err)
	}
}

func TestFixNoChangesOutput(t *testing.T) {
	for _, test := range []struct {
		name   string
		setup  string
		remote bool
	}{
		{name: "no-includes", setup: `<gateway/>`},
		{name: "empty-includes", setup: `<gateway><includes/></gateway>`},
		{name: "valid", setup: `<gateway><includes><include><location>../../includes/valid.xml</location></include></includes></gateway>`},
		{name: "unresolved", setup: `<gateway><includes><include><location>missing.xml</location></include></includes></gateway>`},
		{name: "http-include", setup: `<gateway><includes><include><location>https://example.invalid/include.xml</location></include></includes></gateway>`},
		{name: "http-setup", setup: `<gateway/>`, remote: true},
	} {
		for _, dryRun := range []bool{false, true} {
			t.Run(test.name+"/dry-run="+strconv.FormatBool(dryRun), func(t *testing.T) {
				var (
					h              = newTestGatewayHost(t)
					include        = h.PathTo("gateway", "includes", "valid.xml")
					originalConfig = []byte(`{"port":7039}`)
					previousStdout = os.Stdout
					output         *os.File
					data           []byte
					err            error
				)

				if err := os.MkdirAll(filepath.Dir(include), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(include, []byte("<gateway/>"), 0644); err != nil {
					t.Fatal(err)
				}
				writeTestGatewaySetup(t, h, test.setup, "matching", "other")
				if test.remote {
					originalConfig = []byte(`{"setup":"https://example.invalid/setup.xml"}`)
				}
				for _, name := range []string{"matching", "other"} {
					if err := os.WriteFile(h.PathTo("gateway", "gateways", name, "gateway.json"), originalConfig, 0644); err != nil {
						t.Fatal(err)
					}
				}
				output, err = os.CreateTemp(t.TempDir(), "fix-output-*")
				if err != nil {
					t.Fatal(err)
				}
				os.Stdout = output
				t.Cleanup(func() {
					os.Stdout = previousStdout
					output.Close()
				})
				if err := Fix(h, dryRun, "matching"); err != nil {
					t.Fatal(err)
				}
				data, err = os.ReadFile(output.Name())
				if err != nil {
					t.Fatal(err)
				}
				if strings.Count(string(data), "Gateway matching: no changes made") != 1 {
					t.Fatalf("expected one no-change message for matching gateway, got %q", data)
				}
				if bytes.Contains(data, []byte("Gateway other:")) {
					t.Fatalf("reported an unselected gateway: %s", data)
				}
				if original, err := os.ReadFile(h.PathTo("gateway", "gateways", "matching", "gateway.setup.xml")); err != nil || string(original) != test.setup {
					t.Fatalf("no-change run modified setup: %v", err)
				}
				if original, err := os.ReadFile(h.PathTo("gateway", "gateways", "matching", "gateway.json")); err != nil || !bytes.Equal(original, originalConfig) {
					t.Fatalf("no-change run modified config: %v", err)
				}
			})
		}
	}
}
