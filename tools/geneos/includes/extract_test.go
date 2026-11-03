package includes

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var testGatewayComponentOnce sync.Once

func newTestGatewayHost(t *testing.T) *geneos.Host {
	t.Helper()
	testGatewayComponentOnce.Do(func() {
		(&geneos.Component{Name: "gateway"}).Register(nil)
	})

	h := &geneos.Host{Host: host.NewLocal(), Config: config.New()}
	config.Set(h.Config, cordial.ExecutableName(), t.TempDir())
	return h
}

func TestExtractRejectsExistingVersionDirectoryWithoutForce(t *testing.T) {
	h := newTestGatewayHost(t)
	versionDir := h.PathTo("gateway", "includes", cordial.VERSION)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := Extract(h, false); err == nil {
		t.Fatal("Extract() succeeded with an existing version directory and no force")
	}
}

func TestExtractCreatesVersionedAndConfigFiles(t *testing.T) {
	h := newTestGatewayHost(t)
	if err := Extract(h, false); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	versionDir := h.PathTo("gateway", "includes", cordial.VERSION)
	versionFiles, err := os.ReadDir(versionDir)
	if err != nil {
		t.Fatalf("ReadDir(versioned includes) error = %v", err)
	}
	if len(versionFiles) == 0 {
		t.Fatal("Extract() created no versioned include files")
	}
	for _, entry := range versionFiles {
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("Info(%q) error = %v", entry.Name(), err)
		}
		if got := info.Mode().Perm(); got != 0444 {
			t.Errorf("versioned file %q permissions = %04o, want 0444", entry.Name(), got)
		}
	}

	configDir := filepath.Join(h.PathTo("gateway"), "includes", "config")
	configFiles, err := os.ReadDir(configDir)
	if err != nil {
		t.Fatalf("ReadDir(config includes) error = %v", err)
	}
	if len(configFiles) == 0 {
		t.Fatal("Extract() created no config template files")
	}
}
