package includes

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func captureIncludeAudit(t *testing.T) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	commands, actions := geneos.AuditCommands(), geneos.AuditActions()
	geneos.SetAuditActions(true)
	logger.NewGlobalAuditLogger(&output)
	t.Cleanup(func() {
		instance.CloseAuditLoggers()
		logger.NewGlobalAuditLogger(io.Discard)
		geneos.SetAuditCommands(commands)
		geneos.SetAuditActions(actions)
	})
	return &output
}

func includeAuditRows(t *testing.T, output *bytes.Buffer) (rows []map[string]any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	for {
		var row map[string]any
		if err := decoder.Decode(&row); errors.Is(err, io.EOF) {
			return rows
		} else if err != nil {
			t.Fatalf("invalid audit JSON: %v\n%s", err, output)
		}
		rows = append(rows, row)
	}
}

func TestIncludesCommandAuditing(t *testing.T) {
	h := newTestGatewayHost(t)
	writeTestGatewaySetup(t, h, "<gateway/>")
	previousHost := cmd.Hostname
	cmd.Hostname = "localhost"
	t.Cleanup(func() { cmd.Hostname = previousHost })
	for _, key := range []string{"audit.commands", "audit.actions"} {
		value, exists := config.Lookup[any](config.Global(), key)
		config.Delete(config.Global(), key)
		t.Cleanup(func() {
			if exists {
				config.Set(config.Global(), key, value)
			} else {
				config.Delete(config.Global(), key)
			}
		})
	}
	for _, name := range []string{"install", "extract", "update", "delete", "remove", "fix", "list"} {
		t.Run(name, func(t *testing.T) {
			output := captureIncludeAudit(t)
			command, _, err := includesCmd.Find([]string{name})
			if err != nil {
				t.Fatal(err)
			}
			previous := command.Context()
			command.SetContext(context.WithValue(context.Background(), cmd.CmdKey, &cmd.CmdValType{}))
			t.Cleanup(func() { command.SetContext(previous) })
			geneos.SetAuditCommands(false)
			geneos.SetAuditActions(false)
			if err := cmd.ParseArgs(command, nil); err != nil {
				t.Fatal(err)
			}
			rows := includeAuditRows(t, output)
			if name == "list" {
				if len(rows) != 0 || geneos.AuditActions() {
					t.Fatalf("list enabled auditing: %s", output)
				}
				return
			}
			if len(rows) != 1 || rows[0]["command"] != command.CommandPath() || !geneos.AuditActions() {
				t.Fatalf("mutating command was not audited: %s", output)
			}
		})
	}
}

func TestIncludesFileAuditing(t *testing.T) {
	output := captureIncludeAudit(t)
	h := newTestGatewayHost(t)
	legacy := h.PathTo("gateway", "includes", "config", "config.test.xml")
	if err := os.MkdirAll(filepath.Dir(legacy), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}
	writeTestGatewaySetup(t, h, `<gateway><includes><include><location>../../includes/config/config.test.xml</location></include></includes></gateway>`)
	archive := func(version, content string) string {
		root := "geneos-includes-" + version
		return writeTestIncludeArchive(t,
			includeArchiveEntry{name: root + "/", kind: tar.TypeDir},
			includeArchiveEntry{name: root + "/itrs-" + version + "/itrs.test.xml", contents: content},
			includeArchiveEntry{name: root + "/config/config.test.xml", contents: "template"},
			includeArchiveEntry{name: root + "/config/config.new.xml", contents: "new"},
		)
	}
	first := archive("v1.2.3", "first")
	if err := Install(h, first, true, false); err != nil {
		t.Fatal(err)
	}
	rows := includeAuditRows(t, output)
	output.Reset()
	if err := Install(h, first, true, false); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("unchanged files were audited as mutations: %s", output)
	}
	if err := Install(h, archive("v1.2.3", "changed"), true, false); err != nil {
		t.Fatal(err)
	}
	if err := Install(h, archive("v2.0.0", "new version"), true, true); err != nil {
		t.Fatal(err)
	}
	old := h.PathTo("gateway", "includes", "itrs-v1.2.3")
	if err := Delete(h, "itrs-v1.2.3"); err != nil {
		t.Fatal(err)
	}
	rows = append(rows, includeAuditRows(t, output)...)
	found := make(map[string]bool)
	for _, row := range rows {
		if _, exists := row["instance"]; exists {
			t.Errorf("shared include action has instance details: %v", row)
		}
		if row["host"] != h.String() || row["result"] != "success" {
			t.Errorf("missing host or successful result: %v", row)
		}
		action, _ := row["action"].(string)
		file, _ := row["file"].(string)
		found[action+" "+file] = true
		if action == "includes/backup" {
			if row["source"] != filepath.Join(old, "itrs.test.xml") {
				t.Errorf("backup lacks source: %v", row)
			}
			found["backup"] = true
		}
	}
	for _, want := range []string{
		"includes/extract " + filepath.Join(old, "itrs.test.xml"),
		"includes/copy " + h.PathTo("gateway", geneos.SiteName(), geneos.SiteName()+"-config.test.xml"),
		"includes/extract " + h.PathTo("gateway", geneos.SiteName(), geneos.SiteName()+"-config.new.xml"),
		"includes/remove " + h.PathTo("gateway", "includes", "itrs.test.xml"),
		"includes/symlink " + h.PathTo("gateway", "includes", "itrs.test.xml"),
		"includes/delete " + filepath.Join(old, "itrs.test.xml"),
		"includes/delete " + old,
		"backup",
	} {
		if !found[want] {
			t.Errorf("missing audit action %q", want)
		}
	}
	output.Reset()
	geneos.SetAuditActions(false)
	if err := Install(h, archive("v3.0.0", "unaudited"), true, false); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled auditing emitted actions: %s", output)
	}
}

type includeAuditRemoveHost struct {
	host.Host
	fail string
}

func (h *includeAuditRemoveHost) Remove(name string) error {
	if name == h.fail {
		return errors.New("test removal failed")
	}
	return h.Host.Remove(name)
}

func TestIncludesDeletionAuditsPartialFailure(t *testing.T) {
	output := captureIncludeAudit(t)
	h := newTestGatewayHost(t)
	directory := h.PathTo("gateway", "includes", "itrs-v1.2.3")
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.xml", "b.xml"} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	h.Host = &includeAuditRemoveHost{Host: h.Host, fail: filepath.Join(directory, "a.xml")}
	if err := Delete(h, "itrs-v1.2.3"); err == nil {
		t.Fatal("deletion succeeded despite a file removal failure")
	}
	rows := includeAuditRows(t, output)
	if len(rows) != 2 || rows[0]["file"] != filepath.Join(directory, "b.xml") || rows[0]["result"] != "success" ||
		rows[1]["file"] != filepath.Join(directory, "a.xml") || rows[1]["result"] != "test removal failed" {
		t.Fatalf("partial deletion results not recorded accurately: %s", output)
	}
}

func TestIncludesFixAuditsLocationChanges(t *testing.T) {
	for _, format := range []string{"json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			previous, exists := config.Lookup[any](config.Global(), "configtype")
			config.Set(config.Global(), "configtype", format)
			t.Cleanup(func() {
				if exists {
					config.Set(config.Global(), "configtype", previous)
				} else {
					config.Delete(config.Global(), "configtype")
				}
			})
			output := captureIncludeAudit(t)
			h := newTestGatewayHost(t)
			path := h.PathTo("gateway", "includes", "unique.xml")
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("<gateway/>"), 0644); err != nil {
				t.Fatal(err)
			}
			writeTestGatewaySetup(t, h, `<gateway><includes><include><location>old/unique.xml</location></include></includes></gateway>`, "prod-a")
			configPath := h.PathTo("gateway", "gateways", "prod-a", "gateway."+format)
			content := `{"includes":{"500":"Title | old/unique.xml"}}`
			if format == "yaml" {
				content = "includes:\n  500: Title | old/unique.xml\n"
			}
			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			if err := Fix(h, true, "prod-a"); err != nil {
				t.Fatal(err)
			}
			if output.Len() != 0 {
				t.Fatalf("dry run emitted filesystem audit actions: %s", output)
			}
			if err := Fix(h, false, "prod-a"); err != nil {
				t.Fatal(err)
			}
			rows := includeAuditRows(t, output)
			if len(rows) != 2 {
				t.Fatalf("want setup and configuration repair events: %s", output)
			}
			for _, row := range rows {
				details, ok := row["instance"].(map[string]any)
				if !ok || details["component"] != "gateway" || details["name"] != "prod-a" || details["host"] != h.String() {
					t.Errorf("repair lacks Gateway context: %v", row)
				}
				description, _ := row["description"].(string)
				if row["action"] != "includes/fix" || row["result"] != "success" || row["oldLocation"] != "old/unique.xml" ||
					row["newLocation"] != "../../includes/unique.xml" || !strings.Contains(description, `"old/unique.xml" -> "../../includes/unique.xml"`) {
					t.Errorf("repair lacks location change details: %v", row)
				}
			}
			setupPath := h.PathTo("gateway", "gateways", "prod-a", "gateway.setup.xml")
			if rows[0]["backup"] != setupPath+"_0001.history" || rows[1]["file"] != configPath {
				t.Errorf("repair lacks setup backup or configuration path: %s", output)
			}
		})
	}
}
