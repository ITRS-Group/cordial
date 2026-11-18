package geneos

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeTestArchive writes a gzipped tar archive with one file in a
// single top-level directory to dir and returns the path
func writeTestArchive(t *testing.T, dir, filename, top string) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	data := []byte("content")
	if err := tw.WriteHeader(&tar.Header{Name: top + "/bin/program", Mode: 0o755, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInstallComponentMismatch(t *testing.T) {
	root := t.TempDir()
	setTestHosts(t, root)

	a, b := &Component{Name: "mismatcha"}, &Component{Name: "mismatchb"}
	composite := &Component{Name: "mismatchcomposite", PackageTypes: []*Component{a, b}}
	for _, ct := range []*Component{a, b, composite} {
		registeredComponents[ct.Name] = ct
		t.Cleanup(func() { delete(registeredComponents, ct.Name) })
	}

	archive := writeTestArchive(t, t.TempDir(), "geneos-mismatchb-7.1.0-linux-x64.tar.gz", "mismatchb")

	t.Run("single component", func(t *testing.T) {
		err := Install(LOCAL, a, Source(archive))
		if !errors.Is(err, ErrInvalidArgs) {
			t.Fatalf("Install() of an archive for another component error = %v, want ErrInvalidArgs", err)
		}
		if _, err := os.Stat(filepath.Join(root, "packages", "mismatcha")); !os.IsNotExist(err) {
			t.Errorf("a release directory was created for the wrong component: %v", err)
		}
	})

	t.Run("package types", func(t *testing.T) {
		// the archive is installed for the component that matches and
		// the others are skipped
		if err := Install(LOCAL, composite, Source(archive)); err != nil {
			t.Fatalf("Install() error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "packages", "mismatchb", "7.1.0", "bin", "program")); err != nil {
			t.Errorf("release was not installed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "packages", "mismatcha")); !os.IsNotExist(err) {
			t.Errorf("a release directory was created for the wrong component: %v", err)
		}
		// the default base name is used when none is given
		if target, err := os.Readlink(filepath.Join(root, "packages", "mismatchb", "active_prod")); err != nil || target != "7.1.0" {
			t.Errorf("active_prod = %q, %v", target, err)
		}
	})

	t.Run("no matching package type", func(t *testing.T) {
		other := writeTestArchive(t, t.TempDir(), "geneos-mismatchc-7.1.0-linux-x64.tar.gz", "mismatchc")
		registeredComponents["mismatchc"] = &Component{Name: "mismatchc"}
		t.Cleanup(func() { delete(registeredComponents, "mismatchc") })
		if err := Install(LOCAL, composite, Source(other)); err == nil {
			t.Error("Install() succeeded when no package type matches the archive")
		}
	})
}
