package cmd_test

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
)

func tarNames(t *testing.T, r io.Reader) map[string]string {
	t.Helper()
	names := map[string]string{}
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return names
		}
		if err != nil {
			t.Fatalf("not a valid tar stream: %v", err)
		}
		b, _ := io.ReadAll(tr)
		names[h.Name] = string(b)
	}
}

func TestBackupSharedSymlinks(t *testing.T) {
	root := cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "gw-links"); res.Err != nil {
		t.Fatalf("add: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
	dir := filepath.Join(root, "gateway", "includes")
	versionDir := filepath.Join(dir, "itrs-v0.0.1")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(versionDir, "itrs.custom.xml")
	if err := os.WriteFile(target, []byte("primary include"), 0o664); err != nil {
		t.Fatal(err)
	}
	links := map[string]string{
		"itrs.custom.xml":   "itrs-v0.0.1/itrs.custom.xml",
		"itrs.absolute.xml": target,
		"itrs.dangling.xml": "itrs-v0.0.1/missing.xml",
	}
	for name, link := range links {
		if err := os.Symlink(link, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	res := cmdtest.Run(t, cmd.Cmd, "backup", "gateway", "gw-links", "--shared", "-o", "-", "-z", "none")
	if res.Err != nil {
		t.Fatalf("backup: %v\n%s", res.Err, res.Stderr)
	}
	gotLinks := map[string]string{}
	tr := tar.NewReader(strings.NewReader(res.Stdout))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeSymlink {
			gotLinks[header.Name] = header.Linkname
		}
	}
	for name, want := range map[string]string{
		"itrs.custom.xml":   "itrs-v0.0.1/itrs.custom.xml",
		"itrs.absolute.xml": "itrs-v0.0.1/itrs.custom.xml",
		"itrs.dangling.xml": "itrs-v0.0.1/missing.xml",
	} {
		if got := gotLinks["gateway/includes/"+name]; got != want {
			t.Errorf("archived symlink %s = %q, want %q", name, got, want)
		}
	}
	if got := tarNames(t, strings.NewReader(res.Stdout))["gateway/includes/itrs-v0.0.1/itrs.custom.xml"]; got != "primary include" {
		t.Errorf("archived target contents = %q", got)
	}
	t.Run("restore command round trip", func(t *testing.T) {
		archive := filepath.Join(t.TempDir(), "backup.tar")
		if err := os.WriteFile(archive, []byte(res.Stdout), 0o600); err != nil {
			t.Fatal(err)
		}
		for name := range links {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
		}
		result := cmdtest.Run(t, cmd.Cmd, "restore", "--shared", archive)
		if result.Err != nil {
			t.Fatalf("restore: %v\n%s%s", result.Err, result.Stdout, result.Stderr)
		}
		for name, want := range map[string]string{
			"itrs.custom.xml":   "itrs-v0.0.1/itrs.custom.xml",
			"itrs.absolute.xml": "itrs-v0.0.1/itrs.custom.xml",
			"itrs.dangling.xml": "itrs-v0.0.1/missing.xml",
		} {
			if got, err := os.Readlink(filepath.Join(dir, name)); err != nil || got != want {
				t.Errorf("restored %s = %q, %v; want %q", name, got, err, want)
			}
		}
	})
	t.Run("reject outside target", func(t *testing.T) {
		if err := os.Symlink(t.TempDir(), filepath.Join(dir, "external")); err != nil {
			t.Fatal(err)
		}
		res := cmdtest.Run(t, cmd.Cmd, "backup", "gateway", "gw-links", "--shared", "-o", "-", "-z", "none")
		if res.Err == nil || !strings.Contains(res.Err.Error(), "outside") {
			t.Errorf("backup error = %v, want unsafe external symlink error", res.Err)
		}
	})
}

func TestBackup(t *testing.T) {
	root := cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "gw1"); res.Err != nil {
		t.Fatalf("add: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
	home := filepath.Join(root, "gateway", "gateways", "gw1")
	if err := os.WriteFile(filepath.Join(home, "extra.json"), []byte("hello"), 0o664); err != nil {
		t.Fatal(err)
	}
	// special files must not stop the backup
	if err := os.Symlink("extra.json", filepath.Join(home, "link")); err != nil {
		t.Fatal(err)
	}

	t.Run("stream to stdout", func(t *testing.T) {
		res := cmdtest.Run(t, cmd.Cmd, "backup", "gateway", "gw1", "-o", "-", "-z", "none", "--shared")
		if res.Err != nil {
			t.Fatalf("backup: %v", res.Err)
		}
		// the messages must have gone to stderr, leaving a valid archive
		names := tarNames(t, strings.NewReader(res.Stdout))
		if names["gateway/gateways/gw1/extra.json"] != "hello" {
			t.Errorf("archive entries = %v", slices.Sorted(maps.Keys(names)))
		}
		if !strings.Contains(res.Stderr, "matching shared directories also included") {
			t.Errorf("stderr = %q, want shared message", res.Stderr)
		}
	})

	t.Run("gzip to new directory", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "new", "dir") + "/"
		res := cmdtest.Run(t, cmd.Cmd, "backup", "gateway", "gw1", "-o", dest)
		if res.Err != nil {
			t.Fatalf("backup: %v", res.Err)
		}
		st, err := os.Stat(dest)
		if err != nil || st.Mode().Perm()&0o100 == 0 {
			t.Fatalf("destination directory %v, %v: want searchable directory", st, err)
		}
		f, err := os.Open(filepath.Join(dest, cordial.ExecutableName()+"-gateway-gw1.tar.gz"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		zr, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		if names := tarNames(t, zr); names["gateway/gateways/gw1/extra.json"] != "hello" {
			t.Errorf("archive entries = %v", slices.Sorted(maps.Keys(names)))
		}
	})

	t.Run("invalid compression", func(t *testing.T) {
		res := cmdtest.Run(t, cmd.Cmd, "backup", "gateway", "gw1", "-z", "zip")
		want := "invalid compression type, select one of: bzip2, gzip, none"
		if res.Err == nil || res.Err.Error() != want {
			t.Fatalf("error = %v, want %q", res.Err, want)
		}
	})
}
