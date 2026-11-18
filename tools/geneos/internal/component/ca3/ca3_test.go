package ca3

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

func newCA3Root(t *testing.T) {
	t.Helper()
	root := geneostest.NewRoot(t, "", "netprobe")
	for _, name := range initialFiles {
		filename := filepath.Join(root, "packages", "netprobe", "active_prod", "collection_agent", name)
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte("from package "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAddCopiesPackageFilesWithoutChangingDirectory(t *testing.T) {
	newCA3Root(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	i, err := instance.Add(geneos.LOCAL, &Component, "ca3-files", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	if now, _ := os.Getwd(); now != cwd {
		t.Errorf("working directory changed from %q to %q", cwd, now)
	}
	for _, name := range initialFiles {
		got, err := os.ReadFile(filepath.Join(i.Home(), name))
		if err != nil || string(got) != "from package "+name {
			t.Errorf("%s = %q, %v", name, got, err)
		}
	}
}

func TestAddWithoutPackageFilesIsAnError(t *testing.T) {
	geneostest.NewRoot(t, "", "netprobe")
	if _, err := instance.Add(geneos.LOCAL, &Component, "ca3-missing", instance.Insecure(true)); err == nil {
		t.Fatal("Add() succeeded without the package files")
	}
}

func TestCommandUsesConfiguredPort(t *testing.T) {
	newCA3Root(t)
	i, err := instance.Add(geneos.LOCAL, &Component, "ca3-ports", instance.Insecure(true), instance.WithPort(9250))
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

	if !slices.Contains(env(), "TCP_REPORTER_PORT=9250") {
		t.Errorf("environment does not use the instance port: %v", env())
	}
	if !slices.Contains(env(), "HEALTH_CHECK_PORT=9136") {
		t.Errorf("environment does not use the default health check port: %v", env())
	}

	config.Set(i.Config(), "tcp-reporter-port", uint16(9300))
	config.Set(i.Config(), "hostname", "set-host")
	if !slices.Contains(env(), "TCP_REPORTER_PORT=9300") || !slices.Contains(env(), "HOSTNAME=set-host") {
		t.Errorf("overrides not used: %v", env())
	}
}
