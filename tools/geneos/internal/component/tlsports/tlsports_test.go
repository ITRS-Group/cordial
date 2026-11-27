package tlsports_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/tlsports"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func TestTarget(t *testing.T) {
	tests := []struct {
		name   string
		pairs  []tlsports.Pair
		port   uint16
		secure bool
		want   uint16
		wantOK bool
	}{
		{"gateway TLS on insecure port", tlsports.Gateway, 7039, true, 7038, true},
		{"gateway no TLS on TLS port", tlsports.Gateway, 7038, false, 7039, true},
		{"gateway TLS on TLS port", tlsports.Gateway, 7038, true, 0, false},
		{"gateway no TLS on insecure port", tlsports.Gateway, 7039, false, 0, false},
		{"gateway TLS on user port", tlsports.Gateway, 7100, true, 0, false},
		{"gateway no TLS on user port", tlsports.Gateway, 7100, false, 0, false},
		{"gateway TLS on a similar port", tlsports.Gateway, 17039, true, 0, false},
		{"webserver TLS on HTTP port", tlsports.Webserver, 8080, true, 8443, true},
		{"webserver no TLS on HTTPS port", tlsports.Webserver, 8443, false, 8080, true},
		{"webserver TLS on port 80", tlsports.Webserver, 80, true, 443, true},
		{"webserver no TLS on port 443", tlsports.Webserver, 443, false, 80, true},
		{"webserver TLS on HTTPS port", tlsports.Webserver, 8443, true, 0, false},
		{"webserver TLS on user port", tlsports.Webserver, 8081, true, 0, false},
		{"webserver no TLS on user port", tlsports.Webserver, 9443, false, 0, false},
		{"webserver does not pair gateway ports", tlsports.Webserver, 7039, true, 0, false},
		{"gateway does not pair webserver ports", tlsports.Gateway, 8080, true, 0, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := tlsports.Target(test.pairs, test.port, test.secure)
			if got != test.want || ok != test.wantOK {
				t.Errorf("Target(%d, %t) = %d, %t, want %d, %t", test.port, test.secure, got, ok, test.want, test.wantOK)
			}
		})
	}
}

// freePort returns a port that is not listening
func freePort(t *testing.T) uint16 {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return uint16(l.Addr().(*net.TCPAddr).Port)
}

func TestApply(t *testing.T) {
	root := geneostest.NewRoot(t, "", "licd")
	var logs bytes.Buffer
	logger.LogHandler = slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})

	certFile := filepath.Join(root, "test.pem")
	keyFile := filepath.Join(root, "test.key")
	for _, f := range []string{certFile, keyFile} {
		if err := os.WriteFile(f, []byte("placeholder"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// newInstance creates and saves an instance using port, that is TLS
	// capable if secure is true
	n := 0
	newInstance := func(t *testing.T, port uint16, secure bool) geneos.Instance {
		t.Helper()
		n++
		i, err := instance.Add(geneos.LOCAL, &licd.Component, fmt.Sprintf("apply%d", n), instance.Insecure(true))
		if err != nil {
			t.Fatal(err)
		}
		cf := i.Config()
		config.Set(cf, "port", port)
		if secure {
			config.Set(cf, cf.Join(instance.TLSBASE, instance.CERTIFICATE), certFile)
			config.Set(cf, cf.Join(instance.TLSBASE, instance.PRIVATEKEY), keyFile)
		}
		if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
			t.Fatal(resp.Err)
		}
		return i
	}
	port := func(i geneos.Instance) uint16 { return config.Get[uint16](i.Config(), "port") }

	// every subtest uses a new pair of ports so that they do not see
	// those of the instances in the others
	newPair := func(t *testing.T) (insecure, secure uint16, pairs []tlsports.Pair) {
		t.Helper()
		insecure, secure = freePort(t), freePort(t)
		for insecure == secure {
			secure = freePort(t)
		}
		return insecure, secure, []tlsports.Pair{{Insecure: insecure, Secure: secure}}
	}

	t.Run("TLS moves from the insecure to the TLS port", func(t *testing.T) {
		insecure, secure, pairs := newPair(t)
		i := newInstance(t, insecure, true)
		if !tlsports.Apply(i, pairs) || port(i) != secure {
			t.Errorf("port = %d, want %d", port(i), secure)
		}
	})

	t.Run("no TLS moves from the TLS to the insecure port", func(t *testing.T) {
		insecure, secure, pairs := newPair(t)
		i := newInstance(t, secure, false)
		if !tlsports.Apply(i, pairs) || port(i) != insecure {
			t.Errorf("port = %d, want %d", port(i), insecure)
		}
	})

	t.Run("matching ports are kept", func(t *testing.T) {
		insecure, secure, pairs := newPair(t)
		for _, secureInstance := range []bool{false, true} {
			p := insecure
			if secureInstance {
				p = secure
			}
			i := newInstance(t, p, secureInstance)
			if tlsports.Apply(i, pairs) || port(i) != p {
				t.Errorf("secure=%t: port = %d, want unchanged %d", secureInstance, port(i), p)
			}
		}
	})

	t.Run("user ports are kept", func(t *testing.T) {
		_, _, pairs := newPair(t)
		user := freePort(t)
		for _, secureInstance := range []bool{false, true} {
			i := newInstance(t, user, secureInstance)
			if tlsports.Apply(i, pairs) || port(i) != user {
				t.Errorf("secure=%t: port = %d, want unchanged %d", secureInstance, port(i), user)
			}
		}
	})

	t.Run("a port in use is replaced by the next free port", func(t *testing.T) {
		insecure, secure, pairs := newPair(t)
		logs.Reset()
		// an instance already using the TLS port
		newInstance(t, secure, true)
		i := newInstance(t, insecure, true)
		if !tlsports.Apply(i, pairs) {
			t.Fatalf("port = %d, want a new port", port(i))
		}
		if p := port(i); p == 0 || p == insecure || p == secure {
			t.Errorf("port = %d, want a free port other than %d or %d", p, insecure, secure)
		}
		if !strings.Contains(logs.String(), "using another port") {
			t.Errorf("no warning logged: %q", logs.String())
		}
	})

	// the port lookups are replaced to test the choices made
	t.Run("the range for the component is used", func(t *testing.T) {
		insecure, secure, pairs := newPair(t)
		i := newInstance(t, insecure, true)
		tlsports.Stub(t, nil,
			func(*geneos.Host) map[uint16]struct{} { return map[uint16]struct{}{secure: {}} },
			func(*geneos.Host, *geneos.Component) uint16 { return 7777 },
			func(*geneos.Host, uint16) uint16 { t.Error("fallback used"); return 0 })
		if !tlsports.Apply(i, pairs) || port(i) != 7777 {
			t.Errorf("port = %d, want 7777", port(i))
		}
	})

	t.Run("the next port above is used if the range is full", func(t *testing.T) {
		insecure, secure, pairs := newPair(t)
		i := newInstance(t, insecure, true)
		var gotStart uint16
		tlsports.Stub(t, nil,
			func(*geneos.Host) map[uint16]struct{} { return map[uint16]struct{}{secure: {}} },
			func(*geneos.Host, *geneos.Component) uint16 { return 0 },
			func(_ *geneos.Host, start uint16) uint16 { gotStart = start; return 9999 })
		if !tlsports.Apply(i, pairs) || port(i) != 9999 {
			t.Errorf("port = %d, want 9999", port(i))
		}
		if gotStart != insecure+1 {
			t.Errorf("search started at %d, want %d", gotStart, insecure+1)
		}
	})

	t.Run("the port is kept if nothing is free", func(t *testing.T) {
		insecure, secure, pairs := newPair(t)
		i := newInstance(t, insecure, true)
		tlsports.Stub(t, nil,
			func(*geneos.Host) map[uint16]struct{} { return map[uint16]struct{}{secure: {}} },
			func(*geneos.Host, *geneos.Component) uint16 { return 0 },
			func(*geneos.Host, uint16) uint16 { return 0 })
		if tlsports.Apply(i, pairs) || port(i) != insecure {
			t.Errorf("port = %d, want unchanged %d", port(i), insecure)
		}
	})

	t.Run("an instance already listening on the other port is kept", func(t *testing.T) {
		insecure, secure, pairs := newPair(t)
		i := newInstance(t, insecure, true)
		tlsports.Stub(t,
			func(_ geneos.Instance, p uint16) (bool, error) { return p == secure, nil },
			func(*geneos.Host) map[uint16]struct{} { return map[uint16]struct{}{secure: {}} },
			func(*geneos.Host, *geneos.Component) uint16 { t.Error("allocated a port"); return 0 }, nil)
		if tlsports.Apply(i, pairs) || port(i) != insecure {
			t.Errorf("port = %d, want unchanged %d", port(i), insecure)
		}
	})
}
