package cmd_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"
)

const snapshotCommandTestXPath = `//dataview[(@name="test")]`

func snapshotCommandServer(t *testing.T, username, password string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy/gatewayinfo/timezone", func(w http.ResponseWriter, r *http.Request) {
		assertSnapshotAuth(t, r, username, password)
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/proxy/xpaths/match", func(w http.ResponseWriter, r *http.Request) {
		assertSnapshotAuth(t, r, username, password)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"xpaths": []string{snapshotCommandTestXPath},
		})
	})
	return httptest.NewServer(mux)
}

func assertSnapshotAuth(t *testing.T, r *http.Request, username, password string) {
	t.Helper()
	gotUser, gotPassword, ok := r.BasicAuth()
	if !ok || gotUser != username || gotPassword != password {
		t.Errorf("BasicAuth = %q, %q, %v; want %q, %q", gotUser, gotPassword, ok, username, password)
	}
}

func TestSnapshotPasswordFlagOverridesGlobalConfig(t *testing.T) {
	cmdtest.Env(t)

	cf := config.Global()
	config.Set(cf, cf.Join("snapshot", "password"), "config-password")
	t.Cleanup(func() { config.Delete(cf, cf.Join("snapshot", "password")) })

	values := config.NewKeyValues()
	t.Cleanup(values.Destroy)
	ciphertext, err := values.EncodeString("flag-password")
	if err != nil {
		t.Fatal(err)
	}
	keyfile := filepath.Join(t.TempDir(), "snapshot-flag.aes")
	if err := os.WriteFile(keyfile, []byte(values.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, password := range map[string]string{
		"plaintext": "flag-password",
		"encrypted": "${enc:" + keyfile + ":+encs+" + ciphertext + "}",
	} {
		t.Run(name, func(t *testing.T) {
			server := snapshotCommandServer(t, "alice", "flag-password")
			t.Cleanup(server.Close)
			res := cmdtest.Run(t, cmd.Cmd,
				"snapshot",
				"--connect", server.URL+"/proxy",
				"--username", "alice",
				"--password", password,
				"--xpaths",
				snapshotCommandTestXPath,
			)
			if res.Err != nil {
				t.Fatalf("snapshot --password: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
			}
		})
	}
}

func TestSnapshotCredentialsFromEnvironmentExpandPassword(t *testing.T) {
	cmdtest.Env(t)
	server := snapshotCommandServer(t, "env-user", "environment-password")
	t.Cleanup(server.Close)

	values := config.NewKeyValues()
	t.Cleanup(values.Destroy)
	ciphertext, err := values.EncodeString("environment-password")
	if err != nil {
		t.Fatal(err)
	}
	keyfile := filepath.Join(t.TempDir(), "snapshot.aes")
	if err := os.WriteFile(keyfile, []byte(values.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ITRS_SNAPSHOT_USERNAME", "env-user")
	t.Setenv("ITRS_SNAPSHOT_PASSWORD", "${enc:"+keyfile+":+encs+"+ciphertext+"}")

	res := cmdtest.Run(t, cmd.Cmd,
		"snapshot",
		"--connect", server.URL+"/proxy",
		"--xpaths",
		snapshotCommandTestXPath,
	)
	if res.Err != nil {
		t.Fatalf("snapshot with environment credentials: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
}

func TestSnapshotRejectsMultipleGatewayConnections(t *testing.T) {
	addGateways(t, "snapshot-one", "snapshot-two")
	res := cmdtest.Run(t, cmd.Cmd,
		"snapshot", "gateway", "all", "--xpaths", snapshotCommandTestXPath,
	)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "at most one") {
		t.Fatalf("error = %v, want at-most-one Gateway error", res.Err)
	}
}

func TestSnapshotRejectsRepeatedConnectFlag(t *testing.T) {
	cmdtest.Env(t)
	res := cmdtest.Run(t, cmd.Cmd,
		"snapshot",
		"--connect", "http://gateway-one.example/rest",
		"--connect", "http://gateway-two.example/rest",
		"--xpaths", snapshotCommandTestXPath,
	)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "only be specified once") {
		t.Fatalf("error = %v, want repeated --connect error", res.Err)
	}
}
