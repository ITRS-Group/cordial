package tlscmd

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/webserver"
)

func TestTLSInitDefaultKeyType(t *testing.T) {
	cmdtest.Env(t)

	if res := cmdtest.Run(t, cmd.Cmd, "tls", "init"); res.Err != nil {
		t.Fatalf("tls init without -K: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
	if _, _, err := geneos.ReadRootCertificateAndKey(); err != nil {
		t.Fatalf("no root certificate after tls init: %v", err)
	}
}

func TestTLSCreateForceDefaultKeyType(t *testing.T) {
	cmdtest.Env(t)

	dir := t.TempDir()
	res := cmdtest.Run(t, cmd.Cmd, "tls", "create", "--force", "-o", dir)
	if res.Err != nil {
		t.Fatalf("tls create --force: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, geneos.LOCALHOST+".pem")); err != nil {
		t.Fatalf("certificate not created: %v", err)
	}
}

func TestTLSCreateReportsWriteErrors(t *testing.T) {
	cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "tls", "init"); res.Err != nil {
		t.Fatalf("tls init: %v", res.Err)
	}

	missing := filepath.Join(t.TempDir(), "missing", "dir")
	for _, args := range [][]string{
		{"tls", "create", "-o", missing},
		{"tls", "create", "-S", "host1", "-o", missing},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := cmdtest.Run(t, cmd.Cmd, args...)
			if res.Err == nil {
				t.Fatalf("command succeeded writing to a missing directory: %s", res.Stdout)
			}
			if strings.Contains(res.Stdout, "created in") {
				t.Errorf("reported creation: %s", res.Stdout)
			}
		})
	}
}

func TestTLSListRemaining(t *testing.T) {
	cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "tls", "init"); res.Err != nil {
		t.Fatalf("tls init: %v", res.Err)
	}
	if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "gw1"); res.Err != nil {
		t.Fatalf("add: %v\n%s", res.Err, res.Stderr)
	}

	t.Run("json", func(t *testing.T) {
		res := cmdtest.Run(t, cmd.Cmd, "tls", "list", "-a", "-l", "--json")
		if res.Err != nil {
			t.Fatalf("tls list: %v", res.Err)
		}
		var results []struct {
			Type        string        `json:"type"`
			Name        string        `json:"name"`
			Remaining   time.Duration `json:"remaining"`
			Fingerprint string        `json:"fingerprint"`
		}
		if err := json.Unmarshal([]byte(res.Stdout), &results); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, res.Stdout)
		}

		signing, _, err := geneos.ReadSigningCertificateAndKey()
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]bool{}
		for _, r := range results {
			switch {
			case r.Type == cordial.ExecutableName() && r.Name == geneos.SigningCertBasename:
				found["signing"] = true
				if want := time.Until(signing.NotAfter); r.Remaining < want-time.Minute || r.Remaining > want+time.Minute {
					t.Errorf("signing remaining = %v, want about %v", r.Remaining, want)
				}
				if want := fmt.Sprintf("%X", sha1.Sum(signing.Raw)); r.Fingerprint != want {
					t.Errorf("signing fingerprint = %s, want %s", r.Fingerprint, want)
				}
			case r.Name == "gw1":
				// the instance certificate is valid for a year
				found["instance"] = true
				if r.Remaining < 300*24*time.Hour || r.Remaining > 366*24*time.Hour {
					t.Errorf("instance remaining = %v, want about a year", r.Remaining)
				}
			}
		}
		if !found["signing"] || !found["instance"] {
			t.Errorf("missing signing or instance certificates in %s", res.Stdout)
		}
	})

	t.Run("toolkit headlines", func(t *testing.T) {
		if res := cmdtest.Run(t, cmd.Cmd, "tls", "renew", "gateway", "gw1", "-E", "100"); res.Err != nil {
			t.Fatalf("renew: %v", res.Err)
		}
		res := cmdtest.Run(t, cmd.Cmd, "tls", "list", "--toolkit", "gateway", "gw1")
		if res.Err != nil {
			t.Fatalf("tls list: %v", res.Err)
		}
		// 100 days is not expiring within 30 days
		for _, want := range []string{"<!>totalCerts,1", "<!>expiringLT7Days,0", "<!>expiringLT30Days,0"} {
			if !strings.Contains(res.Stdout, want) {
				t.Errorf("output does not contain %q:\n%s", want, res.Stdout)
			}
		}
	})
}

func TestTLSExportRejectsOutputForManyInstances(t *testing.T) {
	cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "tls", "init"); res.Err != nil {
		t.Fatalf("tls init: %v", res.Err)
	}
	for _, name := range []string{"gw1", "gw2"} {
		if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", name); res.Err != nil {
			t.Fatalf("add: %v", res.Err)
		}
	}
	out := filepath.Join(t.TempDir(), "out.pem")
	res := cmdtest.Run(t, cmd.Cmd, "tls", "export", "gateway", "-o", out)
	if res.Err == nil {
		t.Fatal("export of several instances to one file succeeded")
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("output file was written")
	}
}

func TestTLSInfoReportsFileErrors(t *testing.T) {
	cmdtest.Env(t)
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(bad, []byte("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.pem")

	// the files with errors are listed, and fail the command
	res := cmdtest.Run(t, cmd.Cmd, "tls", "info", bad, missing)
	if !errors.Is(res.Err, cmd.ErrReported) {
		t.Fatalf("tls info error = %v, want the reported error", res.Err)
	}
	if n := strings.Count(res.Stdout, "ERROR"); n != 2 {
		t.Errorf("found %d errors in the output, want one for each file:\n%s", n, res.Stdout)
	}
}

func TestTLSInfoLeafOnlyNeedsPrivateKey(t *testing.T) {
	cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "tls", "init"); res.Err != nil {
		t.Fatalf("tls init: %v", res.Err)
	}
	dir := t.TempDir()
	withKey := filepath.Join(dir, "withkey.pem")
	if res := cmdtest.Run(t, cmd.Cmd, "tls", "create", "-c", "leaf1", "-o", dir); res.Err != nil {
		t.Fatalf("tls create: %v", res.Err)
	}
	if err := os.Rename(filepath.Join(dir, "leaf1.pem"), withKey); err != nil {
		t.Fatal(err)
	}

	// the same leaf certificate, without its private key
	b, err := os.ReadFile(withKey)
	if err != nil {
		t.Fatal(err)
	}
	var certsOnly strings.Builder
	for _, block := range strings.SplitAfter(string(b), "-----END ") {
		if strings.Contains(block, "CERTIFICATE-----") && !strings.Contains(block, "PRIVATE KEY") {
			certsOnly.WriteString(block)
			certsOnly.WriteString("CERTIFICATE-----\n")
		}
	}
	noKey := filepath.Join(dir, "nokey.pem")
	if err := os.WriteFile(noKey, []byte(certsOnly.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	res := cmdtest.Run(t, cmd.Cmd, "tls", "info", "-L", noKey)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if strings.Contains(res.Stdout, "leaf1") {
		t.Errorf("certificate without a private key listed with --leaf-only:\n%s", res.Stdout)
	}
	res = cmdtest.Run(t, cmd.Cmd, "tls", "info", "-L", withKey)
	if res.Err != nil || !strings.Contains(res.Stdout, "withkey") {
		t.Errorf("leaf with key not listed: %v\n%s", res.Err, res.Stdout)
	}
}

func TestTLSInfoConnectTimesOut(t *testing.T) {
	cmdtest.Env(t)
	previous := infoDialTimeout
	infoDialTimeout = 200 * time.Millisecond
	t.Cleanup(func() { infoDialTimeout = previous })

	// accept connections but never answer the TLS handshake
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen: %v", err)
	}
	done := make(chan struct{})
	t.Cleanup(func() {
		l.Close()
		<-done
	})
	go func() {
		defer close(done)
		var conns []net.Conn
		defer func() {
			for _, c := range conns {
				c.Close()
			}
		}()
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			conns = append(conns, c)
		}
	}()

	start := time.Now()
	res := cmdtest.Run(t, cmd.Cmd, "tls", "info", "-c", l.Addr().String())
	if !errors.Is(res.Err, cmd.ErrReported) {
		t.Fatalf("tls info error = %v, want the reported error", res.Err)
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("connection attempt took %v", time.Since(start))
	}
	if !strings.Contains(res.Stdout, "ERROR") {
		t.Errorf("no error reported for a connection that timed out:\n%s", res.Stdout)
	}
}

// a webserver without its security.properties file must produce an
// error for the instance, which fails the command, not a nil pointer
// panic
func TestTLSMigrateWebserverWithoutProperties(t *testing.T) {
	root := cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "tls", "init"); res.Err != nil {
		t.Fatalf("tls init: %v", res.Err)
	}
	// the webserver needs an installed package, but not a real one, with
	// the configuration files that are copied from it to the instance
	pkgConfig := filepath.Join(root, "packages", "webserver", "active_prod", "config")
	if err := os.MkdirAll(pkgConfig, 0o775); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"config.xml.min.tmpl": "<config/>\n", "security.properties": "# none\n"} {
		if err := os.WriteFile(filepath.Join(pkgConfig, name), []byte(content), 0o664); err != nil {
			t.Fatal(err)
		}
	}
	if res := cmdtest.Run(t, cmd.Cmd, "add", "webserver", "ws1", "--insecure"); res.Err != nil {
		t.Fatalf("add: %v\n%s", res.Err, res.Stderr)
	}
	home := filepath.Join(root, "webserver", "webservers", "ws1")
	_ = os.Remove(filepath.Join(home, "config", "security.properties"))

	for _, command := range []string{"migrate", "renew"} {
		t.Run(command, func(t *testing.T) {
			res := cmdtest.Run(t, cmd.Cmd, "tls", command, "webserver", "ws1")
			if !errors.Is(res.Err, cmd.ErrReported) {
				t.Fatalf("tls %s error = %v, want the reported error", command, res.Err)
			}
		})
	}
}
