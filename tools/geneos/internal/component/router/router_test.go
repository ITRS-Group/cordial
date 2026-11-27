package router

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func TestCommandUsesReporterPortDefaultAndOverride(t *testing.T) {
	root := geneostest.NewRoot(t, "", name)
	for _, source := range initialFiles {
		filename := filepath.Join(root, "packages", name, "active_prod", filepath.FromSlash(source))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte("from package"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	i, err := instance.Add(geneos.LOCAL, &Component, "router-ports", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	env := func() []string {
		t.Helper()
		_, env, _, err := i.Command(true)
		if err != nil {
			t.Fatal(err)
		}
		return env
	}

	if !slices.Contains(env(), "COLLECTOR_PORT=4317") {
		t.Errorf("environment does not use the collector default: %v", env())
	}
	if !slices.Contains(env(), "REPORTER_PORT=4318") {
		t.Errorf("environment does not use the reporter default: %v", env())
	}

	config.Set(i.Config(), "reporter-port", uint16(9320))
	if !slices.Contains(env(), "REPORTER_PORT=9320") {
		t.Errorf("environment does not use the reporter override: %v", env())
	}
}
