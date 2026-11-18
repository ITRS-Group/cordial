package initcmd

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestInitRejectsEmptyArchive(t *testing.T) {
	flag := initCmd.PersistentFlags().Lookup("archive")
	previousArchive, previousChanged := initOptions.Archive, flag.Changed
	initOptions.Archive, flag.Changed = "", true
	t.Cleanup(func() { initOptions.Archive, flag.Changed = previousArchive, previousChanged })
	if err := initCmd.RunE(initCmd, nil); err == nil || !strings.Contains(err.Error(), "--archive must not be empty") {
		t.Fatalf("init error = %v, want empty archive error", err)
	}
}

func TestInitRestore(t *testing.T) {
	cmdtest.Env(t)
	tmp := t.TempDir()
	backup := filepath.Join(tmp, "backup.tar.gz")
	writeInitArchive(t, backup, map[string]string{
		"licd/licds/restored/licd.json": `{"name":"restored"}`,
		"gateway/shared/restored.txt":   "shared contents",
	})
	writeInitArchive(t, filepath.Join(tmp, "geneos-licd-7.1.0-linux-x64.tar.gz"), map[string]string{
		"licd/licd.linux_64": "fixture",
	})

	for _, tc := range []struct {
		name        string
		args        []string
		wantInstall bool
	}{
		{"install", []string{"--restore", backup, "--archive", tmp, "--version", "7.1.0", "--username", "unused"}, true},
		{"skip", []string{"-R", backup, "-X", "--archive", tmp}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmdtest.Env(t)
			root := filepath.Join(t.TempDir(), "geneos")
			res := cmdtest.Run(t, cmd.Cmd, append([]string{"init", "--insecure", root}, tc.args...)...)
			if res.Err != nil {
				t.Fatalf("init: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
			}
			if _, err := os.Stat(filepath.Join(root, "licd", "licds", "restored", "licd.json")); err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(filepath.Join(root, "gateway", "shared", "restored.txt")); err != nil || string(got) != "shared contents" {
				t.Fatalf("restored shared file = %q, %v", got, err)
			}
			_, err := os.Stat(filepath.Join(root, "packages", "licd", "active_prod", "licd.linux_64"))
			if tc.wantInstall && err != nil || !tc.wantInstall && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("stat installed release = %v (want install: %v)", err, tc.wantInstall)
			}
		})
	}
}

func TestInitRestoreMissingBackup(t *testing.T) {
	cmdtest.Env(t)
	tmp := t.TempDir()
	backup := filepath.Join(tmp, "missing.tar.gz")
	res := cmdtest.Run(t, cmd.Cmd, "init", "--insecure", "--restore", backup, "--no-install", filepath.Join(tmp, "geneos"))
	if !errors.Is(res.Err, os.ErrNotExist) || !strings.Contains(res.Err.Error(), backup) {
		t.Fatalf("init error = %v, want missing backup error", res.Err)
	}
}

func writeInitArchive(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, body := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := errors.Join(tw.Close(), gz.Close(), f.Close()); err != nil {
		t.Fatal(err)
	}
}

func TestInitParams(t *testing.T) {
	cmdtest.Env(t)
	root := filepath.Join(t.TempDir(), "geneos")

	res := cmdtest.Run(t, cmd.Cmd, "init", "--insecure", root, "key=value")
	if res.Err != nil {
		t.Fatalf("init: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
	if want := []string{"key=value"}; !slices.Equal(initOptions.Extras.Params, want) {
		t.Errorf("params = %q, want %q (without the root directory and not duplicated)", initOptions.Extras.Params, want)
	}
}

// root CA and signing certificates must never be replaced by init
func TestInitKeepsCertificates(t *testing.T) {
	cmdtest.Env(t)
	root := filepath.Join(t.TempDir(), "geneos")

	read := func() map[string]string {
		files := map[string]string{}
		for _, pattern := range []string{geneos.RootCABasename + ".*", geneos.SigningCertBasename + ".*"} {
			matches, _ := filepath.Glob(filepath.Join(config.AppConfigDir(), pattern))
			for _, m := range matches {
				b, err := os.ReadFile(m)
				if err != nil {
					t.Fatal(err)
				}
				files[filepath.Base(m)] = string(b)
			}
		}
		return files
	}

	if res := cmdtest.Run(t, cmd.Cmd, "init", root); res.Err != nil {
		t.Fatalf("init: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
	first := read()
	if len(first) == 0 {
		t.Fatal("no certificates created by init")
	}

	for _, args := range [][]string{{"init", "--force", root}, {"init", "--force", root}} {
		if res := cmdtest.Run(t, cmd.Cmd, args...); res.Err != nil {
			t.Fatalf("%v: %v\n%s%s", args, res.Err, res.Stdout, res.Stderr)
		}
		if got := read(); !maps.Equal(first, got) {
			t.Fatalf("%v replaced the root or signing certificates, which init must not do", args)
		}
	}
}

// init of a directory that is not empty fails, without --force, with an
// error and not by ending the process
func TestInitNotEmpty(t *testing.T) {
	cmdtest.Env(t)
	root := filepath.Join(t.TempDir(), "geneos")

	if res := cmdtest.Run(t, cmd.Cmd, "init", "--insecure", root); res.Err != nil {
		t.Fatalf("init: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}

	res := cmdtest.Run(t, cmd.Cmd, "init", "--insecure", root)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "not empty") || !strings.Contains(res.Err.Error(), "--force") {
		t.Fatalf("second init error = %v, want a directory not empty error that refers to --force", res.Err)
	}

	if res := cmdtest.Run(t, cmd.Cmd, "init", "--insecure", "--force", root); res.Err != nil {
		t.Fatalf("init --force: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
}

// the directory is initialised once, so that the second profile does not
// fail because of what was done for the first, and the profiles that
// fail do so on their own account
func TestInitProfilesInitialiseOnce(t *testing.T) {
	cmdtest.Env(t)
	root := filepath.Join(t.TempDir(), "geneos")

	res := cmdtest.Run(t, cmd.Cmd, "init", "--insecure", "noprofile1", "noprofile2", root)
	if !errors.Is(res.Err, cmd.ErrReported) {
		t.Fatalf("init error = %v, want the error for the profiles that failed, already reported\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
	if !cmdtest.StderrCaptured {
		return
	}
	if strings.Contains(res.Stderr, "not empty") {
		t.Errorf("the directory was initialised more than once:\n%s", res.Stderr)
	}
	for _, name := range []string{"noprofile1", "noprofile2"} {
		if n := strings.Count(res.Stderr, name+": "); n != 1 {
			t.Errorf("%s is reported %d times, want once:\n%s", name, n, res.Stderr)
		}
	}
}
