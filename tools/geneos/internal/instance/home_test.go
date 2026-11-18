package instance_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// countHost counts the calls that are a round trip on a remote host
type countHost struct {
	host.Host
	stats atomic.Int64
}

func (c *countHost) Stat(name string) (fs.FileInfo, error) {
	c.stats.Add(1)
	return c.Host.Stat(name)
}

// useCountHost makes the local test host count its Stat calls
func useCountHost(t *testing.T) *countHost {
	t.Helper()
	ch := &countHost{Host: geneos.LOCAL.Host}
	geneos.LOCAL.Host = ch
	return ch
}

// statsOf returns the number of Stat calls made by f
func statsOf(ch *countHost, f func()) int64 {
	before := ch.stats.Load()
	f()
	return ch.stats.Load() - before
}

// requireCached checks that repeated calls that depend on the home
// directory do not use the host
func requireCached(t *testing.T, ch *countHost, i geneos.Instance) {
	t.Helper()
	i.Home()
	if n := statsOf(ch, func() {
		for range 5 {
			i.Home()
			instance.PathTo(i, "certificate")
			instance.ComponentFilepath(i)
			instance.HomeRel(i, "relative.file")
			instance.LogFilePath(i)
			instance.BaseVersion(i)
		}
	}); n != 0 {
		t.Fatalf("%d Stat calls for repeated Home() and PathTo() calls, want none", n)
	}
}

// requireReset checks that the home directory has to be resolved again
func requireReset(t *testing.T, ch *countHost, i geneos.Instance, want string) {
	t.Helper()
	var got string
	if n := statsOf(ch, func() { got = i.Home() }); n == 0 {
		t.Errorf("Home() did not use the host after the operation, a cached directory was used")
	}
	if got != want {
		t.Errorf("Home() = %q, want %q", got, want)
	}
}

func TestHomeIsCached(t *testing.T) {
	root := newTestRoot(t)
	ch := useCountHost(t)

	i, err := instance.Add(geneos.LOCAL, &licd.Component, "home-cache", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Unload() })
	config.Set(i.Config(), "certificate", "licd.pem")

	want := filepath.Join(root, "licd", "licds", "home-cache")
	if got := i.Home(); got != want {
		t.Fatalf("Home() = %q, want %q", got, want)
	}
	requireCached(t, ch, i)
	if got, want := instance.PathTo(i, "certificate"), filepath.Join(want, "licd.pem"); got != want {
		t.Errorf("PathTo() = %q, want %q", got, want)
	}

	// each instance has its own entry
	j, err := instance.Add(geneos.LOCAL, &licd.Component, "home-cache-2", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = j.Unload() })
	requireCached(t, ch, j)
	requireCached(t, ch, i)
	if i.Home() == j.Home() {
		t.Errorf("two instances have the same home %q", i.Home())
	}
}

// TestHomeNotFoundIsNotCached checks that the default for a directory
// that does not exist yet is not remembered
func TestHomeNotFoundIsNotCached(t *testing.T) {
	root := newTestRoot(t)
	ch := useCountHost(t)

	i := licd.Component.New("not-created")
	want := filepath.Join(root, "licd", "licds", "not-created")
	for range 2 {
		requireReset(t, ch, i, want)
	}

	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	requireReset(t, ch, i, want)
	requireCached(t, ch, i)
}

func TestHomeNoticesConfigChanges(t *testing.T) {
	root := newTestRoot(t)
	ch := useCountHost(t)

	i, err := instance.Add(geneos.LOCAL, &licd.Component, "home-config", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Unload() })
	standard := filepath.Join(root, "licd", "licds", "home-config")
	requireCached(t, ch, i)

	custom := filepath.Join(root, "custom", "home-config")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}

	config.Set(i.Config(), "home", custom)
	requireReset(t, ch, i, custom)
	requireCached(t, ch, i)

	config.Delete(i.Config(), "home")
	requireReset(t, ch, i, standard)
	requireCached(t, ch, i)

	// a configured directory that does not exist is skipped
	config.Set(i.Config(), "home", filepath.Join(root, "custom", "missing"))
	requireReset(t, ch, i, standard)
}

// TestHomeIsResetByOperations checks that everything that can change
// the directory of an instance means it is looked for again
func TestHomeIsResetByOperations(t *testing.T) {
	root := newTestRoot(t)
	ch := useCountHost(t)
	if err := os.MkdirAll(filepath.Join(root, "packages", "netprobe", "active_prod"), 0o755); err != nil {
		t.Fatal(err)
	}

	add := func(t *testing.T, name string) geneos.Instance {
		t.Helper()
		i, err := instance.Add(geneos.LOCAL, &licd.Component, name, instance.Insecure(true))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = i.Unload() })
		return i
	}

	t.Run("add", func(t *testing.T) {
		i := add(t, "reset-add")
		requireReset(t, ch, i, filepath.Join(root, "licd", "licds", "reset-add"))
	})

	t.Run("unload", func(t *testing.T) {
		i := add(t, "reset-unload")
		custom := filepath.Join(root, "custom", "reset-unload")
		if err := os.MkdirAll(custom, 0o755); err != nil {
			t.Fatal(err)
		}
		config.Set(i.Config(), "home", custom)
		requireCached(t, ch, i)

		// the configuration is unchanged, but the directory has gone
		if err := os.RemoveAll(custom); err != nil {
			t.Fatal(err)
		}
		if got := i.Home(); got != custom {
			t.Fatalf("Home() = %q before reset, want the cached %q", got, custom)
		}
		if err := i.Unload(); err != nil {
			t.Fatal(err)
		}
		requireReset(t, ch, i, filepath.Join(root, "licd", "licds", "reset-unload"))
	})

	t.Run("load", func(t *testing.T) {
		i := add(t, "reset-load")
		custom := filepath.Join(root, "custom", "reset-load")
		if err := os.MkdirAll(custom, 0o755); err != nil {
			t.Fatal(err)
		}
		config.Set(i.Config(), "home", custom)
		requireCached(t, ch, i)

		// the configuration is unchanged, but the directory has gone
		if err := os.RemoveAll(custom); err != nil {
			t.Fatal(err)
		}
		if got := i.Home(); got != custom {
			t.Fatalf("Home() = %q before Load(), want the cached %q", got, custom)
		}
		if err := i.Load(); err != nil {
			t.Fatal(err)
		}
		if got, want := i.Home(), filepath.Join(root, "licd", "licds", "reset-load"); got != want {
			t.Errorf("Home() = %q after Load(), want %q", got, want)
		}
	})

	t.Run("write", func(t *testing.T) {
		i := add(t, "reset-write")
		requireCached(t, ch, i)
		if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
			t.Fatal(resp.Err)
		}
		requireReset(t, ch, i, filepath.Join(root, "licd", "licds", "reset-write"))
	})

	t.Run("refactor config", func(t *testing.T) {
		i := add(t, "reset-refactor")
		requireCached(t, ch, i)
		// the aliases of the new configuration are not wanted here
		cf := config.New()
		config.Set(cf, "name", "reset-refactor")
		config.Set(cf, "home", i.Home())
		i.SetConfig(cf)
		requireCached(t, ch, i)
		newDir := filepath.Join(root, "licd", "licds", "reset-refactor-new")
		if err := os.MkdirAll(newDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := instance.RefactorConfig(geneos.LOCAL, &licd.Component, cf, instance.NewDir(newDir)); err != nil {
			t.Fatal(err)
		}
		requireReset(t, ch, i, newDir)
	})

	t.Run("move", func(t *testing.T) {
		i := add(t, "reset-move")
		requireCached(t, ch, i)
		stale := licd.Component.New("reset-move")
		requireCached(t, ch, stale)
		if err := instance.Copy(&licd.Component, "reset-move", "reset-moved", instance.Move()); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if j, err := instance.Get(&licd.Component, "reset-moved"); err == nil {
				_ = j.Unload()
			}
		})
		// a different value for the same instance is reset too
		requireReset(t, ch, i, filepath.Join(root, "licd", "licds", "reset-move"))
		requireReset(t, ch, stale, filepath.Join(root, "licd", "licds", "reset-move"))

		j, err := instance.Get(&licd.Component, "reset-moved")
		if err != nil {
			t.Fatal(err)
		}
		if got, want := j.Home(), filepath.Join(root, "licd", "licds", "reset-moved"); got != want {
			t.Errorf("Home() of the moved instance = %q, want %q", got, want)
		}
		requireCached(t, ch, j)
	})

	t.Run("copy", func(t *testing.T) {
		i := add(t, "reset-copy")
		requireCached(t, ch, i)
		if err := instance.Copy(&licd.Component, "reset-copy", "reset-copied"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if j, err := instance.Get(&licd.Component, "reset-copied"); err == nil {
				_ = j.Unload()
			}
		})
		j, err := instance.Get(&licd.Component, "reset-copied")
		if err != nil {
			t.Fatal(err)
		}
		if got, want := j.Home(), filepath.Join(root, "licd", "licds", "reset-copied"); got != want {
			t.Errorf("Home() of the copy = %q, want %q", got, want)
		}
		requireCached(t, ch, i)
	})

	t.Run("migrate", func(t *testing.T) {
		legacy := filepath.Join(root, "fa2", "fa2s", "reset-migrate")
		if err := os.MkdirAll(legacy, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(legacy, "fa2.json"), []byte(`{"name":"reset-migrate"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		i, err := instance.Get(&fa2.Component, "reset-migrate")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = i.Unload() })
		if got := i.Home(); got != legacy {
			t.Fatalf("Home() = %q, want the legacy directory %q", got, legacy)
		}
		requireCached(t, ch, i)

		if resp := instance.Migrate(i); resp.Err != nil {
			t.Fatal(resp.Err)
		}
		requireReset(t, ch, i, filepath.Join(root, "netprobe", "fa2s", "reset-migrate"))
	})
}
