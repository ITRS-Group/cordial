package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	m.Run()
}

// smtpServer is a fake SMTP server that keeps the messages it is sent
type smtpServer struct {
	addr     string
	mu       sync.Mutex
	messages []string
}

func newSMTPServer(t *testing.T) *smtpServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &smtpServer{addr: ln.Addr().String()}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(conn)
		}
	}()
	return s
}

func (s *smtpServer) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	fmt.Fprint(conn, "220 test\r\n")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			fmt.Fprint(conn, "250 test\r\n")
		case strings.HasPrefix(cmd, "DATA"):
			fmt.Fprint(conn, "354 go\r\n")
			var data strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				data.WriteString(l)
			}
			s.mu.Lock()
			s.messages = append(s.messages, data.String())
			s.mu.Unlock()
			fmt.Fprint(conn, "250 queued\r\n")
		case strings.HasPrefix(cmd, "QUIT"):
			fmt.Fprint(conn, "221 bye\r\n")
			return
		default:
			fmt.Fprint(conn, "250 ok\r\n")
		}
	}
}

func (s *smtpServer) sent() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.messages...)
}

func (s *smtpServer) args(extra ...string) []string {
	host, port, _ := net.SplitHostPort(s.addr)
	return append([]string{
		"_SMTP_SERVER=" + host, "_SMTP_PORT=" + port, "_SMTP_TLS=none",
		"_TO=ops@example.com", "_FROM=geneos@example.com",
	}, extra...)
}

func TestParseArgs(t *testing.T) {
	conf, values := parseArgs([]string{"_TO=a@example.com", "_FLAG", "_EQUALS=a=b", "_Mixed=Case"})
	if conf == nil {
		t.Fatal("parseArgs() returned no configuration")
	}
	if got := config.Get[string](conf, "_TO"); got != "a@example.com" || !conf.IsSet("_FLAG") || config.Get[string](conf, "_EQUALS") != "a=b" {
		t.Errorf("configuration has the wrong values: %q", got)
	}
	want := map[string]string{"_TO": "a@example.com", "_FLAG": "", "_EQUALS": "a=b", "_Mixed": "Case"}
	for k, v := range want {
		if got, ok := values[k]; !ok || got != v {
			t.Errorf("values[%q] = %q, %t, want %q", k, got, ok, v)
		}
	}
	if conf, values := parseArgs(nil); conf == nil || len(values) != 0 {
		t.Error("no arguments must still be a configuration")
	}
}

func TestProtectReturnsFailureOnPanic(t *testing.T) {
	if got := protect("test", func() int { panic("boom") }); got != 1 {
		t.Errorf("protect() = %d, want 1", got)
	}
	if got := protect("test", func() int { return 0 }); got != 0 {
		t.Errorf("protect() = %d, want 0", got)
	}
	// a function that fails in the way that was a panic: a configuration
	// that is nil
	if got := protect("test", func() int { return goSendToMsTeamsChannel(nil, nil) }); got != 1 {
		t.Errorf("protect() = %d, want 1", got)
	}
}

func TestValidWebhook(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://contoso.webhook.office.com/webhookb2/abc", true},
		{"https://contoso.webhook.office365.com/webhookb2/abc", true},
		{"https://outlook.office.com/webhook/abc", true},
		{"https://outlook.office365.com/webhook/abc", true},
		{"https://CONTOSO.webhook.office.com:443/x", true},
		{"http://contoso.webhook.office.com/x", false},
		{"https://contoso.webhook.office.com.example.com/x", false},
		{"https://contoso.webhook.office.com@example.com/x", false},
		{"https://example.com/contoso.webhook.office.com", false},
		{"https://example.com?x=outlook.office.com", false},
		{"https://example.com#outlook.office.com", false},
		{"https://contoso.webhook.office.com:8443/x", false},
		{"https://notoutlook.office.com/x", false},
		{"https://outlook.office.com.evil.net/x", false},
		{"", false},
		{"not a url", false},
	}
	for _, test := range tests {
		if got := validWebhook(test.url); got != test.want {
			t.Errorf("validWebhook(%q) = %t, want %t", test.url, got, test.want)
		}
	}
}

// webhooks returns TLS servers, with the status each replies with, and
// makes them valid webhooks that are sent to using a client that trusts
// them
func webhooks(t *testing.T, statuses ...int) (urls []string, bodies func(i int) string) {
	t.Helper()
	var mu sync.Mutex
	got := make([]string, len(statuses))

	var client *http.Client
	for i, status := range statuses {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			mu.Lock()
			got[i] = string(b)
			mu.Unlock()
			w.WriteHeader(status)
		}))
		t.Cleanup(srv.Close)
		urls = append(urls, srv.URL)
		client = srv.Client()
	}

	// the servers have different certificates, so trust all of them
	client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	origValid, origClient := webhookValid, webhookClient
	webhookValid = func(string) bool { return true }
	webhookClient = func(time.Duration) *http.Client { return client }
	t.Cleanup(func() { webhookValid, webhookClient = origValid, origClient })

	return urls, func(i int) string {
		mu.Lock()
		defer mu.Unlock()
		return got[i]
	}
}

func TestSendToMsTeamsChannel(t *testing.T) {
	t.Run("no webhooks", func(t *testing.T) {
		for _, args := range [][]string{nil, {"_TO="}, {"_TO=not-a-webhook"}} {
			conf, values := parseArgs(args)
			if got := goSendToMsTeamsChannel(conf, values); got != 1 {
				t.Errorf("goSendToMsTeamsChannel(%v) = %d, want 1", args, got)
			}
		}
	})

	t.Run("every webhook gets the whole message", func(t *testing.T) {
		urls, bodies := webhooks(t, 200, 200, 200)
		conf, values := parseArgs([]string{"_TO=" + strings.Join(urls, " | "), "_SUBJECT=Subject", "_TEMPLATE_TEXT=Value {{._VALUE}}", "_TEMPLATE_TEXT_ONLY=true", "_VALUE=42"})
		if got := goSendToMsTeamsChannel(conf, values); got != 0 {
			t.Fatalf("goSendToMsTeamsChannel() = %d, want 0", got)
		}
		first := bodies(0)
		if !strings.Contains(first, `"title":"Subject"`) || !strings.Contains(first, "Value 42") {
			t.Fatalf("first body = %q", first)
		}
		for i := 1; i < len(urls); i++ {
			if bodies(i) != first {
				t.Errorf("webhook %d got %q, want %q", i, bodies(i), first)
			}
		}
	})

	t.Run("fails if none were sent", func(t *testing.T) {
		urls, _ := webhooks(t, 500, 403)
		conf, values := parseArgs([]string{"_TO=" + strings.Join(urls, "|")})
		if got := goSendToMsTeamsChannel(conf, values); got != 1 {
			t.Errorf("goSendToMsTeamsChannel() = %d, want 1", got)
		}
	})

	t.Run("works if any was sent", func(t *testing.T) {
		urls, _ := webhooks(t, 500, 200)
		conf, values := parseArgs([]string{"_TO=" + strings.Join(urls, "|")})
		if got := goSendToMsTeamsChannel(conf, values); got != 0 {
			t.Errorf("goSendToMsTeamsChannel() = %d, want 0", got)
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		urls, _ := webhooks(t, 200)
		conf, values := parseArgs([]string{"_TO=https://127.0.0.1:1/x", "_TIMEOUT=500"})
		_ = urls
		if got := goSendToMsTeamsChannel(conf, values); got != 1 {
			t.Errorf("goSendToMsTeamsChannel() = %d, want 1", got)
		}
	})

	t.Run("invalid html template does not panic", func(t *testing.T) {
		urls, _ := webhooks(t, 200)
		conf, values := parseArgs([]string{"_TO=" + urls[0], "_TEMPLATE_HTML={{"})
		if got := protect("test", func() int { return goSendToMsTeamsChannel(conf, values) }); got != 1 {
			t.Errorf("= %d, want 1", got)
		}
	})
}

func TestWebhookTimeout(t *testing.T) {
	tests := map[string]time.Duration{
		"":      DefaultMsTeamsTimeout * time.Millisecond,
		"5000":  5 * time.Second,
		"250":   250 * time.Millisecond,
		"3s":    3 * time.Second,
		"abc":   DefaultMsTeamsTimeout * time.Millisecond,
		"0":     DefaultMsTeamsTimeout * time.Millisecond,
		"-5":    DefaultMsTeamsTimeout * time.Millisecond,
		" 100 ": 100 * time.Millisecond,
	}
	for in, want := range tests {
		if got := webhookTimeout(in); got != want {
			t.Errorf("webhookTimeout(%q) = %s, want %s", in, got, want)
		}
	}
}

// the values from the data that triggered an alert must be used exactly
// as they are, which includes not being expanded
func TestSendMail(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretFile, []byte("file-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("IMS_TEST_SECRET", "env-secret")
	hostile := []string{"${env:IMS_TEST_SECRET}", "${IMS_TEST_SECRET}", "${file:" + secretFile + "}", "${config:_TO}"}

	for _, value := range hostile {
		t.Run("SendMail "+value, func(t *testing.T) {
			srv := newSMTPServer(t)
			conf, values := parseArgs(srv.args(
				"_ALERT=alert", "_ALERT_TYPE=Alert", "_GATEWAY=gw1", "_ACTION=act", "_VARIABLEPATH=/path",
				"_ALERT_FORMAT=Value is %(_VALUE) from %(_GATEWAY)", "_SUBJECT=Subject %(_VALUE)",
				"_VALUE="+value,
			))
			if got := sendMail(conf, values); got != 0 {
				t.Fatalf("sendMail() = %d, want 0", got)
			}
			messages := srv.sent()
			if len(messages) != 1 || !strings.Contains(messages[0], "Value is "+value+" from gw1") {
				t.Errorf("messages = %q", messages)
			}
			if strings.Contains(messages[0], "secret") && !strings.Contains(value, "secret") {
				t.Errorf("a value was expanded: %q", messages[0])
			}
		})

		t.Run("GoSendMail "+value, func(t *testing.T) {
			srv := newSMTPServer(t)
			conf, values := parseArgs(srv.args(
				"_TEMPLATE_TEXT_ONLY=true", "_TEMPLATE_TEXT=Value {{._VALUE}} from %(_GATEWAY)", "_SUBJECT=Subject",
				"_GATEWAY=gw1", "_VALUE="+value,
			))
			if got := goSendMail(conf, values); got != 0 {
				t.Fatalf("goSendMail() = %d, want 0", got)
			}
			messages := srv.sent()
			if len(messages) != 1 || !strings.Contains(messages[0], "Value "+value) {
				t.Errorf("messages = %q", messages)
			}
		})
	}
}

// the default templates must work, which they could not with the
// configuration as the data
func TestGoSendMailDefaultTemplates(t *testing.T) {
	srv := newSMTPServer(t)
	conf, values := parseArgs(srv.args(
		"_ALERT=alert", "_ALERT_TYPE=Alert", "_GATEWAY=gw1", "_VARIABLE=cell", "_DATAVIEW=dv", "_MANAGED_ENTITY=me",
		"_SEVERITY=CRITICAL", "_VALUE=99", "_VARIABLEPATH=/path",
	))
	if got := goSendMail(conf, values); got != 0 {
		t.Fatalf("goSendMail() = %d, want 0", got)
	}
	messages := srv.sent()
	if len(messages) != 1 || !strings.Contains(messages[0], "gw1") || !strings.Contains(messages[0], "99") {
		t.Errorf("messages = %q", messages)
	}
}

func TestGoSendMailFailures(t *testing.T) {
	srv := newSMTPServer(t)
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := map[string][]string{
		"missing html file":  srv.args("_TEMPLATE_HTML_FILE=/does/not/exist.html"),
		"empty html file":    srv.args("_TEMPLATE_HTML_FILE=" + empty),
		"missing logo":       srv.args("_TEMPLATE_LOGO_FILE=/does/not/exist.png"),
		"unparsable text":    srv.args("_TEMPLATE_TEXT={{"),
		"mismatched headers": srv.args("_TO_NAME=a,b"),
	}
	for name, args := range tests {
		conf, values := parseArgs(args)
		if got := protect(name, func() int { return goSendMail(conf, values) }); got != 1 {
			t.Errorf("%s: goSendMail() = %d, want 1", name, got)
		}
	}
	if len(srv.sent()) != 0 {
		t.Errorf("messages were sent for failures: %q", srv.sent())
	}
}
