package profile

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateProfileListFormat(t *testing.T) {
	for _, format := range []string{"column", "csv", "json", "pretty", "table", "toolkit"} {
		if err := validateProfileListFormat(format); err != nil {
			t.Errorf("validateProfileListFormat(%q) error = %v", format, err)
		}
	}
	if err := validateProfileListFormat("xml"); err == nil {
		t.Fatal("validateProfileListFormat accepted unsupported xml format")
	}
}

func TestFormatProfileListCSV(t *testing.T) {
	profiles := map[string]any{
		"demo": map[string]any{"_short": "Demonstration profile"},
	}
	var output bytes.Buffer
	if err := formatProfileList(&output, profiles, "/etc/geneos/profiles.yaml", "csv"); err != nil {
		t.Fatalf("formatProfileList() error = %v", err)
	}
	for _, want := range []string{"Profile", "Description", "From", "demo", "Demonstration profile", "/etc/geneos/profiles.yaml"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("formatProfileList() output %q does not contain %q", output.String(), want)
		}
	}
}

func TestFormatProfileListJSON(t *testing.T) {
	profiles := map[string]any{
		"demo": map[string]any{"_short": "Demonstration profile"},
	}
	var output bytes.Buffer
	if err := formatProfileList(&output, profiles, "/etc/geneos/profiles.yaml", "json"); err != nil {
		t.Fatalf("formatProfileList() error = %v", err)
	}
	for _, want := range []string{`"profile":"demo"`, `"description":"Demonstration profile"`, `"from":"/etc/geneos/profiles.yaml"`} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("formatProfileList() output %q does not contain %q", output.String(), want)
		}
	}
}

func TestExtractProfileConfigUnchangedWhenIdentical(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "profiles.yaml")
	defaults := []byte("profiles:\n  demo: {}\n")
	if err := os.WriteFile(filename, defaults, 0o644); err != nil {
		t.Fatal(err)
	}

	backup, changed, err := extractProfileConfig(filename, defaults, time.Now())
	if err != nil {
		t.Fatalf("extractProfileConfig() error = %v", err)
	}
	if changed || backup != "" {
		t.Fatalf("extractProfileConfig() = (%q, %t), want unchanged", backup, changed)
	}
}

func TestExtractProfileConfigBacksUpDifferentFile(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "profiles.yaml")
	current := []byte("profiles:\n  custom: {}\n")
	defaults := []byte("profiles:\n  demo: {}\n")
	if err := os.WriteFile(filename, current, 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.November, 3, 12, 34, 56, 0, time.UTC)

	backup, changed, err := extractProfileConfig(filename, defaults, now)
	if err != nil {
		t.Fatalf("extractProfileConfig() error = %v", err)
	}
	wantBackup := filepath.Join(dir, "profiles-20261103123456.yaml")
	if !changed || backup != wantBackup {
		t.Fatalf("extractProfileConfig() = (%q, %t), want (%q, true)", backup, changed, wantBackup)
	}
	if contents, err := os.ReadFile(filename); err != nil || !bytes.Equal(contents, defaults) {
		t.Fatalf("profiles file = %q, err = %v; want embedded defaults", contents, err)
	}
	if contents, err := os.ReadFile(backup); err != nil || !bytes.Equal(contents, current) {
		t.Fatalf("backup file = %q, err = %v; want previous contents", contents, err)
	}
}

func TestExtractProfileConfigCreatesMissingFile(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "geneos", "profiles.yaml")
	defaults := []byte("profiles:\n  demo: {}\n")

	backup, changed, err := extractProfileConfig(filename, defaults, time.Now())
	if err != nil {
		t.Fatalf("extractProfileConfig() error = %v", err)
	}
	if !changed || backup != "" {
		t.Fatalf("extractProfileConfig() = (%q, %t), want created without backup", backup, changed)
	}
	if contents, err := os.ReadFile(filename); err != nil || !bytes.Equal(contents, defaults) {
		t.Fatalf("profiles file = %q, err = %v; want embedded defaults", contents, err)
	}
}

func TestExtractProfileConfigAvoidsBackupCollision(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "profiles.yaml")
	firstBackup := filepath.Join(dir, "profiles-20261103123456.yaml")
	if err := os.WriteFile(filename, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(firstBackup, []byte("older backup"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.November, 3, 12, 34, 56, 0, time.UTC)

	backup, changed, err := extractProfileConfig(filename, []byte("new"), now)
	if err != nil {
		t.Fatalf("extractProfileConfig() error = %v", err)
	}
	wantBackup := filepath.Join(dir, "profiles-20261103123456-1.yaml")
	if !changed || backup != wantBackup {
		t.Fatalf("extractProfileConfig() = (%q, %t), want (%q, true)", backup, changed, wantBackup)
	}
	if contents, err := os.ReadFile(firstBackup); err != nil || string(contents) != "older backup" {
		t.Fatalf("existing backup = %q, err = %v; want it unchanged", contents, err)
	}
}
