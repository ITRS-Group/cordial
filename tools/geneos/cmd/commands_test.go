package cmd_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"

	_ "github.com/itrs-group/cordial/tools/geneos/cmd/pscmd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
)

func addGateways(t *testing.T, names ...string) (root string) {
	t.Helper()
	root = cmdtest.Env(t)
	for _, name := range names {
		if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", name); res.Err != nil {
			t.Fatalf("add %s: %v\n%s", name, res.Err, res.Stderr)
		}
	}
	return
}

// Commands that act on instances must not report success when the
// action failed.
func TestImportErrorsAreReported(t *testing.T) {
	if !cmdtest.StderrCaptured {
		t.Skip("standard error is not captured on this platform")
	}
	addGateways(t, "gw1")

	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"import", "gateway", "gw1", "/nonexistent/dir/file.xml"}, "no such file"},
	} {
		t.Run(test.args[0], func(t *testing.T) {
			res := cmdtest.Run(t, cmd.Cmd, test.args...)
			if res.Err == nil && !strings.Contains(res.Stdout+res.Stderr, test.want) {
				t.Errorf("%v produced no error and its output does not mention %q: %q", test.args, test.want, res.Stdout+res.Stderr)
			}
		})
	}
}

// Commands that used to ignore the error from parsing their arguments
func TestCommandsReturnArgumentErrors(t *testing.T) {
	addGateways(t, "gw1")

	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"unset", "gateway", "gw1", "foo=bar"}, "unexpected parameters"},
		{[]string{"unset", "gateway", "gw1"}, "nothing to unset"},
		{[]string{"snapshot", "gateway", "gw1"}, "no dataview xpath"},
		{[]string{"ps", "gateway", "foo=bar"}, "unexpected parameters"},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			res := cmdtest.Run(t, cmd.Cmd, test.args...)
			if res.Err == nil || !strings.Contains(res.Err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", res.Err, test.want)
			}
		})
	}
}

func TestUnsetReportsEveryInstance(t *testing.T) {
	if !cmdtest.StderrCaptured {
		t.Skip("standard error is not captured on this platform")
	}
	addGateways(t, "gw1", "gw2")
	res := cmdtest.Run(t, cmd.Cmd, "unset", "gateway", "-k", "nosuchkey")
	if got := strings.Count(res.Stdout+res.Stderr, "nothing unset"); got != 2 {
		t.Errorf("%d instances reported nothing unset, want 2:\n%s%s", got, res.Stdout, res.Stderr)
	}
}

func TestShowWritesToOutputFile(t *testing.T) {
	addGateways(t, "gw1")

	out := filepath.Join(t.TempDir(), "show.json")
	res := cmdtest.Run(t, cmd.Cmd, "show", "gateway", "gw1", "-o", out)
	if res.Err != nil {
		t.Fatalf("show: %v", res.Err)
	}
	if strings.Contains(res.Stdout, "gw1") {
		t.Errorf("output written to stdout: %s", res.Stdout)
	}
	if b, err := os.ReadFile(out); err != nil || !strings.Contains(string(b), `"name": "gw1"`) {
		t.Errorf("output file = %q, %v; want the instance configuration", b, err)
	}

	missing := filepath.Join(t.TempDir(), "never.json")
	res = cmdtest.Run(t, cmd.Cmd, "show", "gateway", "gw1", "-m", "-V", "-o", missing)
	if res.Err == nil {
		t.Error("show --merge --validate succeeded")
	}
	if _, err := os.Stat(missing); err == nil {
		t.Error("output file created before the options were checked")
	}
}

func TestRevertRemovesBothFiles(t *testing.T) {
	root := addGateways(t, "gw1")
	home := filepath.Join(root, "gateway", "gateways", "gw1")
	for _, f := range []string{"gateway.rc", "gateway.rc.orig"} {
		if err := os.WriteFile(filepath.Join(home, f), []byte("# rc\n"), 0o664); err != nil {
			t.Fatal(err)
		}
	}

	if res := cmdtest.Run(t, cmd.Cmd, "revert", "gateway", "gw1"); res.Err != nil {
		t.Fatalf("revert: %v", res.Err)
	}
	for _, f := range []string{"gateway.rc.orig", "gateway.json"} {
		if _, err := os.Stat(filepath.Join(home, f)); err == nil {
			t.Errorf("%s was not removed", f)
		}
	}
}

func TestLogoutAll(t *testing.T) {
	cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "logout", "--all"); res.Err != nil {
		t.Fatalf("logout --all: %v", res.Err)
	}
}

func TestMigrateExecutablesIsIdempotent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links are not used on windows")
	}
	cmdtest.Env(t)

	bin := t.TempDir()
	ctl := filepath.Join(bin, "gatewayctl")
	const original = "#!/bin/sh\necho original\n"
	if err := os.WriteFile(ctl, []byte(original), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	for range 2 {
		if res := cmdtest.Run(t, cmd.Cmd, "migrate", "-X"); res.Err != nil {
			t.Fatalf("migrate -X: %v", res.Err)
		}
	}

	if st, err := os.Lstat(ctl); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symbolic link: %v", ctl, err)
	}
	if b, err := os.ReadFile(ctl + ".orig"); err != nil || string(b) != original {
		t.Fatalf("%s.orig = %q, %v; want the original script", ctl, b, err)
	}
}
