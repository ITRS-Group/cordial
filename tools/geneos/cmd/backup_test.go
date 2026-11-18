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
