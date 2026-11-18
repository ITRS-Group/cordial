package email

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wneessen/go-mail"

	"github.com/itrs-group/cordial/pkg/config"
)

func newConf(t *testing.T, kv ...string) *config.Config {
	t.Helper()
	conf := config.New()
	for i := 0; i < len(kv); i += 2 {
		config.Set(conf, kv[i], kv[i+1])
	}
	return conf
}

// unexported reads an unexported field of the client
func unexported(t *testing.T, c *mail.Client, name string) reflect.Value {
	t.Helper()
	v := reflect.ValueOf(c).Elem().FieldByName(name)
	if !v.IsValid() {
		t.Fatalf("go-mail Client has no field %q", name)
	}
	return v
}

func TestDialOptions(t *testing.T) {
	tests := []struct {
		name         string
		kv           []string
		addr         string
		fallbackPort int64
		ssl          bool
		auth         string
		err          bool
	}{
		{"defaults", nil, "localhost:587", 25, false, "NOAUTH", false},
		{"custom port has no fallback to port 25", []string{"_smtp_port", "2525"}, "localhost:2525", 0, false, "NOAUTH", false},
		{"port 25 is not changed by the policy", []string{"_smtp_port", "25"}, "localhost:25", 0, false, "NOAUTH", false},
		{"force with port", []string{"_smtp_tls", "force", "_smtp_port", "2525"}, "localhost:2525", 0, false, "NOAUTH", false},
		{"none", []string{"_smtp_tls", "none"}, "localhost:25", 0, false, "NOAUTH", false},
		{"ssl", []string{"_smtp_tls", "SSL"}, "localhost:465", 0, true, "NOAUTH", false},
		{"ssl custom port", []string{"_smtp_tls", "ssl", "_smtp_port", "2465"}, "localhost:2465", 0, true, "NOAUTH", false},
		{"port 465 is implicit tls", []string{"_smtp_port", "465"}, "localhost:465", 0, true, "NOAUTH", false},
		{"port 465 with tls none is not", []string{"_smtp_port", "465", "_smtp_tls", "none"}, "localhost:465", 0, false, "NOAUTH", false},
		{"server", []string{"_smtp_server", "mail.example.com"}, "mail.example.com:587", 25, false, "NOAUTH", false},
		{"empty server uses default", []string{"_smtp_server", ""}, "localhost:587", 25, false, "NOAUTH", false},
		{"credentials discover the mechanism", []string{"_smtp_username", "u", "_smtp_password", "p"}, "localhost:587", 25, false, "AUTODISCOVER", false},
		{"named mechanism", []string{"_smtp_username", "u", "_smtp_auth", "plain"}, "localhost:587", 25, false, "PLAIN", false},
		{"auto mechanism", []string{"_smtp_username", "u", "_smtp_auth", "auto"}, "localhost:587", 25, false, "AUTODISCOVER", false},
		{"unknown mechanism", []string{"_smtp_username", "u", "_smtp_auth", "nonsense"}, "", 0, false, "", true},
		{"invalid port", []string{"_smtp_port", "70000"}, "", 0, false, "", false}, // reported by mail.NewClient
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, opts, err := dialOptions(newConf(t, tt.kv...))
			if tt.err {
				if err == nil {
					t.Fatal("dialOptions() error = nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			c, err := mail.NewClient(server, opts...)
			if tt.name == "invalid port" {
				if err == nil {
					t.Error("NewClient() accepted an invalid port")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := c.ServerAddr(); got != tt.addr {
				t.Errorf("address = %q, want %q", got, tt.addr)
			}
			if got := unexported(t, c, "fallbackPort").Int(); got != tt.fallbackPort {
				t.Errorf("fallback port = %d, want %d", got, tt.fallbackPort)
			}
			if got := unexported(t, c, "useSSL").Bool(); got != tt.ssl {
				t.Errorf("implicit TLS = %v, want %v", got, tt.ssl)
			}
			if got := unexported(t, c, "smtpAuthType").String(); got != tt.auth {
				t.Errorf("auth = %q, want %q", got, tt.auth)
			}
		})
	}
}

func TestNewEmailConfigDoesNotMaskDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// the environment is copied into the config, so make sure it has none of these
	for _, key := range []string{"_smtp_server", "_SMTP_SERVER", "_from", "_FROM", "_smtp_tls", "_SMTP_TLS"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	em := NewEmailConfig(config.New(), "", "", "", "")
	for _, key := range []string{"_smtp_server", "_from", "_smtp_tls"} {
		if got := config.Get[string](em, key, config.DefaultValue("sentinel")); got != "sentinel" {
			t.Errorf("%s = %q, an empty default masks the default value", key, got)
		}
	}

	server, _, err := dialOptions(em)
	if err != nil || server != "localhost" {
		t.Errorf("server = %q, err = %v, want localhost", server, err)
	}
	m, err := UpdateEnvelope(em, false)
	if err != nil {
		t.Fatal(err)
	}
	if from := m.GetFromString(); len(from) != 1 || !strings.Contains(from[0], "geneos@localhost") {
		t.Errorf("from = %v, want geneos@localhost", from)
	}
}

func TestUpdateEnvelopeInvalidFrom(t *testing.T) {
	if _, err := UpdateEnvelope(newConf(t, "_from", "not an address"), false); err == nil {
		t.Error("UpdateEnvelope() accepted an invalid _from address")
	}
	m, err := UpdateEnvelope(newConf(t, "_from", "a@example.com", "_to", "b@example.com"), false)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.GetFromString(); len(got) != 1 || !strings.Contains(got[0], "a@example.com") {
		t.Errorf("from = %v", got)
	}
}

// fakeSMTP accepts connections on ln and speaks just enough SMTP for a
// client to connect, say EHLO and quit. It returns after the first
// connection has finished, on done
func fakeSMTP(t *testing.T, ln net.Listener) (done chan string) {
	t.Helper()
	done = make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- "accept: " + err.Error()
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		_, _ = conn.Write([]byte("220 fake ESMTP\r\n"))
		r := bufio.NewReader(conn)
		var seen []string
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				break
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			seen = append(seen, cmd)
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				_, _ = conn.Write([]byte("250 fake\r\n"))
			case strings.HasPrefix(cmd, "QUIT"):
				_, _ = conn.Write([]byte("221 bye\r\n"))
				done <- strings.Join(seen, ",")
				return
			default:
				_, _ = conn.Write([]byte("250 ok\r\n"))
			}
		}
		done <- strings.Join(seen, ",")
	}()
	return
}

func dialFake(t *testing.T, ln net.Listener, kv ...string) {
	t.Helper()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	conf := newConf(t, append([]string{"_smtp_server", "127.0.0.1", "_smtp_port", port, "_smtp_timeout", "5"}, kv...)...)
	done := fakeSMTP(t, ln)

	d, err := Dial(conf)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.DialWithContext(ctx); err != nil {
		t.Fatalf("DialWithContext() error = %v", err)
	}
	if err := d.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case seen := <-done:
		if !strings.Contains(seen, "EHLO") {
			t.Errorf("server saw %q, want an EHLO", seen)
		}
	case <-time.After(5 * time.Second):
		t.Error("timed out waiting for the fake server")
	}
}

func TestDialCustomPortPlain(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen: %v", err)
	}
	defer ln.Close()
	dialFake(t, ln, "_smtp_tls", "none")
}

func TestDialImplicitTLS(t *testing.T) {
	cert := httptest.NewUnstartedServer(nil)
	cert.StartTLS()
	defer cert.Close()

	raw, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen: %v", err)
	}
	ln := tls.NewListener(raw, cert.TLS)
	defer ln.Close()
	dialFake(t, ln, "_smtp_tls", "ssl", "_smtp_tls_insecure", "true")
}

// a failure to connect to a custom port must be reported, not retried
// on another port
func TestDialCustomPortFailureIsReported(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen: %v", err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close()
	if _, err := strconv.Atoi(port); err != nil {
		t.Fatal(err)
	}

	d, err := Dial(newConf(t, "_smtp_server", "127.0.0.1", "_smtp_port", port, "_smtp_timeout", "2"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.DialWithContext(ctx); err == nil {
		d.Close()
		t.Fatal("DialWithContext() succeeded with nothing listening")
	}
}
