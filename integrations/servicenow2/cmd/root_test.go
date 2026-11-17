package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestLoadConfigFile(t *testing.T) {
	origFile, origName := configFile, Execname
	defer func() { configFile, Execname = origFile, origName }()
	Execname = "servicenow2"

	dir := t.TempDir()

	t.Run("missing file is an error", func(t *testing.T) {
		configFile = filepath.Join(dir, "does-not-exist.yaml")
		cf, err := LoadConfigFile("client")
		if err == nil || cf != nil {
			t.Errorf("LoadConfigFile() = %v, %v, want an error and no config", cf, err)
		}
	})

	t.Run("invalid file is an error", func(t *testing.T) {
		configFile = filepath.Join(dir, "invalid.yaml")
		if err := os.WriteFile(configFile, []byte("proxy: [unterminated"), 0o600); err != nil {
			t.Fatal(err)
		}
		cf, err := LoadConfigFile("client")
		if err == nil || cf != nil {
			t.Errorf("LoadConfigFile() = %v, %v, want an error and no config", cf, err)
		}
	})

	t.Run("valid file", func(t *testing.T) {
		configFile = filepath.Join(dir, "valid.yaml")
		if err := os.WriteFile(configFile, []byte("proxy:\n  url: http://localhost:3000/x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cf, err := LoadConfigFile("client")
		if err != nil {
			t.Fatal(err)
		}
		if got := config.Get[string](cf, "proxy.url"); got != "http://localhost:3000/x" {
			t.Errorf("proxy.url = %q", got)
		}
	})
}
