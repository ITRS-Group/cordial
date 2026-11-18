package cmd_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
)

func TestAddFailures(t *testing.T) {
	cmdtest.Env(t)

	if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "gw1"); res.Err != nil {
		t.Fatalf("add: %v", res.Err)
	}

	t.Run("existing instance", func(t *testing.T) {
		res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "gw1")
		if !errors.Is(res.Err, geneos.ErrExists) {
			t.Fatalf("error = %v, want an already exists error", res.Err)
		}
	})

	t.Run("unknown host in name", func(t *testing.T) {
		res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "gw2@typo")
		if res.Err == nil || !strings.Contains(res.Err.Error(), "unknown host") {
			t.Fatalf("error = %v, want an unknown host error", res.Err)
		}
	})

	t.Run("deploy unknown host in name", func(t *testing.T) {
		res := cmdtest.Run(t, cmd.Cmd, "deploy", "gateway", "gw3@typo")
		if res.Err == nil || !strings.Contains(res.Err.Error(), "unknown host") {
			t.Fatalf("error = %v, want an unknown host error", res.Err)
		}
	})
}
