package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"

	_ "github.com/itrs-group/cordial/tools/geneos/cmd/cfgcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/hostcmd"
)

// The legacy commands must hand all of their arguments and flags to the
// commands that replace them.
func TestLegacyAliasesPassArguments(t *testing.T) {
	t.Run("add remote", func(t *testing.T) {
		cmdtest.Env(t)
		// the scheme check is made after the arguments are parsed
		res := cmdtest.Run(t, cmd.Cmd, "add", "remote", "bar", "ftp://example.com/geneos")
		if res.Err == nil || !strings.Contains(res.Err.Error(), "unsupported scheme") {
			t.Fatalf("error = %v, want unsupported scheme error", res.Err)
		}
	})

	t.Run("delete host", func(t *testing.T) {
		cmdtest.Env(t)
		cmdtest.AddHost(t, "bar", "hostname", "bar.example.com")
		res := cmdtest.Run(t, cmd.Cmd, "delete", "host", "bar")
		if res.Err != nil || !strings.Contains(res.Stdout, `"bar" deleted`) {
			t.Fatalf("error = %v, stdout = %q", res.Err, res.Stdout)
		}
		res = cmdtest.Run(t, cmd.Cmd, "host", "ls")
		if res.Err != nil || strings.Contains(res.Stdout, "bar") {
			t.Fatalf("error = %v, stdout = %q, host not deleted", res.Err, res.Stdout)
		}
	})

	t.Run("show host", func(t *testing.T) {
		cmdtest.Env(t)
		cmdtest.AddHost(t, "bar", "hostname", "bar.example.com")
		res := cmdtest.Run(t, cmd.Cmd, "show", "host", "bar")
		if res.Err != nil || !strings.Contains(res.Stdout, `"name": "bar"`) {
			t.Fatalf("error = %v, stdout = %q", res.Err, res.Stdout)
		}
	})

	t.Run("set user", func(t *testing.T) {
		cmdtest.Env(t)
		res := cmdtest.Run(t, cmd.Cmd, "set", "user", "foo=bar")
		if res.Err != nil {
			t.Fatalf("error = %v, stdout = %q", res.Err, res.Stdout)
		}
		res = cmdtest.Run(t, cmd.Cmd, "config", "show", "foo")
		if res.Err != nil || !strings.Contains(res.Stdout, `"foo": "bar"`) {
			t.Fatalf("error = %v, stdout = %q", res.Err, res.Stdout)
		}
	})
}

func TestConfigHonoursConfigFileFlag(t *testing.T) {
	cmdtest.Env(t)
	file := filepath.Join(t.TempDir(), "other.json")

	if res := cmdtest.Run(t, cmd.Cmd, "config", "set", "foo=bar", "-G", file); res.Err != nil {
		t.Fatalf("set: %v", res.Err)
	}
	b, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(b), `"foo"`) {
		t.Fatalf("%s = %q, %v; want the setting", file, b, err)
	}
	res := cmdtest.Run(t, cmd.Cmd, "config", "show", "foo", "-G", file)
	if res.Err != nil || !strings.Contains(res.Stdout, `"foo": "bar"`) {
		t.Fatalf("show: error = %v, stdout = %q", res.Err, res.Stdout)
	}
	if res := cmdtest.Run(t, cmd.Cmd, "config", "unset", "foo", "-G", file); res.Err != nil {
		t.Fatalf("unset: %v", res.Err)
	}
	if b, _ := os.ReadFile(file); strings.Contains(string(b), `"foo"`) {
		t.Fatalf("%s = %q, setting not removed", file, b)
	}
}

func TestHostCommandErrors(t *testing.T) {
	cmdtest.Env(t)
	cmdtest.AddHost(t, "other", "hostname", "other.example.com", "username", "orig")
	cmdtest.AddHost(t, "bar", "hostname", "bar.example.com", "username", "orig")

	if res := cmdtest.Run(t, cmd.Cmd, "host", "delete", "nosuchhost"); res.Err == nil {
		t.Errorf("host delete of an unknown host succeeded, stdout = %q", res.Stdout)
	}
	if res := cmdtest.Run(t, cmd.Cmd, "host", "show", "nosuchhost"); res.Err == nil {
		t.Errorf("host show of an unknown host succeeded, stdout = %q", res.Stdout)
	}

	// unset with a name must only change the named host
	if res := cmdtest.Run(t, cmd.Cmd, "host", "unset", "bar", "-k", "username"); res.Err != nil {
		t.Fatalf("host unset: %v", res.Err)
	}
	res := cmdtest.Run(t, cmd.Cmd, "host", "show", "bar", "other")
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if n := strings.Count(res.Stdout, `"username": "orig"`); n != 1 {
		t.Errorf("host unset changed the wrong hosts, %d hosts have a username: %s", n, res.Stdout)
	}
}
