package licd

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func TestAddAllocatesPorts(t *testing.T) {
	root := t.TempDir()
	key := cordial.ExecutableName()
	previousRoot := config.Get[string](config.Global(), key)
	previousRange := config.Get[string](config.Global(), Component.PortRange)
	previousLocal, previousAll, previousHostname := geneos.LOCAL, geneos.ALL, geneos.LOCALHOST
	previousHandler := logger.LogHandler
	t.Cleanup(func() {
		config.Set(config.Global(), key, previousRoot)
		config.Set(config.Global(), Component.PortRange, previousRange)
		geneos.LOCAL, geneos.ALL, geneos.LOCALHOST = previousLocal, previousAll, previousHostname
		logger.LogHandler = previousHandler
	})
	logger.LogHandler = slog.NewTextHandler(io.Discard, nil)
	config.Set(config.Global(), key, root)
	config.Set(config.Global(), Component.PortRange, "7041,7100-")
	geneos.LOCALHOST = "localhost"
	geneos.LOCAL = nil
	geneos.LOCAL = geneos.NewHost(geneos.LOCALHOST)
	geneos.ALL = geneos.NewHost(geneos.ALLHOSTS)
	if err := os.MkdirAll(filepath.Join(root, "packages", name, "active_prod"), 0o755); err != nil {
		t.Fatal(err)
	}

	add := func(t *testing.T, name string, want uint16, options ...instance.AddOption) {
		t.Helper()
		options = append(options, instance.Insecure(true))
		added, err := instance.Add(geneos.LOCAL, &Component, name, options...)
		if err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
		t.Cleanup(func() { _ = added.Unload() })
		if got := config.Get[uint16](added.Config(), "port"); got != want {
			t.Fatalf("add %s: port = %d, want %d", name, got, want)
		}
	}

	t.Run("default", func(t *testing.T) {
		if _, used := instance.GetAllPorts(geneos.LOCAL)[7041]; used {
			t.Skip("port 7041 is already in use")
		}
		add(t, "default", 7041)
	})

	t.Run("configured port", func(t *testing.T) {
		want := instance.NextFreePort(geneos.LOCAL, &Component)
		add(t, "configured", want)
	})

	t.Run("listening port", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = listener.Close() })
		port := listener.Addr().(*net.TCPAddr).Port
		config.Set(config.Global(), Component.PortRange, fmt.Sprintf("%d,7100-", port))
		want := instance.NextFreePort(geneos.LOCAL, &Component)
		if want == uint16(port) {
			t.Fatal("listening port was not detected")
		}
		add(t, "listening", want)
	})

	t.Run("explicit port", func(t *testing.T) {
		port := instance.NextFreePort(geneos.LOCAL, &Component)
		add(t, "explicit", port, instance.WithPort(port))
	})
}
