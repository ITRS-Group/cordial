package cmd_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"

	_ "github.com/itrs-group/cordial/tools/geneos/cmd/aescmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/hostcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/pkgcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/tlscmd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
)

// setupInstances creates the gateways gw1 and gw2 and a licd lic1, none
// of which can run as there is no package installed
func setupInstances(t *testing.T) {
	t.Helper()
	cmdtest.Env(t)
	for _, args := range [][]string{
		{"add", "gateway", "gw1"},
		{"add", "gateway", "gw2"},
		{"add", "licd", "lic1"},
		{"tls", "init"},
	} {
		if res := cmdtest.Run(t, cmd.Cmd, args...); res.Err != nil {
			t.Fatalf("%v: %v\n%s", args, res.Err, res.Stderr)
		}
	}
}

// The exit status of a command must be non-zero if any of the instances,
// or anything else it acts on, failed, but not for the things that are
// not an error by design.
func TestExitStatus(t *testing.T) {
	// the commands that write the configuration are given one instance
	// at a time as writing several concurrently is a known data race
	for _, test := range []struct {
		name string
		args []string
		// wantErr is true if the command must fail
		wantErr bool
		// stdout is output that must be written, to show that the rest of
		// the items were still processed
		stdout string
		// stderr is the one failure that must be reported, once
		stderr string
	}{
		// the instances that do exist are processed, in the order given,
		// and the one that does not fails the command
		{name: "ls unknown", args: []string{"ls", "gateway", "gw1", "nosuch"}, wantErr: true, stdout: "gw1", stderr: "nosuch"},
		{name: "ls only unknown", args: []string{"ls", "gateway", "nosuch"}, wantErr: true, stderr: "nosuch"},
		{name: "ls csv unknown", args: []string{"ls", "--csv", "gateway", "gw1", "nosuch"}, wantErr: true, stdout: "gw1", stderr: "nosuch"},
		{name: "ls json unknown", args: []string{"ls", "--json", "gateway", "gw1", "nosuch"}, wantErr: true, stdout: "gw1", stderr: "nosuch"},
		{name: "ps unknown", args: []string{"ps", "gateway", "nosuch"}, wantErr: true, stderr: "nosuch"},
		{name: "show unknown", args: []string{"show", "gateway", "gw1", "nosuch"}, wantErr: true, stdout: "gw1", stderr: "nosuch"},
		{name: "set unknown", args: []string{"set", "gateway", "gw1", "nosuch", "key=value"}, wantErr: true, stdout: "gw1", stderr: "nosuch"},
		{name: "start unknown", args: []string{"start", "gateway", "nosuch"}, wantErr: true, stderr: "nosuch"},
		{name: "logs unknown", args: []string{"logs", "gateway", "gw1", "nosuch"}, wantErr: true, stdout: "gw1", stderr: "nosuch"},
		{name: "aes list unknown", args: []string{"aes", "list", "gateway", "gw1", "nosuch"}, wantErr: true, stderr: "nosuch"},
		{name: "tls list unknown", args: []string{"tls", "list", "gateway", "gw1", "nosuch"}, wantErr: true, stderr: "nosuch"},
		{name: "host show unknown", args: []string{"host", "show", "nosuch"}, wantErr: true, stderr: "nosuch"},
		{name: "host set unknown", args: []string{"host", "set", "nosuch", "key=value"}, wantErr: true, stderr: "nosuch"},
		{name: "host delete unknown", args: []string{"host", "delete", "nosuch"}, wantErr: true, stderr: "nosuch"},

		// items that fail for any other reason
		{name: "start disabled by name", args: []string{"start", "gateway", "gw1"}, wantErr: true, stderr: "disabled"},
		{name: "unset nothing", args: []string{"unset", "gateway", "gw1", "gw2", "-k", "nosuchkey"}, wantErr: true, stderr: "nothing unset"},
		{name: "import missing file", args: []string{"import", "gateway", "gw1", "/nonexistent/dir/file.xml"}, wantErr: true},
		{name: "copy existing", args: []string{"copy", "gateway", "gw1", "gw2"}, wantErr: true},
		{name: "copy unknown", args: []string{"copy", "gateway", "nosuch", "gw3"}, wantErr: true},
		{name: "move unknown", args: []string{"move", "gateway", "nosuch", "gw3"}, wantErr: true},
		{name: "copy unknown without type", args: []string{"copy", "nosuch", "gw3"}, wantErr: true},
		{name: "copy without type", args: []string{"copy", "gw1", "gw3"}},
		{name: "host unset unknown", args: []string{"host", "unset", "nosuch", "-k", "key"}, wantErr: true, stderr: "nosuch"},
		{name: "host list unknown", args: []string{"host", "list", "nosuch"}, wantErr: true, stderr: "nosuch"},
		{name: "delete protected", args: []string{"delete", "gateway", "gw1"}, wantErr: true, stderr: "protected"},
		{name: "disable protected", args: []string{"disable", "gateway", "gw1"}, wantErr: true, stderr: "protected"},
		{name: "revert protected", args: []string{"revert", "gateway", "gw1"}, wantErr: true, stderr: "protected"},
		{name: "command without a package", args: []string{"command", "gateway", "gw1", "gw2"}, wantErr: true, stderr: "gw2"},
		{name: "tls renew without a certificate", args: []string{"tls", "renew", "gateway", "gw1", "gw2"}, wantErr: true, stderr: "gw2"},
		{name: "snapshot unreachable", args: []string{"snapshot", "gateway", "gw1", "/geneos"}, wantErr: true},
		{name: "package uninstall unknown release", args: []string{"package", "uninstall", "gateway", "99.9.9"}, wantErr: true},
		{name: "add existing", args: []string{"add", "gateway", "gw1"}, wantErr: true},
		{name: "deploy existing", args: []string{"deploy", "gateway", "gw1"}, wantErr: true},

		// commands that check all of their arguments before acting, so
		// that nothing is changed
		{name: "stop unknown", args: []string{"stop", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "restart unknown", args: []string{"restart", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "reload unknown", args: []string{"reload", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "rebuild unknown", args: []string{"rebuild", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "clean unknown", args: []string{"clean", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "reset unknown", args: []string{"reset", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "unset unknown", args: []string{"unset", "gateway", "gw1", "nosuch", "-k", "key"}, wantErr: true},
		{name: "enable unknown", args: []string{"enable", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "disable unknown", args: []string{"disable", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "protect unknown", args: []string{"protect", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "delete unknown", args: []string{"delete", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "import unknown", args: []string{"import", "gateway", "nosuch", "file.xml"}, wantErr: true},
		{name: "command unknown", args: []string{"command", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "snapshot unknown", args: []string{"snapshot", "gateway", "nosuch", "/geneos"}, wantErr: true},
		{name: "backup unknown", args: []string{"backup", "gateway", "gw1", "nosuch", "-o", "-"}, wantErr: true},
		{name: "tls new unknown", args: []string{"tls", "new", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "tls renew unknown", args: []string{"tls", "renew", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "tls migrate unknown", args: []string{"tls", "migrate", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "tls export unknown", args: []string{"tls", "export", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "tls import unknown", args: []string{"tls", "import", "gateway", "nosuch", "-"}, wantErr: true},
		{name: "aes set unknown", args: []string{"aes", "set", "--crc", "00000000", "gateway", "nosuch"}, wantErr: true},
		{name: "aes new unknown", args: []string{"aes", "new", "--update", "gateway", "gw1", "nosuch"}, wantErr: true},
		{name: "aes encode unknown", args: []string{"aes", "encode", "--string", "secret", "gateway", "nosuch"}, wantErr: true},

		// what is not an error by design: stopping what is not running,
		// not supported by a component, nothing to list and the
		// instances that are all disabled
		{name: "stop not running", args: []string{"stop", "gateway", "gw1", "gw2"}},
		{name: "restart not running", args: []string{"restart", "gateway", "gw1", "gw2"}},
		{name: "reload not running", args: []string{"reload", "gateway", "gw1", "gw2"}},
		{name: "reload not supported", args: []string{"reload", "licd", "lic1"}},
		{name: "rebuild", args: []string{"rebuild", "gateway", "gw1"}},
		{name: "clean", args: []string{"clean", "gateway", "gw1", "gw2"}},
		{name: "ls", args: []string{"ls"}, stdout: "gw1"},
		{name: "ls type", args: []string{"ls", "gateway", "gw1", "gw2"}, stdout: "gw2"},
		{name: "ls none of a type", args: []string{"ls", "netprobe"}},
		{name: "ps not running", args: []string{"ps"}},
		{name: "ps named not running", args: []string{"ps", "gateway", "gw1"}},
		{name: "set", args: []string{"set", "gateway", "gw1", "key=value"}, stdout: "gw1"},
		{name: "protect", args: []string{"protect", "gateway", "gw1"}, stdout: "gw1"},
		{name: "enable enabled", args: []string{"enable", "gateway", "gw1", "gw2"}},
		{name: "disable", args: []string{"disable", "gateway", "gw1", "gw2"}, stdout: "gw2"},
		{name: "copy", args: []string{"copy", "gateway", "gw1", "gw3"}},
		{name: "move", args: []string{"move", "gateway", "gw1", "gw3"}},
		{name: "delete", args: []string{"delete", "gateway", "gw1", "gw2"}, stdout: "gw2"},
		{name: "logs", args: []string{"logs", "gateway", "gw1", "gw2"}, stdout: "gw2"},
		{name: "tls new", args: []string{"tls", "new", "gateway", "gw1"}, stdout: "gw1"},
		{name: "tls list", args: []string{"tls", "list", "gateway", "gw1", "gw2"}},
		{name: "aes list", args: []string{"aes", "list", "gateway", "gw1", "gw2"}},
		{name: "aes new", args: []string{"aes", "new", "--update", "gateway", "gw1"}, stdout: "gw1"},
		{name: "host list", args: []string{"host", "list"}},
		{name: "package list", args: []string{"package", "list"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupInstances(t)
			// the commands that are expected to fail on a protected
			// instance or a disabled one need the instance to be so first
			switch {
			case strings.Contains(test.name, "protected"):
				mustRun(t, "protect", "gateway", "gw1")
			case test.name == "start disabled by name":
				mustRun(t, "disable", "gateway", "gw1")
			case test.name == "enable enabled":
				mustRun(t, "enable", "gateway", "gw1")
			case test.name == "move" || strings.HasPrefix(test.name, "copy"):
				mustRun(t, "stop", "gateway", "gw1")
			}

			res := cmdtest.Run(t, cmd.Cmd, test.args...)
			if gotErr := res.Err != nil; gotErr != test.wantErr {
				t.Fatalf("%v: error = %v, want error %v\nstdout: %s\nstderr: %s", test.args, res.Err, test.wantErr, res.Stdout, res.Stderr)
			}
			if !strings.Contains(res.Stdout, test.stdout) {
				t.Errorf("%v: stdout does not contain %q:\n%s", test.args, test.stdout, res.Stdout)
			}
			if test.stderr != "" && cmdtest.StderrCaptured {
				if n := strings.Count(res.Stderr, test.stderr); n < 1 {
					t.Errorf("%v: %q is not reported:\nerror: %v\nstderr: %s", test.args, test.stderr, res.Err, res.Stderr)
				}
			}
		})
	}
}

func mustRun(t *testing.T, args ...string) {
	t.Helper()
	if res := cmdtest.Run(t, cmd.Cmd, args...); res.Err != nil {
		t.Fatalf("%v: %v\n%s", args, res.Err, res.Stderr)
	}
}

// When any of the instances fail the results are reported for each one,
// and only once, and neither the error nor the usage is written again.
func TestExitStatusReportedOnce(t *testing.T) {
	if !cmdtest.StderrCaptured {
		t.Skip("standard error is not captured on this platform")
	}

	for _, test := range []struct {
		args []string
		// failure is the text of each failure reported, which must be
		// found on the lines of standard error exactly count times
		failure string
		count   int
	}{
		{[]string{"ls", "gateway", "gw1", "nosuch"}, "nosuch", 1},
		{[]string{"ls", "--csv", "gateway", "gw1", "nosuch"}, "nosuch", 1},
		{[]string{"ps", "gateway", "gw1", "nosuch"}, "nosuch", 1},
		{[]string{"show", "gateway", "gw1", "nosuch"}, "nosuch", 1},
		{[]string{"set", "gateway", "gw1", "nosuch", "key=value"}, "nosuch", 1},
		{[]string{"logs", "gateway", "gw1", "nosuch"}, "nosuch", 1},
		{[]string{"start", "gateway", "gw1", "gw2", "nosuch"}, "gateway \"", 2},
		{[]string{"unset", "gateway", "gw1", "gw2", "-k", "nosuchkey"}, "nothing unset", 2},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			setupInstances(t)
			res := cmdtest.Run(t, cmd.Cmd, test.args...)
			if !errors.Is(res.Err, cmd.ErrReported) {
				t.Fatalf("error = %v, want the error that is already reported", res.Err)
			}

			all := res.Stdout + res.Stderr
			for _, unwanted := range []string{"Error:", "Usage:", "one or more operations failed"} {
				if strings.Contains(all, unwanted) {
					t.Errorf("output contains %q:\n%s", unwanted, all)
				}
			}

			if n := strings.Count(res.Stderr, test.failure); n != test.count {
				t.Errorf("%q is reported %d times, want %d:\n%s", test.failure, n, test.count, res.Stderr)
			}
		})
	}
}

// Adding an instance that exists is an error, with a clear message, and
// is not changed
func TestAddDeployExisting(t *testing.T) {
	setupInstances(t)

	for _, args := range [][]string{
		{"add", "gateway", "gw1"},
		{"add", "gateway", "gw1@localhost"},
		{"deploy", "gateway", "gw1"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := cmdtest.Run(t, cmd.Cmd, args...)
			if !errors.Is(res.Err, geneos.ErrExists) {
				t.Fatalf("error = %v, want an already exists error", res.Err)
			}
			if !strings.Contains(res.Err.Error(), "already exists") {
				t.Errorf("error = %q, want it to say the instance already exists", res.Err)
			}
		})
	}
}

// Creating a certificate that exists is an error, unless it is forced
func TestTLSCreateExisting(t *testing.T) {
	setupInstances(t)
	dir := t.TempDir()

	mustRun(t, "tls", "create", "-o", dir)
	res := cmdtest.Run(t, cmd.Cmd, "tls", "create", "-o", dir)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "already exists") {
		t.Fatalf("error = %v, want an already exists error", res.Err)
	}
	mustRun(t, "tls", "create", "--force", "-o", dir)
}
