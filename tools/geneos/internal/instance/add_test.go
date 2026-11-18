package instance_test

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// newTestRoot sets up a temporary Geneos directory, with a licd
// package, as the local installation for the test
func newTestRoot(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	key := cordial.ExecutableName()
	previousRoot := config.Get[string](config.Global(), key)
	previousLocal, previousAll, previousHostname := geneos.LOCAL, geneos.ALL, geneos.LOCALHOST
	previousHandler := logger.LogHandler
	t.Cleanup(func() {
		config.Set(config.Global(), key, previousRoot)
		geneos.LOCAL, geneos.ALL, geneos.LOCALHOST = previousLocal, previousAll, previousHostname
		logger.LogHandler = previousHandler
	})
	logger.LogHandler = slog.NewTextHandler(io.Discard, nil)
	config.Set(config.Global(), key, root)
	geneos.LOCALHOST = "localhost"
	geneos.LOCAL = nil
	geneos.LOCAL = geneos.NewHost(geneos.LOCALHOST)
	geneos.ALL = geneos.NewHost(geneos.ALLHOSTS)
	if err := os.MkdirAll(filepath.Join(root, "packages", "licd", "active_prod"), 0o755); err != nil {
		t.Fatal(err)
	}
	return
}

func TestAddExistingInstanceReturnsErrExists(t *testing.T) {
	newTestRoot(t)

	first, err := instance.Add(geneos.LOCAL, &licd.Component, "exists-test", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Unload() })

	again, err := instance.Add(geneos.LOCAL, &licd.Component, "exists-test", instance.Insecure(true))
	if !errors.Is(err, geneos.ErrExists) {
		t.Fatalf("Add() of an existing instance error = %v, want ErrExists", err)
	}
	if again == nil || again.Name() != "exists-test" {
		t.Errorf("Add() of an existing instance returned %v, want the existing instance", again)
	}
}

func TestUnknownAndMissingHosts(t *testing.T) {
	newTestRoot(t)

	for name, h := range map[string]*geneos.Host{"unknown": geneos.UNKNOWN, "named unknown": geneos.GetHost("no-such-host.example.invalid")} {
		if _, err := instance.Add(h, &licd.Component, "x", instance.Insecure(true)); !errors.Is(err, geneos.ErrNotExist) {
			t.Errorf("Add(%s host) error = %v, want ErrNotExist", name, err)
		}
	}
	if _, err := instance.Add(nil, &licd.Component, "x", instance.Insecure(true)); !errors.Is(err, geneos.ErrInvalidArgs) {
		t.Errorf("Add(nil host) error = %v, want ErrInvalidArgs", err)
	}
	if _, err := instance.Add(geneos.LOCAL, &licd.Component, "x@no-such-host.example.invalid", instance.Insecure(true)); !errors.Is(err, geneos.ErrNotExist) {
		t.Errorf("Add(name@unknown) error = %v, want ErrNotExist", err)
	}

	if _, err := instance.Get(&licd.Component, "x@no-such-host.example.invalid"); !errors.Is(err, geneos.ErrNotExist) {
		t.Errorf("Get(name@unknown) error = %v, want ErrNotExist", err)
	}
	if _, err := instance.Get(&licd.Component, "x@"); !errors.Is(err, geneos.ErrInvalidArgs) {
		t.Errorf("Get(name@) error = %v, want ErrInvalidArgs", err)
	}
	if _, err := instance.GetWithHost(geneos.UNKNOWN, &licd.Component, "x"); !errors.Is(err, geneos.ErrNotExist) {
		t.Errorf("GetWithHost(unknown) error = %v, want ErrNotExist", err)
	}
	if _, err := instance.GetWithHost(nil, &licd.Component, "x"); !errors.Is(err, geneos.ErrInvalidArgs) {
		t.Errorf("GetWithHost(nil) error = %v, want ErrInvalidArgs", err)
	}

	// an unknown host is never all hosts
	if got := instance.Instances(geneos.UNKNOWN, nil); len(got) != 0 {
		t.Errorf("Instances(unknown host) = %v, want none", got)
	}
	if got := instance.AllInstanceNames(geneos.UNKNOWN, nil); len(got) != 0 {
		t.Errorf("AllInstanceNames(unknown host) = %v, want none", got)
	}
	if got := instance.GetAllPorts(geneos.UNKNOWN); len(got) != 0 {
		t.Errorf("GetAllPorts(unknown host) = %v, want none", got)
	}
}

func TestAddFailureKeepsExistingDirectory(t *testing.T) {
	root := newTestRoot(t)
	instances := filepath.Join(root, "licd", "licds")

	// a bad certificate bundle makes Add fail after the configuration
	// has been written
	fail := []instance.AddOption{instance.Insecure(true), instance.WithCertBundle(filepath.Join(root, "missing-bundle.pem"))}

	t.Run("new directory is removed", func(t *testing.T) {
		if _, err := instance.Add(geneos.LOCAL, &licd.Component, "fails-new", fail...); err == nil {
			t.Fatal("Add() with a missing certificate bundle succeeded")
		}
		if _, err := os.Stat(filepath.Join(instances, "fails-new")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("partially created instance directory was not removed: %v", err)
		}
	})

	t.Run("existing directory is kept", func(t *testing.T) {
		dir := filepath.Join(instances, "fails-existing")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		precious := filepath.Join(dir, "precious.txt")
		if err := os.WriteFile(precious, []byte("keep me"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := instance.Add(geneos.LOCAL, &licd.Component, "fails-existing", fail...); err == nil {
			t.Fatal("Add() with a missing certificate bundle succeeded")
		}
		if data, err := os.ReadFile(precious); err != nil || string(data) != "keep me" {
			t.Errorf("a file in a directory that existed before was removed: %q, %v", data, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "licd.json")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the configuration of the failed instance was left behind: %v", err)
		}
	})
}

func TestInstancesAreOrdered(t *testing.T) {
	newTestRoot(t)

	for _, name := range []string{"order-c", "order-a", "order-b"} {
		i, err := instance.Add(geneos.LOCAL, &licd.Component, name, instance.Insecure(true))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = i.Unload() })
	}

	want := []string{"order-a", "order-b", "order-c"}
	for range 5 {
		var got []string
		for _, i := range instance.Instances(geneos.LOCAL, nil) {
			got = append(got, i.Name())
		}
		if !slices.Equal(got, want) {
			t.Fatalf("Instances() = %v, want %v", got, want)
		}
	}

	// a parent package match with no component has nothing to add
	if got := instance.Instances(geneos.LOCAL, nil, instance.MatchParentPackage()); len(got) != len(want) {
		t.Errorf("Instances(MatchParentPackage, nil component) = %d instances, want %d", len(got), len(want))
	}
}
