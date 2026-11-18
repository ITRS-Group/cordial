package instance_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func TestMigrateLeavesCustomHome(t *testing.T) {
	root := newTestRoot(t)

	i, err := instance.Add(geneos.LOCAL, &licd.Component, "custom-home-test", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Unload() })

	// a configured home that is not one of the standard locations
	custom := filepath.Join(root, "custom", "custom-home-test")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	config.Set(i.Config(), "home", custom)
	if got := i.Home(); got != custom {
		t.Fatalf("Home() = %q, want the configured directory", got)
	}

	if resp := instance.Migrate(i); resp.Err != nil {
		t.Fatalf("Migrate() error = %v", resp.Err)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Errorf("the custom home directory was moved: %v", err)
	}
}

func TestMigrateMovesLegacyDirectory(t *testing.T) {
	root := newTestRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "packages", "netprobe", "active_prod"), 0o755); err != nil {
		t.Fatal(err)
	}

	// the legacy layout has the instances of fa2 under `fa2/fa2s`
	legacy := filepath.Join(root, "fa2", "fa2s", "legacy-test")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "fa2.json"), []byte(`{"name":"legacy-test"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	i, err := instance.Get(&fa2.Component, "legacy-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Unload() })

	if resp := instance.Migrate(i); resp.Err != nil {
		t.Fatalf("Migrate() error = %v", resp.Err)
	}
	moved := filepath.Join(root, "netprobe", "fa2s", "legacy-test")
	if _, err := os.Stat(filepath.Join(moved, "fa2.json")); err != nil {
		t.Errorf("the instance was not moved to the standard location: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("the legacy directory still exists: %v", err)
	}
}

// TestRefactorConfigRenamedInstance checks that paths in the old geneos
// directory are updated when an instance is also renamed
func TestRefactorConfigRenamedInstance(t *testing.T) {
	const newRoot = "/new-root"
	cf := config.New(config.KeyDelimiter("::"))
	config.Set(cf, cordial.ExecutableName(), newRoot)
	h := &geneos.Host{Host: host.NewLocal(), Config: cf}

	ct := &geneos.Component{Name: "gateway"}
	icf := config.New()
	config.Set(icf, "name", "old")
	config.Set(icf, "home", "/old-root/gateway/gateways/old")
	config.Set(icf, "install", "/old-root/packages/gateway")
	config.Set(icf, "logfile", "/old-root/gateway/gateways/old/gateway.log")
	config.Set(icf, "certchain", "/old-root/tls/ca-bundle.pem")
	config.Set(icf, "unrelated", "/somewhere/else")
	config.Set(icf, "empty", "")

	err := instance.RefactorConfig(h, ct, icf,
		instance.NewName("new"),
		instance.NewDir(newRoot+"/gateway/gateways/new"),
	)
	if err != nil {
		t.Fatal(err)
	}

	for key, want := range map[string]string{
		"name":      "new",
		"logfile":   "${config:home}/gateway.log",
		"certchain": newRoot + "/tls/ca-bundle.pem",
		"install":   newRoot + "/packages/gateway",
		"unrelated": "/somewhere/else",
	} {
		if got := config.Get[string](icf, key, config.NoExpand()); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if got := config.Get[string](icf, "empty"); got != "" {
		t.Errorf("an empty value was changed to %q", got)
	}
}
