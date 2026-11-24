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
	mux.HandleFunc("/proxy/snapshot/dataview", func(w http.ResponseWriter, r *http.Request) {
		assertSnapshotAuth(t, r, username, password)
		_, _ = io.WriteString(w, `{"status":"ok","dataview":{"sample-time":"2026-11-24T02:00:00Z","headlines":{},"table":{"row":{"value":{"value":"line one\nline two\rC:\\temp\tbad\u0000value"}}}}}`)
	})
	return httptest.NewServer(mux)
}

func TestSnapshotToolkitEscapesControlCharacters(t *testing.T) {
	cmdtest.Env(t)
	server := snapshotCommandServer(t, "", "")
	t.Cleanup(server.Close)

	for _, format := range []string{"toolkit", "toolkit-multi"} {
		t.Run(format, func(t *testing.T) {
			res := cmdtest.Run(t, cmd.Cmd,
				"snapshot",
				"--connect", server.URL+"/proxy",
				"--format", format,
				snapshotCommandTestXPath,
			)
			if res.Err != nil {
				t.Fatalf("snapshot --format %s: %v\n%s%s", format, res.Err, res.Stdout, res.Stderr)
			}
			if strings.Contains(res.Stdout, "\r") || strings.Contains(res.Stdout, "\u0000") {
				t.Errorf("output contains raw control characters: %q", res.Stdout)
			}
			if want := `line one\nline two\rC:\\temp\tbadvalue`; !strings.Contains(res.Stdout, want) {
				t.Errorf("output does not contain escaped value %q:\n%s", want, res.Stdout)
			}
			if !strings.Contains(res.Stdout, "<!>samplingStatus,OK") {
				t.Errorf("output does not report successful sampling:\n%s", res.Stdout)
			}
			if got := strings.Count(res.Stdout, "<!>samplingStatus,"); got != 1 {
				t.Errorf("output contains %d samplingStatus headlines, want 1:\n%s", got, res.Stdout)
			}
		})
	}
}

func TestSnapshotToolkitReportsCommandError(t *testing.T) {
	cmdtest.Env(t)
	res := cmdtest.Run(t, cmd.Cmd,
		"snapshot",
		"--connect", "ftp://gateway.example",
		"--format", "toolkit",
		"--xpaths",
		snapshotCommandTestXPath,
	)
	if res.Err == nil {
		t.Fatal("snapshot unexpectedly succeeded")
	}
	for _, want := range []string{"<!>samplingStatus,", "not supported"} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("output does not contain %q:\n%s", want, res.Stdout)
		}
	}
}

func TestSnapshotToolkitReportsFlagError(t *testing.T) {
	cmdtest.Env(t)
	res := cmdtest.Run(t, cmd.Cmd,
		"snapshot",
		"--format", "toolkit",
		"--limit", "not-a-number",
		"--xpaths",
		snapshotCommandTestXPath,
	)
	if res.Err == nil {
		t.Fatal("snapshot unexpectedly succeeded")
	}
	if !strings.Contains(res.Stdout, "<!>samplingStatus,invalid argument") {
		t.Errorf("output does not report flag error:\n%s", res.Stdout)
	}
}

func TestSnapshotForwardsUnsupportedXPathPredicates(t *testing.T) {
	cmdtest.Env(t)
	const predicate = `[(param("Group")="Gateway Info")]`
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/gatewayinfo/timezone", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/rest/xpaths/match", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Target string `json:"target"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if !strings.Contains(request.Target, predicate) {
			t.Errorf("REST target %q does not contain %q", request.Target, predicate)
		}
		_, _ = io.WriteString(w, `{"status":"ok","xpaths":[]}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	res := cmdtest.Run(t, cmd.Cmd,
		"snapshot",
		"--connect", server.URL,
		"--xpaths",
		"//dataview"+predicate,
	)
	if res.Err != nil {
		t.Fatalf("snapshot with predicate: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
}

func assertSnapshotAuth(t *testing.T, r *http.Request, username, password string) {
	t.Helper()
	if username == "" {
		return
	}
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
