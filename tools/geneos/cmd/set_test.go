package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
)

func TestSetDoesNotAuditSecrets(t *testing.T) {
	root := cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "gw1"); res.Err != nil {
		t.Fatalf("add: %v", res.Err)
	}
	args := []string{"set", "gateway", "gw1",
		"-s", "param1=SUPERSECRETPARAM",
		"-E", "ENV1=SUPERSECRETENV",
		"-v", "secret:var1=SUPERSECRETVAR",
	}
	// the audited command line comes from os.Args
	previous := os.Args
	os.Args = append([]string{"geneos"}, args...)
	t.Cleanup(func() { os.Args = previous })
	res := cmdtest.Run(t, cmd.Cmd, args...)
	if res.Err != nil {
		t.Fatalf("set: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}

	audit, err := os.ReadFile(filepath.Join(root, "audit", "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(audit), `"action":"set"`) {
		t.Fatalf("no set action in the audit log: %s", audit)
	}
	for _, secret := range []string{"SUPERSECRETPARAM", "SUPERSECRETENV", "SUPERSECRETVAR"} {
		if strings.Contains(string(audit), secret) {
			t.Errorf("audit log contains secret %q:\n%s", secret, audit)
		}
	}
}
