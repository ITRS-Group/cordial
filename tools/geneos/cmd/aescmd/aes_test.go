package aescmd

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
)

var sharedKeyfileRE = regexp.MustCompile(`([0-9A-F]{8})\.aes`)

func TestSetCRCUsesSharedKeyfile(t *testing.T) {
	cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "gw1"); res.Err != nil {
		t.Fatalf("add: %v", res.Err)
	}

	res := cmdtest.Run(t, cmd.Cmd, "aes", "new", "--shared", "gateway")
	if res.Err != nil {
		t.Fatalf("aes new --shared: %v", res.Err)
	}
	m := sharedKeyfileRE.FindStringSubmatch(res.Stdout)
	if m == nil {
		t.Fatalf("no shared keyfile in output: %s", res.Stdout)
	}
	crc := m[1]

	t.Run("list shared json", func(t *testing.T) {
		res := cmdtest.Run(t, cmd.Cmd, "aes", "list", "--shared", "--json", "gateway")
		if res.Err != nil {
			t.Fatal(res.Err)
		}
		if !strings.Contains(res.Stdout, crc) {
			t.Errorf("shared keyfile %s not listed: %s", crc, res.Stdout)
		}
		if !json.Valid([]byte(res.Stdout)) {
			t.Errorf("output is not JSON: %s", res.Stdout)
		}
	})

	t.Run("existing crc", func(t *testing.T) {
		// the keyfile flag must not stop the CRC from being used
		res := cmdtest.Run(t, cmd.Cmd, "aes", "set", "--crc", crc, "gateway", "gw1")
		if res.Err != nil {
			t.Fatalf("aes set --crc: %v\n%s", res.Err, res.Stderr)
		}
		res = cmdtest.Run(t, cmd.Cmd, "aes", "list", "gateway", "gw1")
		if res.Err != nil || !strings.Contains(res.Stdout, filepath.Join("keyfiles", crc+".aes")) {
			t.Errorf("instance does not use the shared keyfile: %v\n%s", res.Err, res.Stdout)
		}
	})

	t.Run("missing crc", func(t *testing.T) {
		if !cmdtest.StderrCaptured {
			t.Skip("standard error is not captured on this platform")
		}
		res := cmdtest.Run(t, cmd.Cmd, "aes", "set", "--crc", "00000000", "gateway", "gw1")
		if !strings.Contains(res.Stderr+res.Stdout, "not found") {
			t.Errorf("no error for a missing shared keyfile: %q %q", res.Stdout, res.Stderr)
		}
	})

	t.Run("exclusive options", func(t *testing.T) {
		res := cmdtest.Run(t, cmd.Cmd, "aes", "set", "--crc", crc, "-k", "file", "gateway", "gw1")
		if res.Err == nil {
			t.Error("--crc and --keyfile were accepted together")
		}
	})
}
