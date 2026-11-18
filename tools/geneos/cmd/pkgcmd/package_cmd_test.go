package pkgcmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
)

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var n []string
	for _, e := range entries {
		n = append(n, e.Name())
	}
	slices.Sort(n)
	return n
}

// only the cached downloads for the release being removed are removed
func TestUninstallRemovesOnlyMatchingDownloads(t *testing.T) {
	root := cmdtest.Env(t)
	packages := filepath.Join(root, "packages")
	for _, d := range []string{"netprobe/1.0.0", "netprobe/2.0.0", "downloads"} {
		if err := os.MkdirAll(filepath.Join(packages, d), 0o775); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{
		"geneos-netprobe-standard-1.0.0-linux-x64.tar.gz",
		"geneos-netprobe-standard-2.0.0-linux-x64.tar.gz",
		"geneos-gateway-standard-1.0.0-linux-x64.tar.gz",
	} {
		if err := os.WriteFile(filepath.Join(packages, "downloads", f), []byte("x"), 0o664); err != nil {
			t.Fatal(err)
		}
	}

	if res := cmdtest.Run(t, cmd.Cmd, "package", "uninstall", "netprobe", "-V", "1.0.0"); res.Err != nil {
		t.Fatalf("uninstall: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}

	if got, want := names(t, filepath.Join(packages, "netprobe")), []string{"2.0.0"}; !slices.Equal(got, want) {
		t.Errorf("releases = %v, want %v", got, want)
	}
	want := []string{
		"geneos-gateway-standard-1.0.0-linux-x64.tar.gz",
		"geneos-netprobe-standard-2.0.0-linux-x64.tar.gz",
	}
	if got := names(t, filepath.Join(packages, "downloads")); !slices.Equal(got, want) {
		t.Errorf("cached downloads = %v, want %v", got, want)
	}
}

func TestInstallRejectsBadOverride(t *testing.T) {
	cmdtest.Env(t)
	archive := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(archive, []byte("x"), 0o664); err != nil {
		t.Fatal(err)
	}
	res := cmdtest.Run(t, cmd.Cmd, "package", "install", "--override", "notacomponent:1.0.0", archive)
	if res.Err == nil {
		t.Fatalf("install with a bad override succeeded: %s%s", res.Stdout, res.Stderr)
	}
}
