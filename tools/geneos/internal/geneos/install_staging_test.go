package geneos

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

func TestUnarchiveFailureLeavesNoPartialRelease(t *testing.T) {
	root := t.TempDir()
	h := &Host{Host: host.NewLocal(), Config: config.New()}
	config.Set(h.Config, cordial.ExecutableName(), root)

	ct := &Component{Name: "stagetest"}
	registeredComponents[ct.Name] = ct
	t.Cleanup(func() { delete(registeredComponents, ct.Name) })

	const filename = "geneos-stagetest-7.1.0-linux-x64.tar.gz"
	release := filepath.Join(root, "packages", "stagetest", "7.1.0")

	// a valid first entry followed by a truncated stream
	var good bytes.Buffer
	gz := gzip.NewWriter(&good)
	tw := tar.NewWriter(gz)
	for _, name := range []string{"stagetest/bin/one", "stagetest/bin/two"} {
		data := bytes.Repeat([]byte("x"), 4096)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gz.Close()
	truncated := good.Bytes()[:good.Len()/2]

	if _, err := unarchive(h, ct, bytes.NewReader(truncated), filename, int64(len(truncated))); err == nil {
		t.Fatal("unarchive of truncated archive succeeded")
	}
	if _, err := os.Stat(release); !os.IsNotExist(err) {
		t.Fatalf("release directory exists after failed install: %v", err)
	}
	if _, err := os.Stat(StagingDir(h, release)); !os.IsNotExist(err) {
		t.Fatalf("staging directory left after failed install: %v", err)
	}

	// a leftover staging directory from an interrupted install is
	// neither listed nor an obstacle to a retry
	if err := os.MkdirAll(filepath.Join(StagingDir(h, release), "junk"), 0o755); err != nil {
		t.Fatal(err)
	}
	if versions, _ := InstalledReleases(h, ct); len(versions) != 0 {
		t.Fatalf("InstalledReleases = %v, want none", versions)
	}

	if _, err := unarchive(h, ct, bytes.NewReader(good.Bytes()), filename, int64(good.Len())); err != nil {
		t.Fatalf("retry unarchive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(release, "bin", "two")); err != nil {
		t.Fatalf("release not installed on retry: %v", err)
	}
	if _, err := os.Stat(filepath.Join(release, "junk")); !os.IsNotExist(err) {
		t.Fatal("stale staging content merged into release")
	}
	if versions, _ := InstalledReleases(h, ct); len(versions) != 1 || versions[0] != "7.1.0" {
		t.Fatalf("InstalledReleases = %v, want [7.1.0]", versions)
	}
}
